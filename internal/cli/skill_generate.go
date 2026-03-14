package cli

import (
	"errors"
	"fmt"

	skillpkg "github.com/MizukiMachine/agentic-shell/internal/skill"
	"github.com/spf13/cobra"
)

var (
	skillGenerateConfirm bool
	skillGenerateAuto    bool
)

var skillGenerateCmd = &cobra.Command{
	Use:   "skill-generate <spec-file>",
	Short: "AgentSpec から不足スキルを生成",
	Long: `AgentSpec を解析し、既存スキルとの差分から不足しているスキルの
プレースホルダーを .claude/skills 配下に生成します。`,
	Args: cobra.ExactArgs(1),
	RunE: runSkillGenerate,
}

func init() {
	rootCmd.AddCommand(skillGenerateCmd)

	skillGenerateCmd.Flags().BoolVar(&skillGenerateConfirm, "confirm", false, "生成前に確認プロンプトを表示")
	skillGenerateCmd.Flags().BoolVar(&skillGenerateAuto, "auto", false, "確認をスキップして自動生成")
}

func runSkillGenerate(cmd *cobra.Command, args []string) error {
	if skillGenerateConfirm && skillGenerateAuto {
		return fmt.Errorf("--confirm と --auto は同時に指定できません")
	}

	agentSpec, err := loadSpecFromFile(args[0])
	if err != nil {
		return fmt.Errorf("仕様ファイル読み込みエラー: %w", err)
	}

	generator := skillpkg.NewSkillGenerator(agentSpec, skillpkg.SkillGeneratorConfig{
		OutputDir: resolveSkillGenerateOutputDir(cmd),
		Confirm:   skillGenerateConfirm,
		Auto:      skillGenerateAuto,
		Input:     cmd.InOrStdin(),
		Output:    cmd.OutOrStdout(),
	})

	plan, err := generator.Generate()
	if err != nil {
		if errors.Is(err, skillpkg.ErrSkillGenerationDeclined) {
			return err
		}
		return fmt.Errorf("skill generation error: %w", err)
	}

	if len(plan.WrittenFiles) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "生成対象の不足スキルはありませんでした。")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "生成完了: %d files written to %s\n", len(plan.WrittenFiles), plan.OutputDir)
	return nil
}

func resolveSkillGenerateOutputDir(cmd *cobra.Command) string {
	if cmd != nil {
		if flag := cmd.Flags().Lookup("output-dir"); flag != nil && flag.Changed {
			return flag.Value.String()
		}
		if flag := cmd.PersistentFlags().Lookup("output-dir"); flag != nil && flag.Changed {
			return flag.Value.String()
		}
	}
	return skillpkg.DefaultSkillOutputDir
}
