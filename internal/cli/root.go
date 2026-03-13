// Package cli は agentic-shell のCLIコマンドを提供します
package cli

import (
	"fmt"
	"os"

	appconfig "github.com/MizukiMachine/agentic-shell/internal/config"
	"github.com/spf13/cobra"
)

// グローバルフラグの変数
var (
	cfgFile        string
	currentConfig  = appconfig.LoadWithDefaults()
	currentVerbose bool
	initConfigErr  error
	configFileUsed string
)

// バージョン情報（ビルド時に設定可能）
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// rootCmd はベースとなるコマンドです
var rootCmd = &cobra.Command{
	Use:   "agentic-shell",
	Short: "AIエージェント統合シェル",
	Long: `agentic-shell は複数のAIエージェントを統合管理する
ターミナルベースのシェルアプリケーションです。

Claude、GPT、Gemini などの AI エージェントと対話しながら
開発作業を効率化できます。

使用例:
  agentic-shell spec-gather "コードレビューエージェントが欲しい"
  agentic-shell generate --from spec.yaml
  agentic-shell version`,
	Run: func(cmd *cobra.Command, args []string) {
		// デフォルトの動作: ヘルプを表示
		cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfigErr
	},
}

// Execute はCLIアプリケーションを実行します
func Execute() error {
	return rootCmd.Execute()
}

// init はパッケージ初期化時にフラグと設定初期化フックを登録します
func init() {
	cobra.OnInitialize(func() {
		initConfigErr = initConfig()
	})

	// グローバルフラグ
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "設定ファイル (デフォルト: $HOME/.agentic-shell.yaml)")
	rootCmd.PersistentFlags().StringP("output-dir", "o", "", "出力ディレクトリ (設定値を上書き)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "詳細出力モード")
}

// initConfig は設定ファイルと環境変数を読み込みます
func initConfig() error {
	loader := appconfig.NewLoader()

	if cfgFile != "" {
		loader.WithConfigPath(cfgFile)
	}

	cfg, err := loader.Load()
	if err != nil {
		return err
	}

	currentConfig = cfg
	configFileUsed = loader.ConfigFileUsed()

	if flag := rootCmd.PersistentFlags().Lookup("output-dir"); flag != nil && flag.Changed {
		currentConfig.Output.Directory = flag.Value.String()
	}

	verbose, err := rootCmd.PersistentFlags().GetBool("verbose")
	if err != nil {
		return err
	}
	currentVerbose = verbose

	if currentVerbose && configFileUsed != "" {
		fmt.Fprintln(os.Stderr, "設定ファイルを使用:", configFileUsed)
	}

	return nil
}

// GetRootCmd はルートコマンドを返します（テスト用）
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// GetVerbose は詳細モードの状態を返します。
func GetVerbose() bool {
	return currentVerbose
}

// GetOutputDir は出力ディレクトリを返します。
func GetOutputDir() string {
	return currentConfig.Output.Directory
}

// GetConfig はロード済みの設定を返します。
func GetConfig() *appconfig.Config {
	return currentConfig
}
