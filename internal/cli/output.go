package cli

import (
	"fmt"

	"github.com/MizukiMachine/agentic-shell/internal/pipeline"
	"github.com/spf13/cobra"
)

var (
	outputFrom      string
	outputFile      string
	outputFormat    string
	outputStdinName string
	outputSkillsDir string
)

var outputCmd = &cobra.Command{
	Use:   "output [spec-files...]",
	Short: "生成結果を指定先へ書き出す",
	Long:  `skill-gen まで進んだパイプライン結果をファイルへ書き出し、書き込み結果を構造化出力します。`,
	RunE:  runOutput,
}

func init() {
	rootCmd.AddCommand(outputCmd)

	outputCmd.Flags().StringVar(&outputFrom, "from", "", "既存のパイプラインJSON/YAMLを読み込むファイル")
	outputCmd.Flags().StringVar(&outputFile, "output", "", "書き込み結果の出力ファイルパス (指定しない場合は標準出力)")
	outputCmd.Flags().StringVarP(&outputFormat, "format", "f", "json", "書き込み結果の出力形式 (json または yaml)")
	outputCmd.Flags().StringVar(&outputStdinName, "stdin-name", "stdin.md", "標準入力を解析する際の仮想ファイル名")
	outputCmd.Flags().StringVar(&outputSkillsDir, "skills-dir", ".claude/skills", "走査対象スキルディレクトリ")
}

func runOutput(cmd *cobra.Command, args []string) error {
	env, raw, err := loadPipelineEnvelope(outputFrom)
	if err != nil {
		return fmt.Errorf("入力読み込みエラー: %w", err)
	}

	env, err = ensureGeneratedEnvelope(env, raw, outputStdinName, args, outputSkillsDir)
	if err != nil {
		return fmt.Errorf("output エラー: %w", err)
	}

	if err := pipeline.WriteGeneratedFiles(env, GetOutputDir(), outputSkillsDir, GetConfig().Output.Overwrite); err != nil {
		return fmt.Errorf("ファイル出力エラー: %w", err)
	}

	return writeStructuredOutput(env, normalizeOutputFormat(outputFormat), outputFile)
}
