package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDocumentMarkdown(t *testing.T) {
	doc, err := ParseDocument("spec.md", []byte(strings.TrimSpace(`
# Code Review Pipeline

Generate consistent code review guidance for pull requests.

## Requirements
- Parse Markdown specifications
- Match required skills with existing skills

## Skills
- code-review: Analyze diffs and provide feedback
`)), "markdown")
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	if doc.Title != "Code Review Pipeline" {
		t.Fatalf("expected title to be parsed, got %q", doc.Title)
	}
	if len(doc.Sections) < 2 {
		t.Fatalf("expected sections to be parsed, got %+v", doc.Sections)
	}
	if len(doc.Sections[1].Bullets) == 0 {
		t.Fatalf("expected markdown bullets to be extracted, got %+v", doc.Sections[1])
	}
}

func TestExtractBuildsRequirementsAndSkillRequirements(t *testing.T) {
	env, err := ParseStdin(&Envelope{}, "spec.md", []byte(strings.TrimSpace(`
# Skill Pipeline

Create a CLI pipeline for reviewing code and checking security issues.

## Requirements
- Support Unix pipes
- Generate placeholder skills when coverage is missing
`)))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}

	if err := Extract(env); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if env.Extraction == nil {
		t.Fatal("expected extraction result")
	}
	if env.Extraction.AgentSpec == nil {
		t.Fatal("expected agent spec to be materialized")
	}
	if len(env.Extraction.Requirements) == 0 {
		t.Fatal("expected requirements to be extracted")
	}
	if len(env.Extraction.SkillRequirements) == 0 {
		t.Fatal("expected skill requirements to be inferred")
	}
}

func TestScanMatchGenerateAndWritePipeline(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".claude", "skills", "code-review")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(strings.TrimSpace(`
---
name: "code-review"
description: "Analyze code changes and provide review feedback"
---

# code-review
`)), 0644); err != nil {
		t.Fatalf("failed to seed skill file: %v", err)
	}

	env, err := ParseStdin(&Envelope{}, "spec.md", []byte(strings.TrimSpace(`
# Review and Security

Build a CLI workflow for code review and security validation.
`)))
	if err != nil {
		t.Fatalf("ParseStdin() error = %v", err)
	}
	if err := Extract(env); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if err := ScanSkills(env, filepath.Join(dir, ".claude", "skills")); err != nil {
		t.Fatalf("ScanSkills() error = %v", err)
	}
	if err := MatchSkills(env); err != nil {
		t.Fatalf("MatchSkills() error = %v", err)
	}
	if len(env.Match.Matches) == 0 {
		t.Fatal("expected matches to be produced")
	}
	if err := GenerateMissingSkills(env); err != nil {
		t.Fatalf("GenerateMissingSkills() error = %v", err)
	}
	if len(env.SkillGen.Files) == 0 {
		t.Fatal("expected placeholder skill files for missing coverage")
	}
	if err := WriteGeneratedFiles(env, dir, filepath.Join(".claude", "skills"), true); err != nil {
		t.Fatalf("WriteGeneratedFiles() error = %v", err)
	}
	if len(env.Output.WrittenFiles) == 0 {
		t.Fatal("expected generated files to be written")
	}
}

func TestWriteGeneratedFilesRespectsSkillsDir(t *testing.T) {
	dir := t.TempDir()
	env := &Envelope{
		Match: &MatchResult{
			MissingSkills: []SkillRequirement{
				{
					ID:          "req-1",
					Name:        "security audit",
					Description: "Review security-sensitive code paths",
					Required:    true,
				},
			},
		},
	}

	if err := GenerateMissingSkills(env); err != nil {
		t.Fatalf("GenerateMissingSkills() error = %v", err)
	}

	skillsDir := filepath.Join("custom", "skills")
	if err := WriteGeneratedFiles(env, dir, skillsDir, true); err != nil {
		t.Fatalf("WriteGeneratedFiles() error = %v", err)
	}

	expectedPath := filepath.Join(dir, skillsDir, "security-audit", "SKILL.md")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected generated file at custom skills dir: %v", err)
	}

	defaultPath := filepath.Join(dir, ".claude", "skills", "security-audit", "SKILL.md")
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("expected default skills path to remain unused, stat err = %v", err)
	}
}

func TestRenderSkillPlaceholderEscapesYAMLFrontMatter(t *testing.T) {
	content, err := renderSkillPlaceholder(SkillRequirement{
		ID:          "req-quoted",
		Name:        `review "quotes"`,
		Description: "Handle quoted names safely",
		Required:    true,
	})
	if err != nil {
		t.Fatalf("renderSkillPlaceholder() error = %v", err)
	}

	doc, err := ParseDocument("SKILL.md", []byte(content), "markdown")
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	if got := stringValue(doc.Metadata["name"]); got != `review "quotes"` {
		t.Fatalf("expected quoted skill name to round-trip, got %q", got)
	}
}

func TestScanSkillsExcludesReadmeAndNonSkillMarkdown(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"README.md": strings.TrimSpace(`
---
name: "root-readme"
description: "must be ignored"
---
`),
		"code-review/SKILL.md": strings.TrimSpace(`
---
name: "code-review"
description: "Primary skill file"
---
`),
		"security/security.md": strings.TrimSpace(`
---
name: "security"
description: "Skill file named after its directory"
---
`),
		"docs/README.md": strings.TrimSpace(`
---
name: "nested-readme"
description: "must be ignored"
---
`),
		"docs/notes.md": strings.TrimSpace(`
---
name: "notes"
description: "must be ignored"
---
`),
		"flow.skill": strings.TrimSpace(`
name: flow
description: custom skill extension
`),
	}

	for relativePath, content := range files {
		fullPath := filepath.Join(dir, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create parent dir for %s: %v", relativePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content+"\n"), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", relativePath, err)
		}
	}

	env := &Envelope{}
	if err := ScanSkills(env, dir); err != nil {
		t.Fatalf("ScanSkills() error = %v", err)
	}

	paths := make([]string, 0, len(env.SkillScan.Skills))
	for _, skill := range env.SkillScan.Skills {
		paths = append(paths, skill.Path)
	}

	expected := []string{
		"code-review/SKILL.md",
		"flow.skill",
		"security/security.md",
	}
	if strings.Join(paths, ",") != strings.Join(expected, ",") {
		t.Fatalf("unexpected scanned skill paths: got %v want %v", paths, expected)
	}
}
