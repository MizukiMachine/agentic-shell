package plugin

// PromptBuilder は入力からプラグイン独自のプロンプトを構築します。
type PromptBuilder interface {
	Name() string
	Build(input string, context map[string]interface{}) (string, error)
}

// ToolInferencer は入力から必要ツールを推定します。
type ToolInferencer interface {
	Name() string
	Infer(input string, context map[string]interface{}) ([]string, error)
}

// Plugin は PromptBuilder と ToolInferencer の集合です。
type Plugin interface {
	Name() string
	PromptBuilders() []PromptBuilder
	ToolInferencers() []ToolInferencer
}
