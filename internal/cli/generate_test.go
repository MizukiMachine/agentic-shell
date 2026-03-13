package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAgentOutputPath(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		agentName string
		want      string
	}{
		{
			name:      "default output dir nests under claude agents",
			outputDir: ".",
			agentName: "Code Reviewer",
			want:      filepath.Join(".claude", "agents", "code-reviewer.md"),
		},
		{
			name:      "existing claude agents dir is reused",
			outputDir: filepath.Join("tmp", ".claude", "agents"),
			agentName: "Code Reviewer",
			want:      filepath.Join("tmp", ".claude", "agents", "code-reviewer.md"),
		},
		{
			name:      "custom output dir appends claude agents",
			outputDir: "out",
			agentName: "Code Reviewer",
			want:      filepath.Join("out", ".claude", "agents", "code-reviewer.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildAgentOutputPath(tt.outputDir, tt.agentName); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestWriteClaudeAgentMarkdownRespectsOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.md")

	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to prepare file: %v", err)
	}

	err := writeClaudeAgentMarkdown("new", path, false)
	if err == nil {
		t.Fatal("expected overwrite protection error")
	}
	if !strings.Contains(err.Error(), "output.overwrite=false") {
		t.Fatalf("expected descriptive error, got %v", err)
	}
}

func TestWriteClaudeAgentMarkdownOverwritesWhenEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.md")

	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to prepare file: %v", err)
	}

	if err := writeClaudeAgentMarkdown("new", path, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if string(data) != "new" {
		t.Fatalf("expected overwritten content, got %q", string(data))
	}
}
