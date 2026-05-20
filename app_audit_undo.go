package main

import (
	"encoding/json"
	"fmt"

	"lurus-switch/internal/audit"
	"lurus-switch/internal/hub/admin"
)

// registerAuditUndoHandlers wires inverse operations for state-mutating
// bindings into the audit journal so the admin Undo button works.
//
// Each handler receives a copy of the entry containing the Before/After
// snapshots captured at write-time. Reversibility constraints:
//
//   - update: Before is the prior entity; we PUT it back as a patch.
//   - delete: Before is the prior entity; we POST a fresh create.
//             Note that Hub allocates a new ID on re-create — the
//             revived row won't have the same ID as the deleted one.
//             The admin UI surfaces this in the undo confirmation.
//   - delete_batch: Before is a slice of prior entities, re-created
//             one by one. Partial failures are surfaced; the journal
//             records the undo as ok only if all succeeded.
//   - model.switch: Before is the prior model name; we re-apply it.
//
// Create ops are intentionally NOT registered: AddChannel / AddToken
// don't return the new ID, so we can't reliably target a delete to
// undo them. Marking them reversible would require an extra round-trip
// (search by name, hope for uniqueness) — punt until the admin client
// is updated to surface the created ID.
//
// All handlers acquire a fresh admin.Client per invocation, matching
// the stateless pattern used by the bindings.
func (a *App) registerAuditUndoHandlers() {
	if a == nil || a.auditJournal == nil {
		return
	}

	// -- channel ----------------------------------------------------------

	a.auditJournal.Register(auditOpChannelUpdate, func(e audit.Entry) error {
		prior, ok := decodeMap(e.Before)
		if !ok {
			return fmt.Errorf("undo channel.update: missing Before snapshot")
		}
		c, err := hubClient()
		if err != nil {
			return err
		}
		return c.UpdateChannel(a.hubCtx(), prior)
	})

	a.auditJournal.Register(auditOpChannelDelete, func(e audit.Entry) error {
		prior, ok := decodeMap(e.Before)
		if !ok {
			return fmt.Errorf("undo channel.delete: missing Before snapshot")
		}
		c, err := hubClient()
		if err != nil {
			return err
		}
		// Drop server-assigned fields so Hub re-allocates them on the
		// recreated row. Leaving the old id around makes Hub error.
		delete(prior, "id")
		delete(prior, "created_time")
		return c.AddChannel(a.hubCtx(), admin.CreateChannelInput(prior))
	})

	a.auditJournal.Register(auditOpChannelDeleteBatch, func(e audit.Entry) error {
		priors, ok := decodeMapSlice(e.Before)
		if !ok || len(priors) == 0 {
			return fmt.Errorf("undo channel.delete_batch: missing Before snapshots")
		}
		c, err := hubClient()
		if err != nil {
			return err
		}
		for _, p := range priors {
			delete(p, "id")
			delete(p, "created_time")
			if err := c.AddChannel(a.hubCtx(), admin.CreateChannelInput(p)); err != nil {
				return fmt.Errorf("re-create channel %q: %w", p["name"], err)
			}
		}
		return nil
	})

	// -- token ------------------------------------------------------------

	a.auditJournal.Register(auditOpTokenUpdate, func(e audit.Entry) error {
		prior, ok := decodeMap(e.Before)
		if !ok {
			return fmt.Errorf("undo token.update: missing Before snapshot")
		}
		c, err := hubClient()
		if err != nil {
			return err
		}
		return c.UpdateToken(a.hubCtx(), prior)
	})

	a.auditJournal.Register(auditOpTokenDelete, func(e audit.Entry) error {
		prior, ok := decodeMap(e.Before)
		if !ok {
			return fmt.Errorf("undo token.delete: missing Before snapshot")
		}
		c, err := hubClient()
		if err != nil {
			return err
		}
		delete(prior, "id")
		delete(prior, "created_time")
		delete(prior, "accessed_time")
		delete(prior, "key") // Hub regenerates the secret
		return c.AddToken(a.hubCtx(), admin.CreateTokenInput(prior))
	})

	a.auditJournal.Register(auditOpTokenDeleteBatch, func(e audit.Entry) error {
		priors, ok := decodeMapSlice(e.Before)
		if !ok || len(priors) == 0 {
			return fmt.Errorf("undo token.delete_batch: missing Before snapshots")
		}
		c, err := hubClient()
		if err != nil {
			return err
		}
		for _, p := range priors {
			delete(p, "id")
			delete(p, "created_time")
			delete(p, "accessed_time")
			delete(p, "key")
			if err := c.AddToken(a.hubCtx(), admin.CreateTokenInput(p)); err != nil {
				return fmt.Errorf("re-create token %q: %w", p["name"], err)
			}
		}
		return nil
	})

	// -- redemption -------------------------------------------------------

	a.auditJournal.Register(auditOpRedemptionDelete, func(e audit.Entry) error {
		prior, ok := decodeMap(e.Before)
		if !ok {
			return fmt.Errorf("undo redemption.delete: missing Before snapshot")
		}
		c, err := hubClient()
		if err != nil {
			return err
		}
		var quota int64
		switch q := prior["quota"].(type) {
		case float64:
			quota = int64(q)
		case int64:
			quota = q
		case int:
			quota = int64(q)
		}
		var expiredTime int64
		switch t := prior["expired_time"].(type) {
		case float64:
			expiredTime = int64(t)
		case int64:
			expiredTime = t
		case int:
			expiredTime = int64(t)
		}
		name, _ := prior["name"].(string)
		_, err = c.CreateRedemptions(a.hubCtx(), admin.CreateRedemptionInput{
			Name:        name,
			Quota:       quota,
			Count:       1,
			ExpiredTime: expiredTime,
		})
		return err
	})

	// -- model.switch -----------------------------------------------------

	a.auditJournal.Register(auditOpModelSwitch, func(e audit.Entry) error {
		prior, ok := decodeMap(e.Before)
		if !ok {
			return fmt.Errorf("undo model.switch: missing Before snapshot")
		}
		prevModel, _ := prior["model"].(string)
		if prevModel == "" {
			return fmt.Errorf("undo model.switch: prior model name is empty")
		}
		// Re-apply the prior model. The audit journal will record the
		// re-apply as its own entry — that's the audit trail for the
		// undo, in addition to the synthetic "audit.undo" marker.
		result := a.SwitchModel(prevModel)
		if errMsg, has := result["error"]; has && errMsg != "" {
			return fmt.Errorf("undo model.switch: %s", errMsg)
		}
		return nil
	})
}

// decodeMap is a tolerant Before-snapshot decoder. The journal stores
// Before as `any` — in-memory it might be the original *Channel pointer,
// while after a hot-restart it comes back as a map[string]any from JSON.
// Re-marshalling normalises both shapes.
func decodeMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	if m, ok := v.(map[string]any); ok {
		// Return a shallow copy so the handler can mutate without
		// touching the journal's stored entry.
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[k] = val
		}
		return out, true
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, false
	}
	return m, true
}

// decodeMapSlice is decodeMap's variant for batch entries.
func decodeMapSlice(v any) ([]map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	if arr, ok := v.([]map[string]any); ok {
		out := make([]map[string]any, len(arr))
		for i, m := range arr {
			cp := make(map[string]any, len(m))
			for k, val := range m {
				cp[k] = val
			}
			out[i] = cp
		}
		return out, true
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var arr []map[string]any
	if err := json.Unmarshal(b, &arr); err != nil {
		return nil, false
	}
	return arr, true
}
