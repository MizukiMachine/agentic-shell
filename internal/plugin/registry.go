package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

const DefaultPluginDir = ".claude/plugins"

// Registry はプラグイン本体と拡張ポイント実装を管理します。
type Registry struct {
	plugins         map[string]Plugin
	promptBuilders  map[string]PromptBuilder
	toolInferencers map[string]ToolInferencer
}

// NewRegistry は新しい Registry を返します。
func NewRegistry() *Registry {
	return &Registry{
		plugins:         map[string]Plugin{},
		promptBuilders:  map[string]PromptBuilder{},
		toolInferencers: map[string]ToolInferencer{},
	}
}

// Register はプラグインを登録します。
func (r *Registry) Register(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("plugin is required")
	}

	pluginName := normalizeName(plugin.Name())
	if pluginName == "" {
		return fmt.Errorf("plugin name is required")
	}
	if _, exists := r.plugins[pluginName]; exists {
		return fmt.Errorf("plugin %q already registered", pluginName)
	}

	for _, builder := range plugin.PromptBuilders() {
		name := normalizeName(builder.Name())
		if name == "" {
			return fmt.Errorf("plugin %q has prompt builder with empty name", pluginName)
		}
		if _, exists := r.promptBuilders[name]; exists {
			return fmt.Errorf("prompt builder %q already registered", name)
		}
	}
	for _, inferencer := range plugin.ToolInferencers() {
		name := normalizeName(inferencer.Name())
		if name == "" {
			return fmt.Errorf("plugin %q has tool inferencer with empty name", pluginName)
		}
		if _, exists := r.toolInferencers[name]; exists {
			return fmt.Errorf("tool inferencer %q already registered", name)
		}
	}

	r.plugins[pluginName] = plugin
	for _, builder := range plugin.PromptBuilders() {
		r.promptBuilders[normalizeName(builder.Name())] = builder
	}
	for _, inferencer := range plugin.ToolInferencers() {
		r.toolInferencers[normalizeName(inferencer.Name())] = inferencer
	}

	return nil
}

// Load はディレクトリからプラグイン定義を発見・登録します。
func (r *Registry) Load(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read plugin dir: %w", err)
	}

	manifestPaths := make([]string, 0, len(entries))
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if manifestPath := findManifestFile(path); manifestPath != "" {
				manifestPaths = append(manifestPaths, manifestPath)
			}
			continue
		}
		if isManifestFile(entry.Name()) {
			manifestPaths = append(manifestPaths, path)
		}
	}
	sort.Strings(manifestPaths)

	for _, manifestPath := range manifestPaths {
		manifest, err := readManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("load plugin manifest %q: %w", manifestPath, err)
		}

		fallbackName := strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))
		if parent := filepath.Base(filepath.Dir(manifestPath)); parent != "." && parent != string(filepath.Separator) {
			fallbackName = parent
		}

		plugin, err := newManifestPlugin(manifest, fallbackName)
		if err != nil {
			return fmt.Errorf("create plugin from %q: %w", manifestPath, err)
		}
		if err := r.Register(plugin); err != nil {
			return fmt.Errorf("register plugin from %q: %w", manifestPath, err)
		}
	}

	return nil
}

// Plugins は登録済みプラグインを名前順で返します。
func (r *Registry) Plugins() []Plugin {
	if r == nil {
		return nil
	}

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	sort.Strings(names)

	plugins := make([]Plugin, 0, len(names))
	for _, name := range names {
		plugins = append(plugins, r.plugins[name])
	}
	return plugins
}

// PromptBuilder は名前で PromptBuilder を返します。
func (r *Registry) PromptBuilder(name string) (PromptBuilder, bool) {
	if r == nil {
		return nil, false
	}
	builder, ok := r.promptBuilders[normalizeName(name)]
	return builder, ok
}

// ToolInferencer は名前で ToolInferencer を返します。
func (r *Registry) ToolInferencer(name string) (ToolInferencer, bool) {
	if r == nil {
		return nil, false
	}
	inferencer, ok := r.toolInferencers[normalizeName(name)]
	return inferencer, ok
}

type manifestPlugin struct {
	name            string
	promptBuilders  []PromptBuilder
	toolInferencers []ToolInferencer
}

func (p *manifestPlugin) Name() string {
	return p.name
}

func (p *manifestPlugin) PromptBuilders() []PromptBuilder {
	return p.promptBuilders
}

func (p *manifestPlugin) ToolInferencers() []ToolInferencer {
	return p.toolInferencers
}

type templatePromptBuilder struct {
	name     string
	template *template.Template
}

func (b *templatePromptBuilder) Name() string {
	return b.name
}

func (b *templatePromptBuilder) Build(input string, context map[string]interface{}) (string, error) {
	data := map[string]interface{}{
		"Input":   input,
		"Context": context,
	}

	var buf bytes.Buffer
	if err := b.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute prompt builder %q: %w", b.name, err)
	}
	return buf.String(), nil
}

type keywordToolInferencer struct {
	name         string
	matchers     map[string][]string
	defaultTools []string
}

func (i *keywordToolInferencer) Name() string {
	return i.name
}

func (i *keywordToolInferencer) Infer(input string, _ map[string]interface{}) ([]string, error) {
	matched := []string{}
	lowerInput := strings.ToLower(input)

	for keyword, tools := range i.matchers {
		if strings.Contains(lowerInput, strings.ToLower(keyword)) {
			matched = append(matched, tools...)
		}
	}
	if len(matched) == 0 {
		matched = append(matched, i.defaultTools...)
	}

	return uniqueStrings(matched), nil
}

type manifest struct {
	Name            string                   `json:"name" yaml:"name"`
	PromptBuilders  []promptBuilderManifest  `json:"prompt_builders" yaml:"prompt_builders"`
	ToolInferencers []toolInferencerManifest `json:"tool_inferencers" yaml:"tool_inferencers"`
}

type promptBuilderManifest struct {
	Name     string `json:"name" yaml:"name"`
	Template string `json:"template" yaml:"template"`
}

type toolInferencerManifest struct {
	Name         string              `json:"name" yaml:"name"`
	Matchers     map[string][]string `json:"matchers" yaml:"matchers"`
	DefaultTools []string            `json:"default_tools" yaml:"default_tools"`
}

func newManifestPlugin(manifest manifest, fallbackName string) (Plugin, error) {
	pluginName := normalizeName(firstNonEmpty(manifest.Name, fallbackName))
	if pluginName == "" {
		return nil, fmt.Errorf("plugin name is required")
	}

	promptBuilders := make([]PromptBuilder, 0, len(manifest.PromptBuilders))
	for _, builderManifest := range manifest.PromptBuilders {
		name := normalizeName(builderManifest.Name)
		if name == "" {
			return nil, fmt.Errorf("plugin %q has prompt builder with empty name", pluginName)
		}

		builderTemplate := builderManifest.Template
		if strings.TrimSpace(builderTemplate) == "" {
			builderTemplate = "{{ .Input }}"
		}

		tmpl, err := template.New(name).Option("missingkey=zero").Parse(builderTemplate)
		if err != nil {
			return nil, fmt.Errorf("parse prompt builder template %q: %w", name, err)
		}
		promptBuilders = append(promptBuilders, &templatePromptBuilder{
			name:     name,
			template: tmpl,
		})
	}

	toolInferencers := make([]ToolInferencer, 0, len(manifest.ToolInferencers))
	for _, inferencerManifest := range manifest.ToolInferencers {
		name := normalizeName(inferencerManifest.Name)
		if name == "" {
			return nil, fmt.Errorf("plugin %q has tool inferencer with empty name", pluginName)
		}

		toolInferencers = append(toolInferencers, &keywordToolInferencer{
			name:         name,
			matchers:     inferencerManifest.Matchers,
			defaultTools: inferencerManifest.DefaultTools,
		})
	}

	return &manifestPlugin{
		name:            pluginName,
		promptBuilders:  promptBuilders,
		toolInferencers: toolInferencers,
	}, nil
}

func findManifestFile(dir string) string {
	for _, name := range []string{"plugin.yaml", "plugin.yml", "plugin.json"} {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func isManifestFile(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return base == "plugin.yaml" || base == "plugin.yml" || base == "plugin.json"
}

func readManifest(path string) (manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, err
	}

	var decoded manifest
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		if err := json.Unmarshal(data, &decoded); err != nil {
			return manifest{}, err
		}
	default:
		if err := yaml.Unmarshal(data, &decoded); err != nil {
			return manifest{}, err
		}
	}

	return decoded, nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
