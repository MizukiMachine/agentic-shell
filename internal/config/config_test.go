package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LLM.ClaudePath != "claude" {
		t.Fatalf("expected default Claude path, got %q", cfg.LLM.ClaudePath)
	}
	if cfg.Output.Directory != ".claude/agents" {
		t.Fatalf("expected default output directory, got %q", cfg.Output.Directory)
	}
	if cfg.Gathering.ConfidenceThreshold != 0.85 {
		t.Fatalf("expected default confidence threshold 0.85, got %v", cfg.Gathering.ConfidenceThreshold)
	}
	if cfg.Gathering.UseLLMQuestions {
		t.Fatal("expected default use_llm_questions=false")
	}
	if cfg.Generation.DefaultTemperature != 0.7 {
		t.Fatalf("expected default temperature 0.7, got %v", cfg.Generation.DefaultTemperature)
	}
}

func TestLoaderLoadFromEnv(t *testing.T) {
	t.Setenv("AGENTIC_LLM_CLAUDE_PATH", "/usr/local/bin/claude")
	t.Setenv("AGENTIC_LLM_MAX_RETRIES", "0")
	t.Setenv("AGENTIC_GATHERING_CONFIDENCE_THRESHOLD", "0")
	t.Setenv("AGENTIC_GATHERING_USE_LLM_QUESTIONS", "true")
	t.Setenv("AGENTIC_GENERATION_DEFAULT_TEMPERATURE", "0")
	t.Setenv("AGENTIC_OUTPUT_OVERWRITE", "true")

	cfg, err := NewLoader().WithConfigName("definitely-missing-config-for-env-test").Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.LLM.ClaudePath != "/usr/local/bin/claude" {
		t.Fatalf("expected env override for llm.claude_path, got %q", cfg.LLM.ClaudePath)
	}
	if cfg.LLM.MaxRetries != 0 {
		t.Fatalf("expected env override for llm.max_retries=0, got %d", cfg.LLM.MaxRetries)
	}
	if cfg.Gathering.ConfidenceThreshold != 0 {
		t.Fatalf("expected env override for gathering.confidence_threshold=0, got %v", cfg.Gathering.ConfidenceThreshold)
	}
	if !cfg.Gathering.UseLLMQuestions {
		t.Fatal("expected env override for gathering.use_llm_questions=true")
	}
	if cfg.Generation.DefaultTemperature != 0 {
		t.Fatalf("expected env override for generation.default_temperature=0, got %v", cfg.Generation.DefaultTemperature)
	}
	if !cfg.Output.Overwrite {
		t.Fatal("expected env override for output.overwrite=true")
	}
}

func TestLoaderLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agentic.yaml")
	content := []byte(`llm:
  claude_path: /opt/claude
output:
  directory: ./out
  overwrite: true
gathering:
  confidence_threshold: 0.9
  max_question_rounds: 3
  use_llm_questions: true
generation:
  default_model: claude-opus-4-1
  default_temperature: 0.2
`)

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loader := NewLoader().WithConfigPath(path)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if loader.ConfigFileUsed() != path {
		t.Fatalf("expected ConfigFileUsed %q, got %q", path, loader.ConfigFileUsed())
	}
	if cfg.Output.Directory != "./out" {
		t.Fatalf("expected output.directory from file, got %q", cfg.Output.Directory)
	}
	if !cfg.Output.Overwrite {
		t.Fatal("expected overwrite=true from file")
	}
	if cfg.Gathering.MaxQuestionRounds != 3 {
		t.Fatalf("expected max_question_rounds=3, got %d", cfg.Gathering.MaxQuestionRounds)
	}
	if !cfg.Gathering.UseLLMQuestions {
		t.Fatal("expected use_llm_questions=true from file")
	}
	if cfg.Generation.DefaultModel != "claude-opus-4-1" {
		t.Fatalf("expected default_model from file, got %q", cfg.Generation.DefaultModel)
	}
}

func TestLoaderLoadReturnsErrorForInvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.yaml")

	if err := os.WriteFile(path, []byte("llm: [broken"), 0644); err != nil {
		t.Fatalf("failed to write broken config: %v", err)
	}

	_, err := NewLoader().WithConfigPath(path).Load()
	if err == nil {
		t.Fatal("expected invalid config file error")
	}
}

func TestConfigMergePreservesZeroValues(t *testing.T) {
	cfg := DefaultConfig()

	cfg.Merge(&ConfigOverrides{
		LLM: LLMConfigOverrides{
			MaxRetries: intPtr(0),
		},
		Output: OutputConfigOverrides{
			Overwrite: boolPtr(true),
		},
		Gathering: GatheringConfigOverrides{
			ConfidenceThreshold: float64Ptr(0),
			MaxQuestionRounds:   intPtr(1),
			UseLLMQuestions:     boolPtr(true),
		},
		Generation: GenerationConfigOverrides{
			DefaultTemperature: float64Ptr(0),
		},
	})

	if cfg.LLM.MaxRetries != 0 {
		t.Fatalf("expected llm.max_retries=0, got %d", cfg.LLM.MaxRetries)
	}
	if !cfg.Output.Overwrite {
		t.Fatal("expected output.overwrite=true")
	}
	if cfg.Gathering.ConfidenceThreshold != 0 {
		t.Fatalf("expected gathering.confidence_threshold=0, got %v", cfg.Gathering.ConfidenceThreshold)
	}
	if cfg.Gathering.MaxQuestionRounds != 1 {
		t.Fatalf("expected gathering.max_question_rounds=1, got %d", cfg.Gathering.MaxQuestionRounds)
	}
	if !cfg.Gathering.UseLLMQuestions {
		t.Fatal("expected gathering.use_llm_questions=true")
	}
	if cfg.Generation.DefaultTemperature != 0 {
		t.Fatalf("expected generation.default_temperature=0, got %v", cfg.Generation.DefaultTemperature)
	}
}

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}
