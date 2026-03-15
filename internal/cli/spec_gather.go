package cli

import (
	"bufio"
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
	specOutput      string
	specQuick       bool
	specTotalTimeout int
	specInputTimeout int
	specFormat      string
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
  ags spec-gather "コードレビューエージェントが欲しい"

  # 出力ファイルを指定
  ags spec-gather --output spec.yaml "テスト自動化"

  # クイックモード（最小限の質問で収束、低信頼度でも継続）
  ags spec-gather --quick "ドキュメント生成"

  # JSON形式で出力
  ags spec-gather --format json --output spec.json "APIエージェント"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSpecGather,
}

func init() {
	rootCmd.AddCommand(specGatherCmd)

	// フラグ設定
	specGatherCmd.Flags().StringVar(&specOutput, "output", "", "出力ファイルパス (指定しない場合は標準出力)")
	specGatherCmd.Flags().BoolVarP(&specQuick, "quick", "q", false, "クイックモード（低信頼度でも継続）")
	specGatherCmd.Flags().IntVarP(&specTotalTimeout, "timeout", "t", 0, "全体タイムアウト（秒）、0=設定値使用")
	specGatherCmd.Flags().IntVar(&specInputTimeout, "input-timeout", 0, "入力待ちタイムアウト（秒）、0=設定値使用")
	specGatherCmd.Flags().StringVarP(&specFormat, "format", "f", "yaml", "出力形式 (yaml または json)")
}

// runSpecGather はspec-gatherコマンドのメイン処理です
func runSpecGather(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	outputDir := cfg.Output.Directory
	verbose := GetVerbose()

	inputReader := bufio.NewReader(os.Stdin)
	var input string
	if len(args) == 0 {
		var err error
		input, err = PromptForInput(inputReader, os.Stderr, "収集したい仕様を入力してください: ")
		if err != nil {
			return fmt.Errorf("入力取得エラー: %w", err)
		}
	} else {
		input = args[0]
	}

	// タイムアウト解決ロジック: CLI > 設定 > デフォルト
	// 全体タイムアウト
	totalTimeout := time.Duration(0)
	if cmd.Flags().Changed("timeout") {
		totalTimeout = time.Duration(specTotalTimeout) * time.Second
	} else if cfgTimeout, err := cfg.Interaction.GetTotalTimeout(); err == nil {
		totalTimeout = cfgTimeout
	}

	// 入力タイムアウト
	inputTimeout := time.Duration(0)
	if cmd.Flags().Changed("input-timeout") {
		inputTimeout = time.Duration(specInputTimeout) * time.Second
	} else if cfgTimeout, err := cfg.Interaction.GetInputTimeout(); err == nil {
		inputTimeout = cfgTimeout
	}

	// コンテキストにタイムアウトを設定
	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	// Gathererを作成
	gatherer := spec.NewGatherer(inputReader, os.Stderr)
	gatherer.SetMaxRounds(cfg.Gathering.MaxQuestionRounds)
	gatherer.SetConfidenceThreshold(cfg.Gathering.ConfidenceThreshold)
	gatherer.SetInputTimeout(inputTimeout)

	if verbose {
		fmt.Fprintf(os.Stderr, "入力: %s\n", input)
		fmt.Fprintf(os.Stderr, "全体タイムアウト: %s\n", totalTimeout)
		fmt.Fprintf(os.Stderr, "入力タイムアウト: %s\n", inputTimeout)
		if specQuick {
			fmt.Fprintf(os.Stderr, "クイックモード: 有効\n")
		}
	}

	// インタラクティブ収集を実行
	agentSpec, err := gatherer.GatherInteractive(ctx, input)

	// エラーハンドリング（クイックモード対応）
	if err != nil {
		// クイックモードで、かつ部分結果がある場合は継続
		if specQuick && agentSpec != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "クイックモード: エラーあり but 部分結果を使用: %v\n", err)
			}
		} else {
			return fmt.Errorf("仕様収集エラー: %w", err)
		}
	}

	// クイックモードの場合、低信頼度でも警告のみ
	if specQuick && agentSpec.Intent.Metadata.Confidence < cfg.Gathering.ConfidenceThreshold {
		fmt.Fprintf(os.Stderr, "⚠️ クイックモード: 信頼度 %.2f は閾値 %.2f 未満です\n",
			agentSpec.Intent.Metadata.Confidence, cfg.Gathering.ConfidenceThreshold)
	}

	// 出力形式を決定（優先順位: --format > 拡張子 > 設定ファイル > デフォルト）
	format := specFormat
	if !cmd.Flags().Changed("format") {
		// --format が指定されていない場合、設定ファイルの値を確認
		if cfg.Output.Format != "" {
			format = cfg.Output.Format
		}
	}
	// ファイル拡張子が明示的な場合はそれを優先
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

		// ファイルの親ディレクトリも作成
		fullPath := filepath.Join(outDir, specOutput)
		parentDir := filepath.Dir(fullPath)
		if parentDir != "." && parentDir != "" {
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("親ディレクトリ作成エラー: %w", err)
			}
		}

		// ファイルに書き込み
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
