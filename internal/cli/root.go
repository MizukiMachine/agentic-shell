// Package cli は agentic-shell のCLIコマンドを提供します
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// グローバルフラグの変数
var (
	cfgFile string
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
}

// Execute はCLIアプリケーションを実行します
func Execute() error {
	return rootCmd.Execute()
}

// init はパッケージ初期化時にフラグとViperを設定します
func init() {
	cobra.OnInitialize(initConfig)

	// グローバルフラグ
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "設定ファイル (デフォルト: $HOME/.agentic-shell.yaml)")
	rootCmd.PersistentFlags().StringP("output-dir", "o", ".", "出力ディレクトリ")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "詳細出力モード")

	// Viperにフラグをバインド
	viper.BindPFlag("output-dir", rootCmd.PersistentFlags().Lookup("output-dir"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))

	// デフォルト値を設定
	viper.SetDefault("output-dir", ".")
	viper.SetDefault("verbose", false)
}

// initConfig は設定ファイルと環境変数を読み込みます
func initConfig() {
	if cfgFile != "" {
		// 指定された設定ファイルを使用
		viper.SetConfigFile(cfgFile)
	} else {
		// ホームディレクトリを検索
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// 設定ファイルの検索パスを追加
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".agentic-shell")
	}

	// 環境変数を読み込み
	viper.SetEnvPrefix("AGENTIC")
	viper.AutomaticEnv()

	// 設定ファイルを読み込み（存在する場合）
	if err := viper.ReadInConfig(); err == nil && viper.GetBool("verbose") {
		fmt.Fprintln(os.Stderr, "設定ファイルを使用:", viper.ConfigFileUsed())
	}
}

// GetRootCmd はルートコマンドを返します（テスト用）
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// GetVerbose は詳細モードの状態を返します（Viperから取得）
func GetVerbose() bool {
	return viper.GetBool("verbose")
}

// GetOutputDir は出力ディレクトリを返します（Viperから取得）
func GetOutputDir() string {
	return viper.GetString("output-dir")
}
