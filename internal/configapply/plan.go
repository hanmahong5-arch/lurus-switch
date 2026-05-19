package configapply

type ChangeKind string

const (
	KindCreate ChangeKind = "create"
	KindUpdate ChangeKind = "update"
	KindDelete ChangeKind = "delete"
)

type FileChange struct {
	Path        string     `json:"path"`
	Kind        ChangeKind `json:"kind"`
	Before      string     `json:"before,omitempty"`
	After       string     `json:"after,omitempty"`
	Mode        uint32     `json:"mode"`
	DiffSummary string     `json:"diffSummary"`
	UnifiedDiff string     `json:"unifiedDiff,omitempty"`
}

type ChangePlan struct {
	ID          string       `json:"id"`
	Intent      string       `json:"intent"`
	Description string       `json:"description"`
	CreatedAt   string       `json:"createdAt"`
	Changes     []FileChange `json:"changes"`
	SideEffects []string     `json:"sideEffects,omitempty"`
}

func (p *ChangePlan) IsEmpty() bool {
	if p == nil || len(p.Changes) == 0 {
		return true
	}
	for _, ch := range p.Changes {
		if ch.Before != ch.After || ch.Kind != KindUpdate {
			return false
		}
	}
	return true
}

func (p *ChangePlan) TotalAdditions() int {
	if p == nil {
		return 0
	}
	var n int
	for _, ch := range p.Changes {
		n += countLines(ch.After) - sharedPrefix(ch.Before, ch.After)
	}
	return n
}
