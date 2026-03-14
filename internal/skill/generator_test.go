package skill

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	types "github.com/MizukiMachine/agentic-shell/pkg/types"
)

func TestSkillGeneratorAnalyzeSkipsExistingSkills(t *testing.T) {
	dir := t.TempDir()
	existingPath := filepath.Join(dir, "code-review", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(existingPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(existingPath, []byte(strings.TrimSpace(`
---
name: Code Review
category: development
description: Review code changes carefully.
tags:
  - review
---
`)), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	generator := NewSkillGenerator(skillGeneratorSpec(), SkillGeneratorConfig{
		OutputDir: dir,
		Auto:      true,
	})

	plan, err := generator.Analyze()
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if len(plan.ExistingSkills) != 1 {
		t.Fatalf("expected 1 existing skill, got %d", len(plan.ExistingSkills))
	}
	if containsPlannedSkill(plan.MissingSkills, "Code Review") {
		t.Fatalf("expected existing Code Review skill to be skipped, got %+v", plan.MissingSkills)
	}
	if !containsPlannedSkill(plan.MissingSkills, "Security Audit") {
		t.Fatalf("expected Security Audit to be generated, got %+v", plan.MissingSkills)
	}
}

func TestSkillGeneratorGenerateWritesSkillFiles(t *testing.T) {
	dir := t.TempDir()
	var output bytes.Buffer

	generator := NewSkillGenerator(skillGeneratorSpec(), SkillGeneratorConfig{
		OutputDir: dir,
		Auto:      true,
		Output:    &output,
	})

	plan, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(plan.WrittenFiles) == 0 {
		t.Fatal("expected at least one skill file to be written")
	}

	targetPath := filepath.Join(dir, "security-audit", "SKILL.md")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	meta, err := ParseSkillMetadata(data)
	if err != nil {
		t.Fatalf("ParseSkillMetadata() error = %v", err)
	}
	if meta.Name != "Security Audit" {
		t.Fatalf("expected frontmatter name, got %q", meta.Name)
	}
	if meta.Category != "security" {
		t.Fatalf("expected category security, got %q", meta.Category)
	}
	if !strings.Contains(string(data), "## Validation") {
		t.Fatalf("expected validation section, got:\n%s", string(data))
	}
	if !strings.Contains(output.String(), "Skill generation preview") {
		t.Fatalf("expected preview output, got %q", output.String())
	}
}

func TestSkillGeneratorGenerateRequiresConfirmation(t *testing.T) {
	dir := t.TempDir()
	input := bytes.NewBufferString("n\n")
	var output bytes.Buffer

	generator := NewSkillGenerator(skillGeneratorSpec(), SkillGeneratorConfig{
		OutputDir: dir,
		Confirm:   true,
		Input:     input,
		Output:    &output,
	})

	_, err := generator.Generate()
	if !errors.Is(err, ErrSkillGenerationDeclined) {
		t.Fatalf("expected ErrSkillGenerationDeclined, got %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(dir, "security-audit", "SKILL.md")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no file to be written, stat error = %v", statErr)
	}
	if !strings.Contains(output.String(), "Generate these skill files?") {
		t.Fatalf("expected confirmation prompt, got %q", output.String())
	}
}

func skillGeneratorSpec() *types.AgentSpec {
	spec := types.NewAgentSpec("security reviewer", "1.0.0")
	spec.Metadata.Description = "Review code and security-sensitive changes."
	spec.Metadata.Tags = []string{"review", "security"}
	spec.Capabilities = []types.Capability{
		{
			ID:          "cap-1",
			Name:        "Code Review",
			Description: "Inspect code changes and explain the main risks.",
			Category:    "development",
			Level:       "expert",
			Keywords:    []string{"review", "code"},
		},
	}
	spec.Skills = []types.Skill{
		{
			ID:          "skill-1",
			Name:        "Security Audit",
			Description: "Assess authentication, authorization, and secret handling risks.",
			Complexity:  "high",
		},
	}
	spec.Tools = []types.Tool{
		{
			ID:          "tool-1",
			Name:        "Shell",
			Description: "Execute shell commands",
			Category:    "processing",
			RiskLevel:   "medium",
		},
	}
	spec.Intent.Goals.Primary.Main.Description = "Review code, validate tests, and flag security regressions."
	spec.Intent.Objectives.Functional = []types.FunctionalRequirement{
		{
			ID:                 "fr-1",
			Description:        "Review diffs, test coverage, and authentication changes.",
			AcceptanceCriteria: []string{"Security issues are explained clearly"},
			Testable:           true,
		},
	}
	return spec
}

func containsPlannedSkill(skills []PlannedSkill, name string) bool {
	for _, skill := range skills {
		if skill.Name == name {
			return true
		}
	}
	return false
}
