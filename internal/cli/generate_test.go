package cli

import (
	"path/filepath"
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
