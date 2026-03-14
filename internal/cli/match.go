package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	matchFrom      string
	matchOutput    string
	matchFormat    string
	matchStdinName string
	matchSkillsDir string
)

var matchCmd = &cobra.Command{
	Use:   "match [spec-files...]",
	Short: "必要スキルと既存スキルを照合",
	Long:  `抽出済みスキル要件と skill-scan の結果を突き合わせ、既存スキルで賄えるか判定します。`,
	RunE:  runMatch,
}

func init() {
	rootCmd.AddCommand(matchCmd)

	matchCmd.Flags().StringVar(&matchFrom, "from", "", "既存のパイプラインJSON/YAMLを読み込むファイル")
	matchCmd.Flags().StringVar(&matchOutput, "output", "", "出力ファイルパス (指定しない場合は標準出力)")
	matchCmd.Flags().StringVarP(&matchFormat, "format", "f", "json", "出力形式 (json または yaml)")
	matchCmd.Flags().StringVar(&matchStdinName, "stdin-name", "stdin.md", "標準入力を解析する際の仮想ファイル名")
	matchCmd.Flags().StringVar(&matchSkillsDir, "skills-dir", ".claude/skills", "走査対象スキルディレクトリ")
}

func runMatch(cmd *cobra.Command, args []string) error {
	env, raw, err := loadPipelineEnvelope(matchFrom)
	if err != nil {
		return fmt.Errorf("入力読み込みエラー: %w", err)
	}

	env, err = ensureMatchedEnvelope(env, raw, matchStdinName, args, matchSkillsDir)
	if err != nil {
		return fmt.Errorf("match エラー: %w", err)
	}

	return writeStructuredOutput(env, normalizeOutputFormat(matchFormat), matchOutput)
}
