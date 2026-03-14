package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	extractFrom      string
	extractOutput    string
	extractFormat    string
	extractStdinName string
)

var extractCmd = &cobra.Command{
	Use:   "extract [spec-files...]",
	Short: "構造化仕様から意図と要件を抽出",
	Long:  `parse 済みデータ、または YAML/Markdown 仕様ファイルから、意図・要件・必要スキルを抽出します。`,
	RunE:  runExtract,
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().StringVar(&extractFrom, "from", "", "既存のパイプラインJSON/YAMLを読み込むファイル")
	extractCmd.Flags().StringVar(&extractOutput, "output", "", "出力ファイルパス (指定しない場合は標準出力)")
	extractCmd.Flags().StringVarP(&extractFormat, "format", "f", "json", "出力形式 (json または yaml)")
	extractCmd.Flags().StringVar(&extractStdinName, "stdin-name", "stdin.md", "標準入力を解析する際の仮想ファイル名")
}

func runExtract(cmd *cobra.Command, args []string) error {
	env, raw, err := loadPipelineEnvelope(extractFrom)
	if err != nil {
		return fmt.Errorf("入力読み込みエラー: %w", err)
	}

	env, err = ensureExtractedEnvelope(env, raw, extractStdinName, args)
	if err != nil {
		return fmt.Errorf("extract エラー: %w", err)
	}

	return writeStructuredOutput(env, normalizeOutputFormat(extractFormat), extractOutput)
}
