package cli

import (
	"fmt"
	"strings"

	outputfmt "github.com/MizukiMachine/agentic-shell/internal/output"
	pluginpkg "github.com/MizukiMachine/agentic-shell/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	pluginDir string
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "プラグインを管理",
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "利用可能なプラグインを一覧表示",
	RunE:  runPluginList,
}

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)

	pluginListCmd.Flags().StringVar(&pluginDir, "dir", pluginpkg.DefaultPluginDir, "プラグイン探索ディレクトリ")
}

func runPluginList(cmd *cobra.Command, args []string) error {
	registry := pluginpkg.NewRegistry()
	if err := registry.Load(pluginDir); err != nil {
		return fmt.Errorf("plugin discovery failed: %w", err)
	}

	summaries := make([]pluginSummary, 0, len(registry.Plugins()))
	for _, plugin := range registry.Plugins() {
		summaries = append(summaries, pluginSummary{
			Name:            plugin.Name(),
			PromptBuilders:  pluginPromptBuilderNames(plugin),
			ToolInferencers: pluginToolInferencerNames(plugin),
		})
	}

	formatter, err := outputfmt.NewFormatter(GetConfig().Output.Format)
	if err != nil {
		return err
	}

	value := interface{}(summaries)
	if formatter.Name() == "markdown" {
		value = renderPluginMarkdown(summaries)
	}

	data, err := formatter.Format(value)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

type pluginSummary struct {
	Name            string   `json:"name" yaml:"name"`
	PromptBuilders  []string `json:"prompt_builders" yaml:"prompt_builders"`
	ToolInferencers []string `json:"tool_inferencers" yaml:"tool_inferencers"`
}

func pluginPromptBuilderNames(plugin pluginpkg.Plugin) []string {
	names := make([]string, 0, len(plugin.PromptBuilders()))
	for _, builder := range plugin.PromptBuilders() {
		names = append(names, builder.Name())
	}
	return names
}

func pluginToolInferencerNames(plugin pluginpkg.Plugin) []string {
	names := make([]string, 0, len(plugin.ToolInferencers()))
	for _, inferencer := range plugin.ToolInferencers() {
		names = append(names, inferencer.Name())
	}
	return names
}

func renderPluginMarkdown(plugins []pluginSummary) string {
	if len(plugins) == 0 {
		return "# Plugins\n\nNo plugins discovered."
	}

	lines := []string{"# Plugins", ""}
	for _, plugin := range plugins {
		lines = append(lines, fmt.Sprintf("- `%s`", plugin.Name))
		lines = append(lines, fmt.Sprintf("  prompt builders: %s", joinOrFallback(plugin.PromptBuilders)))
		lines = append(lines, fmt.Sprintf("  tool inferencers: %s", joinOrFallback(plugin.ToolInferencers)))
	}

	return strings.Join(lines, "\n")
}

func joinOrFallback(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}
