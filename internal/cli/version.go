package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// versionCmd はバージョン情報を表示するコマンドです
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "バージョン情報を表示",
	Long: `agentic-shell のバージョン情報を表示します。

表示内容:
  - バージョン番号
  - Gitコミットハッシュ
  - ビルド日時
  - Go バージョン
  - プラットフォーム情報`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// printVersion はバージョン情報を整形して表示します
func printVersion() {
	fmt.Printf("agentic-shell %s\n", Version)
	fmt.Println("---")
	fmt.Printf("  Commit:  %s\n", Commit)
	fmt.Printf("  Built:   %s\n", Date)
	fmt.Printf("  Go:      %s\n", runtime.Version())
	fmt.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
