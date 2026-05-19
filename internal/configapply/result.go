package configapply

type ApplyPhase string

const (
	PhasePending  ApplyPhase = "pending"
	PhaseValidate ApplyPhase = "validate"
	PhaseSnapshot ApplyPhase = "snapshot"
	PhaseWrite    ApplyPhase = "write"
	PhaseVerify   ApplyPhase = "verify"
	PhaseDone     ApplyPhase = "done"
)

type NextStep struct {
	Label  string            `json:"label"`
	Action string            `json:"action"`
	URL    string            `json:"url,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

type ApplyResult struct {
	PlanID     string     `json:"planID"`
	Success    bool       `json:"success"`
	Phase      ApplyPhase `json:"phase"`
	StartedAt  string     `json:"startedAt"`
	FinishedAt string     `json:"finishedAt,omitempty"`

	WhatHappened string     `json:"whatHappened,omitempty"`
	WhatExpected string     `json:"whatExpected,omitempty"`
	RollbackDone bool       `json:"rollbackDone"`
	RollbackNote string     `json:"rollbackNote,omitempty"`
	NextSteps    []NextStep `json:"nextSteps,omitempty"`

	FilesWritten []string `json:"filesWritten,omitempty"`
	FilesRolled  []string `json:"filesRolled,omitempty"`
	RawError     string   `json:"rawError,omitempty"`
}

func (r *ApplyResult) Failed() bool {
	return r != nil && !r.Success
}
