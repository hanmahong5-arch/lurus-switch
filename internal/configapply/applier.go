package configapply

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Applier runs the 5-phase apply: validate → snapshot → write → verify → done.
// On any failure it restores all touched files from the pre-write snapshot and
// returns an ApplyResult with WhatHappened / WhatExpected / NextSteps populated
// via the Explainer chain.
//
// The applier never returns a Go error from Apply: callers (Wails bindings)
// always get an *ApplyResult and must inspect .Success. This eliminates the
// "Result.success silent failure" class of bug recorded in feedback_wails_result_success.
type Applier struct {
	store      *Store
	explainers []Explainer
}

func NewApplier(store *Store, explainers ...Explainer) *Applier {
	if store == nil {
		return nil
	}
	if len(explainers) == 0 {
		explainers = DefaultExplainers()
	}
	return &Applier{store: store, explainers: explainers}
}

func (a *Applier) Apply(plan *ChangePlan) *ApplyResult {
	result := &ApplyResult{
		PlanID:    safeID(plan),
		Phase:     PhasePending,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if plan == nil {
		return a.failResult(result, fmt.Errorf("nil plan"), nil)
	}

	result.Phase = PhaseValidate
	if err := validatePlan(plan); err != nil {
		return a.failResult(result, err, plan)
	}

	if plan.IsEmpty() {
		result.Phase = PhaseDone
		result.Success = true
		result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		result.WhatHappened = "无改动:plan 的 before/after 等价。"
		return result
	}

	result.Phase = PhaseSnapshot
	tx, err := a.store.BeginTransaction(plan)
	if err != nil {
		return a.failResult(result, err, plan)
	}

	result.Phase = PhaseWrite
	written := []string{}
	for _, ch := range plan.Changes {
		if err := writeChange(ch); err != nil {
			rolled := rollback(written, tx)
			result.FilesRolled = rolled
			result.RollbackDone = len(rolled) == len(written)
			result.RollbackNote = fmt.Sprintf("回滚 %d/%d 个文件", len(rolled), len(written))
			_ = a.store.CompleteTransaction(tx, false)
			return a.failResult(result, err, plan)
		}
		written = append(written, ch.Path)
	}

	result.Phase = PhaseVerify
	if err := verifyChanges(plan.Changes); err != nil {
		rolled := rollback(written, tx)
		result.FilesRolled = rolled
		result.RollbackDone = len(rolled) == len(written)
		result.RollbackNote = fmt.Sprintf("校验失败,回滚 %d/%d 个文件", len(rolled), len(written))
		_ = a.store.CompleteTransaction(tx, false)
		return a.failResult(result, err, plan)
	}

	result.Phase = PhaseDone
	result.Success = true
	result.FilesWritten = written
	result.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	_ = a.store.CompleteTransaction(tx, true)
	return result
}

func (a *Applier) failResult(r *ApplyResult, err error, plan *ChangePlan) *ApplyResult {
	r.Success = false
	r.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	r.RawError = err.Error()
	r.WhatHappened, r.WhatExpected, r.NextSteps = Explain(a.explainers, err, plan)
	return r
}

func safeID(plan *ChangePlan) string {
	if plan == nil {
		return ""
	}
	return plan.ID
}

func validatePlan(plan *ChangePlan) error {
	if plan.ID == "" {
		return fmt.Errorf("plan has empty ID")
	}
	for i, ch := range plan.Changes {
		if !filepath.IsAbs(ch.Path) {
			return fmt.Errorf("change[%d]: path not absolute: %s", i, ch.Path)
		}
		if ch.Kind == "" {
			return fmt.Errorf("change[%d]: kind is empty", i)
		}
	}
	return nil
}

func writeChange(ch FileChange) error {
	mode := os.FileMode(ch.Mode)
	if mode == 0 {
		mode = 0644
	}
	switch ch.Kind {
	case KindCreate, KindUpdate:
		return WriteAtomic(ch.Path, []byte(ch.After), mode)
	case KindDelete:
		err := os.Remove(ch.Path)
		if err != nil && os.IsNotExist(err) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unknown change kind: %s", ch.Kind)
	}
}

func verifyChanges(changes []FileChange) error {
	for _, ch := range changes {
		if ch.Kind == KindDelete {
			if _, err := os.Stat(ch.Path); !os.IsNotExist(err) {
				return fmt.Errorf("verify: delete failed for %s", ch.Path)
			}
			continue
		}
		ok, err := FileSizeMatches(ch.Path, len(ch.After))
		if err != nil {
			return fmt.Errorf("verify stat %s: %w", ch.Path, err)
		}
		if !ok {
			return fmt.Errorf("verify size mismatch: %s", ch.Path)
		}
	}
	return nil
}

// rollback restores up to len(written) files from tx pre-state. Returns the
// list of files actually restored (best-effort; missing pre-state means we
// just delete the file since it didn't exist before).
func rollback(written []string, tx *Transaction) []string {
	var rolled []string
	for _, path := range written {
		preContent, hadContent := tx.PreContents[path]
		existed := tx.PreExisted[path]
		if !existed {
			if err := os.Remove(path); err == nil || os.IsNotExist(err) {
				rolled = append(rolled, path)
			}
			continue
		}
		if hadContent {
			if err := WriteAtomic(path, []byte(preContent), 0644); err == nil {
				rolled = append(rolled, path)
			}
		}
	}
	return rolled
}
