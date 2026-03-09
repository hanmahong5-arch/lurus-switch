package promptlib

// Prompt represents a reusable system prompt or instruction snippet
type Prompt struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`    // "coding" | "writing" | "analysis" | "custom"
	Tags        []string `json:"tags"`
	Content     string   `json:"content"`
	TargetTools []string `json:"targetTools"` // ["claude", "all", ...]
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}
