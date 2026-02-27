package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"lurus-switch/internal/config"
	"lurus-switch/internal/installer"
)

func TestNewPicoClawGenerator(t *testing.T) {
	gen := NewPicoClawGenerator()
	if gen == nil {
		t.Fatal("NewPicoClawGenerator should return non-nil")
	}
}

// === GenerateString Tests ===

func TestPicoClawGenerator_GenerateString_DefaultConfig(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := config.NewPicoClawConfig()

	result, err := gen.GenerateString(cfg)
	if err != nil {
		t.Fatalf("GenerateString error: %v", err)
	}

	if result == "" {
		t.Error("result should not be empty")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify model_list exists
	modelList, ok := parsed["model_list"].([]interface{})
	if !ok {
		t.Fatal("missing model_list in output")
	}
	if len(modelList) != 1 {
		t.Errorf("model_list length = %d, want 1", len(modelList))
	}

	// Verify agents section
	agents, ok := parsed["agents"].(map[string]interface{})
	if !ok {
		t.Fatal("missing agents in output")
	}
	defaults, ok := agents["defaults"].(map[string]interface{})
	if !ok {
		t.Fatal("missing agents.defaults in output")
	}
	if defaults["model_name"] != installer.DefaultPicoClawModel {
		t.Errorf("model_name = %v, want %s", defaults["model_name"], installer.DefaultPicoClawModel)
	}
}

func TestPicoClawGenerator_GenerateString_CustomConfig(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "proxy-a", APIBase: "https://a.com", APIKey: "key-a", ModelName: "model-a"},
			{Name: "proxy-b", APIBase: "https://b.com", APIKey: "key-b", ModelName: "model-b"},
		},
		Agents: config.PicoClawAgentSettings{
			Defaults: config.PicoClawAgentDefaults{ModelName: "proxy-a"},
		},
	}

	result, err := gen.GenerateString(cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	var parsed config.PicoClawConfig
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed.ModelList) != 2 {
		t.Errorf("model_list length = %d", len(parsed.ModelList))
	}
	if parsed.ModelList[0].APIBase != "https://a.com" {
		t.Errorf("first model APIBase = %q", parsed.ModelList[0].APIBase)
	}
	if parsed.Agents.Defaults.ModelName != "proxy-a" {
		t.Errorf("agent default = %q", parsed.Agents.Defaults.ModelName)
	}
}

func TestPicoClawGenerator_GenerateString_EmptyModelList(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{},
	}

	result, err := gen.GenerateString(cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// Should still produce valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

// === Generate (file output) Tests ===

func TestPicoClawGenerator_Generate_CreatesFile(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := config.NewPicoClawConfig()
	tmpDir := t.TempDir()

	outputPath, err := gen.Generate(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "config.json")
	if outputPath != expectedPath {
		t.Errorf("outputPath = %q, want %q", outputPath, expectedPath)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	var parsed config.PicoClawConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(parsed.ModelList) != 1 {
		t.Errorf("model_list length = %d", len(parsed.ModelList))
	}
}

func TestPicoClawGenerator_Generate_CreatesDirectory(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := config.NewPicoClawConfig()
	tmpDir := filepath.Join(t.TempDir(), "nested", "dir")

	_, err := gen.Generate(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	stat, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("dir not created: %v", err)
	}
	if !stat.IsDir() {
		t.Error("expected directory")
	}
}

func TestPicoClawGenerator_Generate_Overwrite(t *testing.T) {
	gen := NewPicoClawGenerator()
	tmpDir := t.TempDir()

	// Write initial
	cfg1 := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "first", ModelName: "m1"},
		},
	}
	gen.Generate(cfg1, tmpDir)

	// Overwrite
	cfg2 := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "second", ModelName: "m2"},
			{Name: "third", ModelName: "m3"},
		},
	}
	gen.Generate(cfg2, tmpDir)

	data, _ := os.ReadFile(filepath.Join(tmpDir, "config.json"))
	var parsed config.PicoClawConfig
	json.Unmarshal(data, &parsed)

	if len(parsed.ModelList) != 2 {
		t.Errorf("overwrite should have 2 models, got %d", len(parsed.ModelList))
	}
	if parsed.ModelList[0].Name != "second" {
		t.Errorf("first model = %q, want second", parsed.ModelList[0].Name)
	}
}

// === Validate Tests ===

func TestPicoClawGenerator_Validate_ValidConfig(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := config.NewPicoClawConfig()

	err := gen.Validate(cfg)
	if err != nil {
		t.Errorf("valid config should not error: %v", err)
	}
}

func TestPicoClawGenerator_Validate_EmptyModelList(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{},
	}

	err := gen.Validate(cfg)
	if err == nil {
		t.Error("empty model_list should fail validation")
	}
}

func TestPicoClawGenerator_Validate_MissingModelName(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: ""},
		},
	}

	err := gen.Validate(cfg)
	if err == nil {
		t.Error("missing model name should fail validation")
	}
}

func TestPicoClawGenerator_Validate_MultipleModels(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "a", ModelName: "m1"},
			{Name: "b", ModelName: "m2"},
		},
	}

	err := gen.Validate(cfg)
	if err != nil {
		t.Errorf("multiple valid models should pass: %v", err)
	}
}

func TestPicoClawGenerator_Validate_SecondModelMissingName(t *testing.T) {
	gen := NewPicoClawGenerator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "ok"},
			{Name: ""},
		},
	}

	err := gen.Validate(cfg)
	if err == nil {
		t.Error("second model missing name should fail")
	}
}

// === JSON Roundtrip Tests ===

func TestPicoClawConfig_JSONRoundTrip(t *testing.T) {
	original := config.NewPicoClawConfig()
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded config.PicoClawConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.ModelList) != len(original.ModelList) {
		t.Errorf("ModelList length mismatch")
	}
	if decoded.Agents.Defaults.ModelName != original.Agents.Defaults.ModelName {
		t.Errorf("ModelName = %q, want %q", decoded.Agents.Defaults.ModelName, original.Agents.Defaults.ModelName)
	}
}
