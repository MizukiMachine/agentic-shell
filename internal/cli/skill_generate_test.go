package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveSkillGenerateOutputDirDefaults(t *testing.T) {
	cmd := &cobra.Command{Use: "skill-generate"}
	cmd.PersistentFlags().String("output-dir", "", "output dir")

	if got := resolveSkillGenerateOutputDir(cmd); got != ".claude/skills" {
		t.Fatalf("expected default output dir, got %q", got)
	}
}

func TestResolveSkillGenerateOutputDirUsesFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "skill-generate"}
	cmd.PersistentFlags().String("output-dir", "", "output dir")
	if err := cmd.PersistentFlags().Set("output-dir", "custom-skills"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if got := resolveSkillGenerateOutputDir(cmd); got != "custom-skills" {
		t.Fatalf("expected custom output dir, got %q", got)
	}
}
