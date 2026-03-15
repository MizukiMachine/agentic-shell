package cli

import (
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	cmd := GetRootCmd()
	if cmd == nil {
		t.Fatal("rootCmd is nil")
	}

	if cmd.Use != "ags" {
		t.Fatalf("expected root use to be ags, got %s", cmd.Use)
	}

	// ヘルプテキストの検証
	if !strings.Contains(cmd.Short, "AI") {
		t.Errorf("expected short description to contain 'AI', got %s", cmd.Short)
	}
}

func TestVersionCommandExists(t *testing.T) {
	cmd := GetRootCmd()
	versionCmd, _, err := cmd.Find([]string{"version"})
	if err != nil {
		t.Fatalf("version command not found: %v", err)
	}
	if versionCmd == nil {
		t.Fatal("version command is nil")
	}
	if versionCmd.Name() != "version" {
		t.Errorf("expected command name 'version', got %s", versionCmd.Name())
	}
}

func TestSpecGatherCommandExists(t *testing.T) {
	cmd := GetRootCmd()
	specGatherCmd, _, err := cmd.Find([]string{"spec-gather"})
	if err != nil {
		t.Fatalf("spec-gather command not found: %v", err)
	}
	if specGatherCmd == nil {
		t.Fatal("spec-gather command is nil")
	}
	if specGatherCmd.Name() != "spec-gather" {
		t.Errorf("expected command name 'spec-gather', got %s", specGatherCmd.Name())
	}

	noLLMFlag := specGatherCmd.Flags().Lookup("no-llm")
	if noLLMFlag == nil {
		t.Fatal("spec-gather no-llm flag not found")
	}
}

func TestGenerateCommandExists(t *testing.T) {
	cmd := GetRootCmd()
	generateCmd, _, err := cmd.Find([]string{"generate"})
	if err != nil {
		t.Fatalf("generate command not found: %v", err)
	}
	if generateCmd == nil {
		t.Fatal("generate command is nil")
	}
	if generateCmd.Name() != "generate" {
		t.Errorf("expected command name 'generate', got %s", generateCmd.Name())
	}
}

func TestPipelineCommandsExist(t *testing.T) {
	tests := []string{
		"parse",
		"extract",
		"skill-scan",
		"match",
		"skill-gen",
		"skill-generate",
		"output",
		"plugin",
	}

	cmd := GetRootCmd()
	for _, name := range tests {
		found, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatalf("%s command not found: %v", name, err)
		}
		if found == nil {
			t.Fatalf("%s command is nil", name)
		}
		if found.Name() != name {
			t.Fatalf("expected command name %q, got %q", name, found.Name())
		}
	}
}

func TestVersionOutput(t *testing.T) {
	// バージョン情報を設定
	Version = "test-version"
	Commit = "test-commit"
	Date = "test-date"

	// printVersion関数が存在することを確認
	// 注: printVersionは直接fmt.Printfを使用するため、
	// 出力のテストは統合テストで行う
}

func TestGlobalFlags(t *testing.T) {
	cmd := GetRootCmd()

	// --config フラグの確認
	configFlag := cmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("config flag not found")
	}

	// --output-dir フラグの確認
	outputDirFlag := cmd.PersistentFlags().Lookup("output-dir")
	if outputDirFlag == nil {
		t.Error("output-dir flag not found")
	}

	// --profile フラグの確認
	profileFlag := cmd.PersistentFlags().Lookup("profile")
	if profileFlag == nil {
		t.Error("profile flag not found")
	}

	// --output-format フラグの確認
	outputFormatFlag := cmd.PersistentFlags().Lookup("output-format")
	if outputFormatFlag == nil {
		t.Error("output-format flag not found")
	}

	// --verbose フラグの確認
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("verbose flag not found")
	}
}
