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

// generateCmd のフラグ変数
var (
	genFrom   string
	genQuick  bool
	genTimeout int
)

// generateCmd はエージェント定義を生成するコマンドです
var generateCmd = &cobra.Command{
	Use:   "generate [input]",
	Short: "エージェント定義ファイルを生成",
	Long: `spec-gatherを実行し、エージェント定義ファイルを生成します。
既存の仕様ファイルから読み込むこともできます。

使用例:
  # 新規に仕様を収集して生成
  agentic-shell generate "ドキュメント生成エージェントを作りたい"

  # 既存の仕様ファイルから生成
  agentic-shell generate --from spec.yaml

  # クイックモード
  agentic-shell generate --quick "簡易モード"

  # 出力ディレクトリを指定
  agentic-shell generate -o ./output "テストエージェント"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// フラグ設定
	generateCmd.Flags().StringVarP(&genFrom, "from", "f", "", "入力仕様ファイル (指定しない場合はspec-gatherを実行)")
	generateCmd.Flags().BoolVarP(&genQuick, "quick", "q", false, "クイックモード")
	generateCmd.Flags().IntVarP(&genTimeout, "timeout", "t", 300, "タイムアウト（秒）")
}

// runGenerate はgenerateコマンドのメイン処理です
func runGenerate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(genTimeout)*time.Second)
	defer cancel()

	var agentSpec *spec.AgentSpec
	var err error

	if genFrom != "" {
		// 既存の仕様ファイルから読み込み
		agentSpec, err = loadSpecFromFile(genFrom)
		if err != nil {
			return fmt.Errorf("仕様ファイル読み込みエラー: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "仕様ファイルを読み込み: %s\n", genFrom)
		}
	} else {
		// 引数チェック
		if len(args) == 0 {
			return fmt.Errorf("入力を指定するか --from フラグを使用してください")
		}

		// spec-gatherを実行
		input := args[0]
		gatherer := spec.NewGatherer(os.Stdin, os.Stderr)

		if verbose {
			fmt.Fprintf(os.Stderr, "仕様収集中: %s\n", input)
		}

		agentSpec, err = gatherer.GatherInteractive(ctx, input)
		if err != nil {
			return fmt.Errorf("仕様収集エラー: %w", err)
		}

		// クイックモードの場合、信頼度が低くても許可
		if genQuick && agentSpec.Intent.Metadata.Confidence < spec.ConfidenceThreshold {
			if verbose {
				fmt.Fprintf(os.Stderr, "クイックモード: 信頼度 %.2f で継続\n", agentSpec.Intent.Metadata.Confidence)
			}
		}
	}

	// 出力ディレクトリを作成
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("ディレクトリ作成エラー: %w", err)
		}
	}

	// エージェント定義ファイルを生成
	outputPath := filepath.Join(outputDir, "agent-definition.yaml")
	if err := writeAgentDefinition(agentSpec, outputPath); err != nil {
		return fmt.Errorf("エージェント定義出力エラー: %w", err)
	}

	fmt.Printf("エージェント定義を生成しました: %s\n", outputPath)
	if verbose {
		fmt.Printf("信頼度スコア: %.2f\n", agentSpec.Intent.Metadata.Confidence)
		fmt.Printf("エージェント名: %s\n", agentSpec.Metadata.Name)
	}

	return nil
}

// loadSpecFromFile はファイルからAgentSpecを読み込みます
func loadSpecFromFile(path string) (*spec.AgentSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	agentSpec := &spec.AgentSpec{}

	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, agentSpec); err != nil {
			return nil, err
		}
	default: // yaml
		if err := yaml.Unmarshal(data, agentSpec); err != nil {
			return nil, err
		}
	}

	return agentSpec, nil
}

// writeAgentDefinition はエージェント定義ファイルを書き込みます
func writeAgentDefinition(agentSpec *spec.AgentSpec, path string) error {
	data, err := yaml.Marshal(agentSpec)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
