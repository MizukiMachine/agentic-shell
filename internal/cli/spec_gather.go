package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MizukiMachine/agentic-shell/internal/spec"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// specGatherCmd のフラグ変数
var (
	specOutput string
	specQuick  bool
	specTimeout int
	specFormat string
)

// specGatherCmd はインタラクティブにAgentSpecを収集するコマンドです
var specGatherCmd = &cobra.Command{
	Use:   "spec-gather [input]",
	Short: "インタラクティブにエージェント仕様を収集",
	Long: `Step-back質問手法を使って、ユーザーからインタラクティブに
エージェント仕様を収集します。

収集した仕様は JSON または YAML 形式で出力できます。

使用例:
  # 基本的な使用方法
  agentic-shell spec-gather "コードレビューエージェントが欲しい"

  # 出力ファイルを指定
  agentic-shell spec-gather --output spec.yaml "テスト自動化"

  # クイックモード（最小限の質問）
  agentic-shell spec-gather --quick "ドキュメント生成"

  # JSON形式で出力
  agentic-shell spec-gather --format json --output spec.json "APIエージェント"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSpecGather,
}

func init() {
	rootCmd.AddCommand(specGatherCmd)

	// フラグ設定
	specGatherCmd.Flags().StringVar(&specOutput, "output", "", "出力ファイルパス (指定しない場合は標準出力)")
	specGatherCmd.Flags().BoolVarP(&specQuick, "quick", "q", false, "クイックモード（最小限の質問で収束）")
	specGatherCmd.Flags().IntVarP(&specTimeout, "timeout", "t", 300, "タイムアウト（秒）")
	specGatherCmd.Flags().StringVarP(&specFormat, "format", "f", "yaml", "出力形式 (yaml または json)")
}

// runSpecGather はspec-gatherコマンドのメイン処理です
func runSpecGather(cmd *cobra.Command, args []string) error {
	input := args[0]

	// コンテキストにタイムアウトを設定
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(specTimeout)*time.Second)
	defer cancel()

	// Gathererを作成
	gatherer := spec.NewGatherer(os.Stdin, os.Stderr)

	if verbose {
		fmt.Fprintf(os.Stderr, "入力: %s\n", input)
		fmt.Fprintf(os.Stderr, "タイムアウト: %d秒\n", specTimeout)
	}

	// インタラクティブ収集を実行
	agentSpec, err := gatherer.GatherInteractive(ctx, input)
	if err != nil {
		return fmt.Errorf("仕様収集エラー: %w", err)
	}

	// クイックモードの場合、信頼度が低くても許可
	if specQuick && agentSpec.Intent.Metadata.Confidence < spec.ConfidenceThreshold {
		if verbose {
			fmt.Fprintf(os.Stderr, "クイックモード: 信頼度 %.2f で継続\n", agentSpec.Intent.Metadata.Confidence)
		}
	}

	// 出力形式を決定
	format := specFormat
	if specOutput != "" {
		ext := filepath.Ext(specOutput)
		if ext == ".json" {
			format = "json"
		} else if ext == ".yaml" || ext == ".yml" {
			format = "yaml"
		}
	}

	// 出力を生成
	var output []byte
	switch format {
	case "json":
		output, err = json.MarshalIndent(agentSpec, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON変換エラー: %w", err)
		}
	default: // yaml
		output, err = yaml.Marshal(agentSpec)
		if err != nil {
			return fmt.Errorf("YAML変換エラー: %w", err)
		}
	}

	// 出力先を決定
	if specOutput != "" {
		// 出力ディレクトリを作成
		outDir := outputDir
		if outDir != "." {
			if err := os.MkdirAll(outDir, 0755); err != nil {
				return fmt.Errorf("ディレクトリ作成エラー: %w", err)
			}
		}

		// ファイルに書き込み
		fullPath := filepath.Join(outDir, specOutput)
		if err := os.WriteFile(fullPath, output, 0644); err != nil {
			return fmt.Errorf("ファイル書き込みエラー: %w", err)
		}

		fmt.Printf("仕様を出力しました: %s\n", fullPath)
		if verbose {
			fmt.Printf("信頼度スコア: %.2f\n", agentSpec.Intent.Metadata.Confidence)
		}
	} else {
		// 標準出力に表示
		fmt.Println(string(output))
	}

	return nil
}
