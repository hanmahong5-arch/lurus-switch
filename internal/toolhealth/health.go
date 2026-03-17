package toolhealth

import (
	"encoding/json"
	"fmt"
	"strings"

	"lurus-switch/internal/toolconfig"

	"github.com/BurntSushi/toml"
)

// HealthStatus represents the health level of a tool's configuration
type HealthStatus string

const (
	StatusGreen  HealthStatus = "green"
	StatusYellow HealthStatus = "yellow"
	StatusRed    HealthStatus = "red"
)

// HealthResult contains the health check result for a single tool
type HealthResult struct {
	Tool   string       `json:"tool"`
	Status HealthStatus `json:"status"`
	Issues []string     `json:"issues"`
}

// supportedTools lists all known tool names
var supportedTools = []string{"claude", "codex", "gemini", "picoclaw", "nullclaw"}

// CheckTool performs a health check on a single tool's configuration
func CheckTool(tool string) *HealthResult {
	result := &HealthResult{
		Tool:   tool,
		Status: StatusGreen,
		Issues: []string{},
	}

	info, err := toolconfig.ReadConfig(tool)
	if err != nil {
		result.Status = StatusRed
		result.Issues = append(result.Issues, "config read error: "+err.Error())
		return result
	}

	if !info.Exists {
		result.Status = StatusRed
		result.Issues = append(result.Issues, "config file not found")
		return result
	}

	switch tool {
	case "claude":
		checkClaudeHealth(info.Content, result)
	case "codex":
		checkCodexHealth(info.Content, result)
	case "gemini":
		checkGeminiHealth(info.Content, result)
	case "picoclaw":
		checkPicoClawHealth(info.Content, result)
	case "nullclaw":
		checkNullClawHealth(info.Content, result)
	}

	return result
}

// CheckAll performs health checks on all known tools
func CheckAll() map[string]*HealthResult {
	results := make(map[string]*HealthResult)
	for _, tool := range supportedTools {
		results[tool] = CheckTool(tool)
	}
	return results
}

// checkClaudeHealth validates Claude config JSON
func checkClaudeHealth(content string, r *HealthResult) {
	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		r.Status = StatusRed
		r.Issues = append(r.Issues, "invalid JSON format")
		return
	}

	env, _ := data["env"].(map[string]any)
	apiKey, _ := env["ANTHROPIC_API_KEY"].(string)
	baseURL, _ := env["ANTHROPIC_BASE_URL"].(string)

	if apiKey == "" && baseURL == "" {
		r.Status = StatusYellow
		r.Issues = append(r.Issues, "ANTHROPIC_API_KEY and ANTHROPIC_BASE_URL both empty")
	}
}

// checkCodexHealth validates Codex config TOML
func checkCodexHealth(content string, r *HealthResult) {
	var data map[string]any
	if _, err := toml.Decode(content, &data); err != nil {
		r.Status = StatusRed
		r.Issues = append(r.Issues, "invalid TOML format")
		return
	}

	model, _ := data["model"].(string)
	if model == "" {
		r.Status = StatusYellow
		r.Issues = append(r.Issues, "model is empty")
	}
}

// checkGeminiHealth validates Gemini config JSON
func checkGeminiHealth(content string, r *HealthResult) {
	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		r.Status = StatusRed
		r.Issues = append(r.Issues, "invalid JSON format")
		return
	}

	modelObj, _ := data["model"].(map[string]any)
	modelName, _ := modelObj["name"].(string)
	if modelName == "" {
		r.Status = StatusYellow
		r.Issues = append(r.Issues, "model.name is missing")
	}
}

// checkPicoClawHealth validates PicoClaw config JSON
func checkPicoClawHealth(content string, r *HealthResult) {
	checkClawHealth(content, r)
}

// checkNullClawHealth validates NullClaw config JSON
func checkNullClawHealth(content string, r *HealthResult) {
	checkClawHealth(content, r)
}

// checkClawHealth is shared logic for PicoClaw/NullClaw configs
func checkClawHealth(content string, r *HealthResult) {
	var data map[string]any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		r.Status = StatusRed
		r.Issues = append(r.Issues, "invalid JSON format")
		return
	}

	modelList, _ := data["model_list"].([]any)
	if len(modelList) == 0 {
		r.Status = StatusYellow
		r.Issues = append(r.Issues, "no models configured")
		return
	}

	for i, entry := range modelList {
		item, _ := entry.(map[string]any)
		apiBase, _ := item["api_base"].(string)
		modelName, _ := item["model_name"].(string)
		if strings.TrimSpace(apiBase) == "" {
			r.Status = StatusYellow
			r.Issues = append(r.Issues, fmt.Sprintf("model_list[%d].api_base is empty", i))
		}
		if strings.TrimSpace(modelName) == "" {
			r.Status = StatusYellow
			r.Issues = append(r.Issues, fmt.Sprintf("model_list[%d].model_name is empty", i))
		}
	}
}
