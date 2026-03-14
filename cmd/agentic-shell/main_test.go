// main_test.go - メインパッケージのテスト
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	testBinaryPath  string
	testProjectRoot string
)

func TestMain(m *testing.M) {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	testProjectRoot = filepath.Dir(filepath.Dir(wd))

	tempDir, err := os.MkdirTemp("", "agentic-shell-test-binary-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	testBinaryPath = filepath.Join(tempDir, "agentic-shell")
	buildCmd := exec.Command("go", "build", "-o", testBinaryPath, "./cmd/agentic-shell")
	buildCmd.Dir = testProjectRoot
	if output, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build test binary: %v\n%s", err, output)
		_ = os.RemoveAll(tempDir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}

// getBinaryPath はテスト用バイナリのパスを取得します
func getBinaryPath(t *testing.T) string {
	t.Helper()

	if testBinaryPath == "" {
		t.Fatal("test binary path is not initialized")
	}

	return testBinaryPath
}

// TestVersionCommand は version コマンドをテストします
func TestVersionCommand(t *testing.T) {
	// ビルドしたバイナリをテスト
	binaryPath := getBinaryPath(t)
	cmd := exec.Command(binaryPath, "version")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		t.Fatalf("version コマンドの実行に失敗しました: %v", err)
	}

	output := out.String()
	// バージョン情報が出力されることを確認
	if !strings.Contains(output, "agentic-shell") {
		t.Errorf("バージョン出力に 'agentic-shell' が含まれていません: %s", output)
	}
}

// TestHelpCommand は help コマンドをテストします
func TestHelpCommand(t *testing.T) {
	binaryPath := getBinaryPath(t)
	cmd := exec.Command(binaryPath, "--help")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		t.Fatalf("help コマンドの実行に失敗しました: %v", err)
	}

	output := out.String()
	// ヘルプメッセージが含まれることを確認
	// Long説明は--helpで表示される
	if !strings.Contains(output, "AIエージェント") {
		t.Errorf("ヘルプ出力に説明文が含まれていません: %s", output)
	}
}

// TestRootCommand は引数なしで実行した場合をテストします
func TestRootCommand(t *testing.T) {
	binaryPath := getBinaryPath(t)
	cmd := exec.Command(binaryPath)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		t.Fatalf("ルートコマンドの実行に失敗しました: %v", err)
	}

	output := out.String()
	// デフォルトでヘルプが表示されることを確認
	if !strings.Contains(output, "Usage:") {
		t.Errorf("ヘルプ出力が表示されていません: %s", output)
	}
}
