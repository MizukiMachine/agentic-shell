// main_test.go - メインパッケージのテスト
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// getBinaryPath はテスト用バイナリのパスを取得します
func getBinaryPath() string {
	// 作業ディレクトリからの相対パスを解決
	wd, _ := os.Getwd()
	// プロジェクトルートに移動してbin/agentic-shellを探す
	projectRoot := filepath.Dir(filepath.Dir(wd))
	return filepath.Join(projectRoot, "bin", "agentic-shell")
}

// TestVersionCommand は version コマンドをテストします
func TestVersionCommand(t *testing.T) {
	// ビルドしたバイナリをテスト
	binaryPath := getBinaryPath()
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
	binaryPath := getBinaryPath()
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
	binaryPath := getBinaryPath()
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
