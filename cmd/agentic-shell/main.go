// agentic-shell - AIエージェント統合シェル
// このパッケージはCLIアプリケーションのエントリーポイントです
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// バージョン情報（ビルド時に設定可能）
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// rootCmd はベースとなるコマンドです
var rootCmd = &cobra.Command{
	Use:   "agentic-shell",
	Short: "AIエージェント統合シェル",
	Long: `agentic-shell は複数のAIエージェントを統合管理する
ターミナルベースのシェルアプリケーションです。

Claude、GPT、Gemini などの AI エージェントと対話しながら
開発作業を効率化できます。`,
	Run: func(cmd *cobra.Command, args []string) {
		// デフォルトの動作: ヘルプを表示
		cmd.Help()
	},
}

// versionCmd はバージョン情報を表示するコマンドです
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "バージョン情報を表示",
	Long:  `agentic-shell のバージョン情報を表示します。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("agentic-shell %s\n", version)
		fmt.Printf("  Commit: %s\n", commit)
		fmt.Printf("  Built:  %s\n", date)
	},
}

func init() {
	// バージョンコマンドをルートコマンドに追加
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
