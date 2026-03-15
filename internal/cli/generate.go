package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MizukiMachine/agentic-shell/internal/agent"
	outputfmt "github.com/MizukiMachine/agentic-shell/internal/output"
	"github.com/MizukiMachine/agentic-shell/internal/spec"
	"github.com/MizukiMachine/agentic-shell/pkg/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// generateCmd のフラグ変数
var (
	genFrom         string
	genQuick        bool
	genTotalTimeout int
	genInputTimeout int
)

// generateCmd はエージェント定義を生成するコマンドです
var generateCmd = &cobra.Command{
	Use:   "generate [input]",
	Short: "エージェント定義ファイルを生成",
	Long: `spec-gatherを実行し、エージェント定義ファイルを生成します。
既存の仕様ファイルから読み込むこともできます。

使用例:
  # 新規に仕様を収集して生成
  ags generate "ドキュメント生成エージェントを作りたい"

  # 既存の仕様ファイルから生成
  ags generate --from spec.yaml

  # クイックモード
  ags generate --quick "簡易モード"

  # 出力ディレクトリを指定
  ags generate -o ./output "テストエージェント"`,
	Args: cobra.MaximumNArgs(1),
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// フラグ設定
	generateCmd.Flags().StringVarP(&genFrom, "from", "f", "", "入力仕様ファイル (指定しない場合はspec-gatherを実行)")
	generateCmd.Flags().BoolVarP(&genQuick, "quick", "q", false, "クイックモード（低信頼度でも継続）")
	generateCmd.Flags().IntVarP(&genTotalTimeout, "timeout", "t", 0, "全体タイムアウト（秒）、0=設定値使用")
	generateCmd.Flags().IntVar(&genInputTimeout, "input-timeout", 0, "入力待ちタイムアウト（秒）、0=設定値使用")
}

// runGenerate はgenerateコマンドのメイン処理です
func runGenerate(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	outputDir := cfg.Output.Directory
	verbose := GetVerbose()

	// タイムアウト解決ロジック: CLI > 設定 > デフォルト
	// 全体タイムアウト
	totalTimeout := time.Duration(0)
	if cmd.Flags().Changed("timeout") {
		totalTimeout = time.Duration(genTotalTimeout) * time.Second
	} else if cfgTimeout, err := cfg.Interaction.GetTotalTimeout(); err == nil {
		totalTimeout = cfgTimeout
	}

	// 入力タイムアウト
	inputTimeout := time.Duration(0)
	if cmd.Flags().Changed("input-timeout") {
		inputTimeout = time.Duration(genInputTimeout) * time.Second
	} else if cfgTimeout, err := cfg.Interaction.GetInputTimeout(); err == nil {
		inputTimeout = cfgTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
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
		inputReader := bufio.NewReader(os.Stdin)
		var input string
		if len(args) == 0 {
			input, err = PromptForInput(inputReader, os.Stderr, "生成したいエージェントの要件を入力してください: ")
			if err != nil {
				return fmt.Errorf("入力取得エラー: %w", err)
			}
		} else {
			input = args[0]
		}

		// spec-gatherを実行
		gatherer := spec.NewGatherer(inputReader, os.Stderr)
		gatherer.SetMaxRounds(cfg.Gathering.MaxQuestionRounds)
		gatherer.SetConfidenceThreshold(cfg.Gathering.ConfidenceThreshold)
		gatherer.SetInputTimeout(inputTimeout)

		if verbose {
			fmt.Fprintf(os.Stderr, "仕様収集中: %s\n", input)
			fmt.Fprintf(os.Stderr, "全体タイムアウト: %s\n", totalTimeout)
			fmt.Fprintf(os.Stderr, "入力タイムアウト: %s\n", inputTimeout)
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

	generator := agent.NewGeneratorWithConfig(agent.GeneratorConfig{
		DefaultModel:        cfg.Generation.DefaultModel,
		DefaultTemperature:  cfg.Generation.DefaultTemperature,
		ConfidenceThreshold: cfg.Gathering.ConfidenceThreshold,
	})
	claudeDef, err := generator.Generate(ctx, agentSpec)
	if err != nil {
		return fmt.Errorf("エージェント定義生成エラー: %w", err)
	}

	formatter, err := outputfmt.NewFormatter(cfg.Output.Format)
	if err != nil {
		return fmt.Errorf("出力フォーマッタ作成エラー: %w", err)
	}

	rendered, err := renderAgentDefinition(generator, formatter, claudeDef)
	if err != nil {
		return err
	}

	// エージェント定義ファイルを生成
	outputPath := buildAgentOutputPathForFormat(outputDir, claudeDef.Metadata.Name, formatter.Name())
	if err := writeOutputFile(rendered, outputPath, cfg.Output.Overwrite); err != nil {
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

func renderAgentDefinition(generator *agent.Generator, formatter outputfmt.OutputFormatter, def *types.ClaudeAgentDefinition) ([]byte, error) {
	if formatter.Name() == "markdown" {
		markdown, err := generator.RenderMarkdown(def)
		if err != nil {
			return nil, fmt.Errorf("Markdownレンダリングエラー: %w", err)
		}
		return formatter.Format(markdown)
	}

	data, err := formatter.Format(def)
	if err != nil {
		return nil, fmt.Errorf("構造化出力エラー: %w", err)
	}
	return data, nil
}

// writeClaudeAgentMarkdown はClaude Code互換のMarkdownファイルを書き込みます。
func writeClaudeAgentMarkdown(markdown, path string, overwrite bool) error {
	return writeOutputFile([]byte(markdown), path, overwrite)
}

func writeOutputFile(data []byte, path string, overwrite bool) error {
	// 親ディレクトリを作成
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("親ディレクトリ作成エラー: %w", err)
		}
	}

	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file already exists and output.overwrite=false: %s", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check output file: %w", err)
		}
	}

	return os.WriteFile(path, data, 0644)
}

func buildAgentOutputPath(outputDir, name string) string {
	return buildAgentOutputPathForFormat(outputDir, name, "markdown")
}

func buildAgentOutputPathForFormat(outputDir, name, format string) string {
	filename := agent.MarkdownFileName(name) + ".md"
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		filename = agent.MarkdownFileName(name) + ".json"
	case "yaml":
		filename = agent.MarkdownFileName(name) + ".yaml"
	}
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
