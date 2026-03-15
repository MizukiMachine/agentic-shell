package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MizukiMachine/agentic-shell/pkg/types"
	"gopkg.in/yaml.v3"
)

const e2eInitialPrompt = "code review agent"

func TestE2ESpecGatherQuickOutputsSpecYAML(t *testing.T) {
	workdir := t.TempDir()
	configPath := writeE2EConfig(t, workdir)

	stdout, stderr, err := runCLI(
		t,
		workdir,
		e2eSpecGatherAnswers(),
		"--config", configPath,
		"spec-gather",
		"--quick",
		"--no-llm",
		"--output", "spec.yaml",
		e2eInitialPrompt,
	)
	if err != nil {
		t.Fatalf("spec-gather failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	specPath := filepath.Join(workdir, "spec.yaml")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read generated spec: %v", err)
	}

	var spec types.AgentSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("failed to decode generated spec: %v", err)
	}

	if spec.Metadata.Name != "code-review-agent" {
		t.Fatalf("expected metadata.name to be code-review-agent, got %q", spec.Metadata.Name)
	}
	if spec.Intent.Metadata.Confidence < 0.85 {
		t.Fatalf("expected confidence >= 0.85, got %.2f", spec.Intent.Metadata.Confidence)
	}
	if len(spec.Capabilities) == 0 || len(spec.Skills) == 0 || len(spec.Tools) == 0 {
		t.Fatalf("expected generated spec to include capabilities, skills, and tools: %+v", spec)
	}
	if !strings.Contains(stdout, "仕様を出力しました: spec.yaml") {
		t.Fatalf("expected output confirmation, got stdout:\n%s", stdout)
	}
	if !strings.Contains(stderr, "What is the core problem") {
		t.Fatalf("expected interactive prompt in stderr, got:\n%s", stderr)
	}
}

func TestE2EGenerateFromSpecCreatesAgentFile(t *testing.T) {
	workdir := t.TempDir()
	configPath := writeE2EConfig(t, workdir)

	if _, stderr, err := runCLI(
		t,
		workdir,
		e2eSpecGatherAnswers(),
		"--config", configPath,
		"spec-gather",
		"--quick",
		"--no-llm",
		"--output", "spec.yaml",
		e2eInitialPrompt,
	); err != nil {
		t.Fatalf("failed to prepare spec.yaml: %v\nstderr:\n%s", err, stderr)
	}

	stdout, stderr, err := runCLI(
		t,
		workdir,
		"",
		"--config", configPath,
		"generate",
		"--from", "spec.yaml",
	)
	if err != nil {
		t.Fatalf("generate failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}

	agentPath := filepath.Join(workdir, ".claude", "agents", "code-review-agent.md")
	data, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("failed to read generated agent file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "name: \"code-review-agent\"") {
		t.Fatalf("expected generated markdown frontmatter name, got:\n%s", content)
	}
	if !strings.Contains(content, "## Mission") {
		t.Fatalf("expected generated markdown body, got:\n%s", content)
	}
	if !strings.Contains(stdout, "エージェント定義を生成しました:") {
		t.Fatalf("expected generation confirmation, got stdout:\n%s", stdout)
	}
}

func TestE2EPipelineSubcommandsCanBeChained(t *testing.T) {
	workdir := t.TempDir()
	configPath := writeE2EConfig(t, workdir)

	skillDir := filepath.Join(workdir, ".claude", "skills", "code-review")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(strings.TrimSpace(`
---
name: "code-review"
description: "Review code changes and provide actionable feedback"
---

# code-review
`)), 0644); err != nil {
		t.Fatalf("failed to seed skill file: %v", err)
	}

	specPath := filepath.Join(workdir, "pipeline-spec.md")
	if err := os.WriteFile(specPath, []byte(strings.TrimSpace(`
# Review Pipeline

Build an individual CLI pipeline for code review and security analysis.

## Requirements
- Each stage must be independently executable
- Support chaining with Unix pipes
`)), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	command := strings.Join([]string{
		getBinaryPath(t) + " --config " + shellQuote(configPath) + " parse " + shellQuote(specPath),
		getBinaryPath(t) + " --config " + shellQuote(configPath) + " extract",
		getBinaryPath(t) + " --config " + shellQuote(configPath) + " skill-scan --skills-dir .claude/skills",
		getBinaryPath(t) + " --config " + shellQuote(configPath) + " match --skills-dir .claude/skills",
		getBinaryPath(t) + " --config " + shellQuote(configPath) + " skill-gen --skills-dir .claude/skills",
		getBinaryPath(t) + " --config " + shellQuote(configPath) + " output --skills-dir .claude/skills",
	}, " | ")

	cmd := exec.Command("bash", "-lc", command)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "HOME="+workdir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("pipeline execution failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	generated := filepath.Join(workdir, ".claude", "skills", "security-audit", "SKILL.md")
	data, err := os.ReadFile(generated)
	if err != nil {
		t.Fatalf("expected placeholder skill file to be created: %v", err)
	}
	if !strings.Contains(string(data), "Placeholder skill generated") {
		t.Fatalf("expected placeholder content, got:\n%s", string(data))
	}
	if !strings.Contains(stdout.String(), "\"written_files\"") {
		t.Fatalf("expected output summary JSON, got:\n%s", stdout.String())
	}
}

func writeE2EConfig(t *testing.T, dir string) string {
	t.Helper()

	configPath := filepath.Join(dir, ".ags.yaml")
	config := strings.TrimSpace(`
llm:
  claude_path: "claude"
  timeout: "2m"
  max_retries: 3
output:
  directory: "."
  format: "markdown"
  overwrite: true
gathering:
  confidence_threshold: 0.85
  max_question_rounds: 5
generation:
  default_model: "claude-sonnet-4-6"
  default_temperature: 0.4
`) + "\n"

	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write e2e config: %v", err)
	}

	return configPath
}

func runCLI(t *testing.T, workdir, stdin string, args ...string) (string, string, error) {
	t.Helper()

	cmd := exec.Command(getBinaryPath(t), args...)
	cmd.Dir = workdir
	cmd.Env = append(os.Environ(), "HOME="+workdir)

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func e2eSpecGatherAnswers() string {
	return strings.Join([]string{
		"The core problem is turning pull request diffs into implementation-ready code review guidance",
		"This matters because teams need reliable review quality across the delivery workflow",
		"Prefer quality, safety, human review, and explicit validation over raw speed",
		"The ideal solution is an interactive Go CLI that captures requirements and exports validated YAML for a code review agent",
		"It supports broader objectives like reusable agent definitions and repeatable automation",
	}, "\n") + "\n"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
