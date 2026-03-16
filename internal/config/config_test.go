package config

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LLM.Provider != "glm" {
		t.Fatalf("expected default provider glm, got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.BaseURL != "https://open.bigmodel.cn/api/paas/v4/" {
		t.Fatalf("expected default base_url, got %q", cfg.LLM.BaseURL)
	}
	if cfg.LLM.Model != "glm-4-flash" {
		t.Fatalf("expected default model glm-4-flash, got %q", cfg.LLM.Model)
	}
	if cfg.Output.Directory != ".claude/agents" {
		t.Fatalf("expected default output directory, got %q", cfg.Output.Directory)
	}
	if cfg.Gathering.ConfidenceThreshold != 0.85 {
		t.Fatalf("expected default confidence threshold 0.85, got %v", cfg.Gathering.ConfidenceThreshold)
	}
	if !cfg.Gathering.UseLLMQuestions {
		t.Fatal("expected default use_llm_questions=true")
	}
	if cfg.Generation.DefaultTemperature != 0.7 {
		t.Fatalf("expected default temperature 0.7, got %v", cfg.Generation.DefaultTemperature)
	}
}

func TestLoaderLoadFromEnv(t *testing.T) {
	t.Setenv("AGENTIC_LLM_PROVIDER", "glm")
	t.Setenv("AGENTIC_LLM_BASE_URL", "https://glm.example.test/v1/")
	t.Setenv("AGENTIC_LLM_MODEL", "glm-4.5")
	t.Setenv("AGENTIC_LLM_MAX_RETRIES", "0")
	t.Setenv("AGENTIC_GATHERING_CONFIDENCE_THRESHOLD", "0")
	t.Setenv("AGENTIC_GATHERING_USE_LLM_QUESTIONS", "true")
	t.Setenv("AGENTIC_GENERATION_DEFAULT_TEMPERATURE", "0")
	t.Setenv("AGENTIC_OUTPUT_OVERWRITE", "true")

	cfg, err := NewLoader().WithConfigName("definitely-missing-config-for-env-test").Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.LLM.Provider != "glm" {
		t.Fatalf("expected env override for llm.provider, got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.BaseURL != "https://glm.example.test/v1/" {
		t.Fatalf("expected env override for llm.base_url, got %q", cfg.LLM.BaseURL)
	}
	if cfg.LLM.Model != "glm-4.5" {
		t.Fatalf("expected env override for llm.model, got %q", cfg.LLM.Model)
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
  provider: glm
  base_url: https://open.bigmodel.cn/api/paas/v4/custom/
  model: glm-4-plus
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
	if cfg.LLM.Provider != "glm" {
		t.Fatalf("expected llm.provider from file, got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.BaseURL != "https://open.bigmodel.cn/api/paas/v4/custom/" {
		t.Fatalf("expected llm.base_url from file, got %q", cfg.LLM.BaseURL)
	}
	if cfg.LLM.Model != "glm-4-plus" {
		t.Fatalf("expected llm.model from file, got %q", cfg.LLM.Model)
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

func TestLoaderIgnoresDeprecatedClaudePathWithWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agentic.yaml")
	content := []byte(`llm:
  claude_path: /opt/claude
`)

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	var logBuffer bytes.Buffer
	restore := captureLogs(t, &logBuffer)
	defer restore()

	cfg, err := NewLoader().WithConfigPath(path).Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.LLM.Provider != "glm" {
		t.Fatalf("expected default provider to remain glm, got %q", cfg.LLM.Provider)
	}
	if !strings.Contains(logBuffer.String(), "llm.claude_path is deprecated and ignored") {
		t.Fatalf("expected deprecation warning, got %q", logBuffer.String())
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
			Model:      stringPtr("glm-4-air"),
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
	if cfg.LLM.Model != "glm-4-air" {
		t.Fatalf("expected llm.model=glm-4-air, got %q", cfg.LLM.Model)
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

func captureLogs(t *testing.T, dst *bytes.Buffer) func() {
	t.Helper()

	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(dst)
	log.SetFlags(0)

	return func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	}
}
