package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	skillGenFrom      string
	skillGenOutput    string
	skillGenFormat    string
	skillGenStdinName string
	skillGenSkillsDir string
)

var skillGenCmd = &cobra.Command{
	Use:   "skill-gen [spec-files...]",
	Short: "不足スキルの生成計画を作成",
	Long:  `match の結果をもとに、不足スキル向けのプレースホルダーファイル計画を作成します。`,
	RunE:  runSkillGen,
}

func init() {
	rootCmd.AddCommand(skillGenCmd)

	skillGenCmd.Flags().StringVar(&skillGenFrom, "from", "", "既存のパイプラインJSON/YAMLを読み込むファイル")
	skillGenCmd.Flags().StringVar(&skillGenOutput, "output", "", "出力ファイルパス (指定しない場合は標準出力)")
	skillGenCmd.Flags().StringVarP(&skillGenFormat, "format", "f", "json", "出力形式 (json または yaml)")
	skillGenCmd.Flags().StringVar(&skillGenStdinName, "stdin-name", "stdin.md", "標準入力を解析する際の仮想ファイル名")
	skillGenCmd.Flags().StringVar(&skillGenSkillsDir, "skills-dir", ".claude/skills", "走査対象スキルディレクトリ")
}

func runSkillGen(cmd *cobra.Command, args []string) error {
	env, raw, err := loadPipelineEnvelope(skillGenFrom)
	if err != nil {
		return fmt.Errorf("入力読み込みエラー: %w", err)
	}

	env, err = ensureGeneratedEnvelope(env, raw, skillGenStdinName, args, skillGenSkillsDir)
	if err != nil {
		return fmt.Errorf("skill-gen エラー: %w", err)
	}

	return writeStructuredOutput(env, normalizeOutputFormat(skillGenFormat), skillGenOutput)
}
