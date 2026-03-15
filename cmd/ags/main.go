// ags - AIエージェント統合シェル
// このパッケージはCLIアプリケーションのエントリーポイントです
package main

import (
	"fmt"
	"os"

	"github.com/MizukiMachine/agentic-shell/internal/cli"
)

// バージョン情報（ビルド時に -ldflags で設定可能）
// 使用例: go build -ldflags "-X main.version=1.0.0 -X main.commit=abc123 -X main.date=2024-01-01"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// CLIパッケージにバージョン情報を設定
	cli.Version = version
	cli.Commit = commit
	cli.Date = date

	// CLIを実行
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
