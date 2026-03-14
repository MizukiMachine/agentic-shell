package cli

import (
	"fmt"

	"github.com/MizukiMachine/agentic-shell/internal/pipeline"
	"github.com/spf13/cobra"
)

var (
	skillScanFrom      string
	skillScanOutput    string
	skillScanFormat    string
	skillScanStdinName string
	skillScanDir       string
)

var skillScanCmd = &cobra.Command{
	Use:   "skill-scan [spec-files...]",
	Short: "既存スキルを .claude/skills/ から走査",
	Long:  `既存スキルを走査し、必要なら抽出済み要件と同じペイロードにマージして出力します。`,
	RunE:  runSkillScan,
}

func init() {
	rootCmd.AddCommand(skillScanCmd)

	skillScanCmd.Flags().StringVar(&skillScanFrom, "from", "", "既存のパイプラインJSON/YAMLを読み込むファイル")
	skillScanCmd.Flags().StringVar(&skillScanOutput, "output", "", "出力ファイルパス (指定しない場合は標準出力)")
	skillScanCmd.Flags().StringVarP(&skillScanFormat, "format", "f", "json", "出力形式 (json または yaml)")
	skillScanCmd.Flags().StringVar(&skillScanStdinName, "stdin-name", "stdin.md", "標準入力を解析する際の仮想ファイル名")
	skillScanCmd.Flags().StringVar(&skillScanDir, "skills-dir", ".claude/skills", "走査対象スキルディレクトリ")
}

func runSkillScan(cmd *cobra.Command, args []string) error {
	env, raw, err := loadPipelineEnvelope(skillScanFrom)
	if err != nil {
		return fmt.Errorf("入力読み込みエラー: %w", err)
	}

	if len(args) > 0 || len(raw) > 0 || (env != nil && len(env.Documents) > 0) {
		env, err = ensureScannedEnvelope(env, raw, skillScanStdinName, args, skillScanDir)
		if err != nil {
			return fmt.Errorf("skill-scan エラー: %w", err)
		}
	} else {
		if err := pipeline.ScanSkills(env, skillScanDir); err != nil {
			return fmt.Errorf("skill-scan エラー: %w", err)
		}
	}

	return writeStructuredOutput(env, normalizeOutputFormat(skillScanFormat), skillScanOutput)
}
