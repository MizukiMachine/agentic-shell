package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MizukiMachine/agentic-shell/internal/agent"
	"github.com/MizukiMachine/agentic-shell/internal/spec"
	"github.com/MizukiMachine/agentic-shell/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// generateCmd のフラグ変数
var (
	genFrom    string
	genQuick   bool
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
	generateCmd.Flags().BoolVarP(&genQuick, "quick", "q", false, "クイックモード（低信頼度でも継続）")
	generateCmd.Flags().IntVarP(&genTimeout, "timeout", "t", 300, "タイムアウト（秒）")
}

// runGenerate はgenerateコマンドのメイン処理です
func runGenerate(cmd *cobra.Command, args []string) error {
	// Viperから設定値を取得（設定ファイル・環境変数を反映）
	outputDir := viper.GetString("output-dir")
	verbose := viper.GetBool("verbose")

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
			// クイックモードの場合、信頼度エラーでも部分結果を使用
			if genQuick && agentSpec != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "クイックモード: 信頼度 %.2f で継続\n", agentSpec.Intent.Metadata.Confidence)
				}
			} else {
				return fmt.Errorf("仕様収集エラー: %w", err)
			}
		}
	}

	// 出力ディレクトリを作成
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("ディレクトリ作成エラー: %w", err)
		}
	}

	generator := agent.NewGenerator()
	claudeDef, err := generator.Generate(ctx, agentSpec)
	if err != nil {
		return fmt.Errorf("エージェント定義生成エラー: %w", err)
	}

	markdown, err := generator.RenderMarkdown(claudeDef)
	if err != nil {
		return fmt.Errorf("Markdownレンダリングエラー: %w", err)
	}

	// エージェント定義ファイルを生成
	outputPath := buildAgentOutputPath(outputDir, claudeDef.Metadata.Name)
	if err := writeClaudeAgentMarkdown(markdown, outputPath); err != nil {
		return fmt.Errorf("エージェント定義出力エラー: %w", err)
	}

	fmt.Printf("エージェント定義を生成しました: %s\n", outputPath)
	if verbose {
		fmt.Printf("信頼度スコア: %.2f\n", agentSpec.Intent.Metadata.Confidence)
		fmt.Printf("エージェント名: %s\n", claudeDef.Metadata.Name)
		fmt.Printf("モデル: %s\n", claudeDef.Model.ModelID)
	}

	return nil
}

// loadSpecFromFile はファイルからAgentSpecを読み込みます
func loadSpecFromFile(path string) (*spec.AgentSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	agentSpec := &types.AgentSpec{}

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

// writeClaudeAgentMarkdown はClaude Code互換のMarkdownファイルを書き込みます
func writeClaudeAgentMarkdown(markdown, path string) error {
	// 親ディレクトリを作成
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("親ディレクトリ作成エラー: %w", err)
		}
	}

	return os.WriteFile(path, []byte(markdown), 0644)
}

func buildAgentOutputPath(outputDir, name string) string {
	filename := agent.MarkdownFileName(name) + ".md"
	cleanDir := filepath.Clean(outputDir)
	if endsWithClaudeAgents(cleanDir) {
		return filepath.Join(cleanDir, filename)
	}
	return filepath.Join(cleanDir, ".claude", "agents", filename)
}

func endsWithClaudeAgents(path string) bool {
	normalized := filepath.ToSlash(filepath.Clean(path))
	return strings.HasSuffix(normalized, "/.claude/agents") || normalized == ".claude/agents"
}
