package cli

import (
	"fmt"

	"github.com/MizukiMachine/agentic-shell/internal/pipeline"
	"github.com/spf13/cobra"
)

var (
	parseFrom      string
	parseOutput    string
	parseFormat    string
	parseStdinName string
)

var parseCmd = &cobra.Command{
	Use:   "parse [spec-files...]",
	Short: "YAML/Markdown 仕様を構造化して出力",
	Long: `仕様ファイルを解析し、後続のパイプライン段で扱いやすい構造化データへ変換します。

使用例:
  ags parse spec.md
  cat spec.yaml | ags parse --stdin-name spec.yaml
  ags parse spec.md --output parsed.json --format json`,
	RunE: runParse,
}

func init() {
	rootCmd.AddCommand(parseCmd)

	parseCmd.Flags().StringVar(&parseFrom, "from", "", "既存のパイプラインJSON/YAMLを読み込むファイル")
	parseCmd.Flags().StringVar(&parseOutput, "output", "", "出力ファイルパス (指定しない場合は標準出力)")
	parseCmd.Flags().StringVarP(&parseFormat, "format", "f", "json", "出力形式 (json または yaml)")
	parseCmd.Flags().StringVar(&parseStdinName, "stdin-name", "stdin.md", "標準入力を解析する際の仮想ファイル名")
}

func runParse(cmd *cobra.Command, args []string) error {
	env, raw, err := loadPipelineEnvelope(parseFrom)
	if err != nil {
		return fmt.Errorf("入力読み込みエラー: %w", err)
	}
	if env == nil {
		env = &pipeline.Envelope{}
	}

	env, err = ensureParsedEnvelope(env, raw, parseStdinName, args)
	if err != nil {
		return fmt.Errorf("parse エラー: %w", err)
	}
	if len(env.Documents) == 0 {
		return fmt.Errorf("解析対象がありません")
	}

	return writeStructuredOutput(env, normalizeOutputFormat(parseFormat), parseOutput)
}
