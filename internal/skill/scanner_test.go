package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseSkillMetadataMarkdownFrontMatter(t *testing.T) {
	meta, err := ParseSkillMetadata([]byte(strings.TrimSpace(`
---
name: skill-name
category: development
description: Description
tools:
  - cargo build
  - cargo test
tags:
  - rust
  - testing
---

# skill-name
`)))
	if err != nil {
		t.Fatalf("ParseSkillMetadata() error = %v", err)
	}

	if meta.Name != "skill-name" {
		t.Fatalf("expected name to be parsed, got %q", meta.Name)
	}
	if meta.Category != "development" {
		t.Fatalf("expected category to be parsed, got %q", meta.Category)
	}
	if len(meta.Tools) != 2 || meta.Tools[1] != "cargo test" {
		t.Fatalf("expected tools to be parsed, got %v", meta.Tools)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "rust" {
		t.Fatalf("expected tags to be parsed, got %v", meta.Tags)
	}
}

func TestParseSkillMetadataBrokenYAMLReturnsError(t *testing.T) {
	_, err := ParseSkillMetadata([]byte(strings.TrimSpace(`
name: broken
category: development
tools:
  - cargo test
description: [missing
`)))
	if err == nil {
		t.Fatal("expected broken YAML to return an error")
	}
}

func TestParseSkillFileMarkdownFallbackExtractsTitleAndSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy-skill", "SKILL.md")

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(`
# Legacy Markdown Skill

Supports legacy skills without front matter.

## Details
- Keep compatibility
`)), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	skill, err := parseSkillFile(dir, path)
	if err != nil {
		t.Fatalf("parseSkillFile() error = %v", err)
	}

	if skill.Metadata.Name != "Legacy Markdown Skill" {
		t.Fatalf("expected markdown heading fallback, got %q", skill.Metadata.Name)
	}
	if skill.Metadata.Description != "Supports legacy skills without front matter." {
		t.Fatalf("expected first paragraph fallback, got %q", skill.Metadata.Description)
	}
}

func TestScanSkillsFindsMarkdownAndCustomSkillFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"README.md": strings.TrimSpace(`
---
name: ignore-me
---
`),
		"code-review/SKILL.md": strings.TrimSpace(`
---
name: code-review
category: development
description: Review code changes
tags:
  - review
---
`),
		"security/security.md": strings.TrimSpace(`
---
name: security
category: security
description: Security checks
---
`),
		"flow.skill": strings.TrimSpace(`
name: flow
category: automation
description: YAML based skill file
`),
	}

	for relativePath, content := range files {
		fullPath := filepath.Join(dir, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", relativePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content+"\n"), 0644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", relativePath, err)
		}
	}

	skills, err := ScanSkills(dir)
	if err != nil {
		t.Fatalf("ScanSkills() error = %v", err)
	}

	if len(skills) != 3 {
		t.Fatalf("expected 3 skill files, got %d", len(skills))
	}
	if skills[0].Path != "code-review/SKILL.md" || skills[1].Path != "flow.skill" || skills[2].Path != "security/security.md" {
		t.Fatalf("unexpected skill paths: %+v", skills)
	}
}

func TestSkillCacheUsesModificationTime(t *testing.T) {
	dir := t.TempDir()
	cache := NewSkillCache()
	path := filepath.Join(dir, "cached", "SKILL.md")

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(`
---
name: first
description: first version
---
`)), 0644); err != nil {
		t.Fatalf("WriteFile(first) error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	originalModTime := info.ModTime()

	skills, err := cache.ScanSkills(dir)
	if err != nil {
		t.Fatalf("ScanSkills(first) error = %v", err)
	}
	if got := skills[0].Metadata.Name; got != "first" {
		t.Fatalf("expected first cached value, got %q", got)
	}

	if err := os.WriteFile(path, []byte(strings.TrimSpace(`
---
name: second
description: second version
---
`)), 0644); err != nil {
		t.Fatalf("WriteFile(second) error = %v", err)
	}
	if err := os.Chtimes(path, originalModTime, originalModTime); err != nil {
		t.Fatalf("Chtimes(original) error = %v", err)
	}

	skills, err = cache.ScanSkills(dir)
	if err != nil {
		t.Fatalf("ScanSkills(cached) error = %v", err)
	}
	if got := skills[0].Metadata.Name; got != "first" {
		t.Fatalf("expected cached value when mtime is unchanged, got %q", got)
	}

	updatedModTime := originalModTime.Add(2 * time.Second)
	if err := os.Chtimes(path, updatedModTime, updatedModTime); err != nil {
		t.Fatalf("Chtimes(updated) error = %v", err)
	}

	skills, err = cache.ScanSkills(dir)
	if err != nil {
		t.Fatalf("ScanSkills(updated) error = %v", err)
	}
	if got := skills[0].Metadata.Name; got != "second" {
		t.Fatalf("expected updated value after mtime change, got %q", got)
	}
}
