package llm

import (
	"fmt"
	"strings"
)

// PromptBuilder はLLM向けのプロンプトを構築します
type PromptBuilder struct {
	systemContext string
}

// NewPromptBuilder は新しいPromptBuilderを作成します
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// SetSystemContext はシステムコンテキストを設定します
func (pb *PromptBuilder) SetSystemContext(ctx string) *PromptBuilder {
	pb.systemContext = ctx
	return pb
}

// BuildSpecGatherPrompt は仕様収集用のプロンプトを構築します
// ユーザーの意図を理解し、必要な情報を収集するためのプロンプト
func (pb *PromptBuilder) BuildSpecGatherPrompt(userInput string, context *SpecGatherContext) string {
	var sb strings.Builder

	// システムコンテキスト
	if pb.systemContext != "" {
		sb.WriteString(fmt.Sprintf("# システムコンテキスト\n%s\n\n", pb.systemContext))
	}

	// 役割定義
	sb.WriteString(`# 役割
あなたは要件定義の専門家です。ユーザーの入力から以下の情報を抽出・整理してください。

`)

	// ユーザー入力
	sb.WriteString(fmt.Sprintf("# ユーザー入力\n%s\n\n", userInput))

	// 既存のコンテキスト情報
	if context != nil {
		if len(context.KnownIntents) > 0 {
			sb.WriteString("# 既知のインテント\n")
			for _, intent := range context.KnownIntents {
				sb.WriteString(fmt.Sprintf("- %s\n", intent))
			}
			sb.WriteString("\n")
		}

		if len(context.ExistingSpecs) > 0 {
			sb.WriteString("# 既存の仕様\n")
			for _, spec := range context.ExistingSpecs {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", spec.Name, spec.Description))
			}
			sb.WriteString("\n")
		}

		if len(context.Questions) > 0 {
			sb.WriteString("# 確認事項\n")
			for _, q := range context.Questions {
				sb.WriteString(fmt.Sprintf("- [ ] %s\n", q))
			}
			sb.WriteString("\n")
		}
	}

	// 出力形式
	sb.WriteString(`# 出力形式
以下のJSON形式で回答してください:

` + "```json" + `
{
  "understood_intent": "理解したユーザーの意図（簡潔に）",
  "key_requirements": ["主要な要件1", "主要な要件2"],
  "ambiguities": ["不明確な点1", "不明確な点2"],
  "follow_up_questions": ["確認すべき質問1", "確認すべき質問2"],
  "suggested_scope": "推奨するスコープ",
  "confidence_score": 0.8
}
` + "```" + `
`)

	return sb.String()
}

// SpecGatherContext は仕様収集のコンテキストです
type SpecGatherContext struct {
	KnownIntents   []string
	ExistingSpecs  []SpecInfo
	Questions      []string
	RelatedAgents  []string
}

// SpecInfo は仕様情報です
type SpecInfo struct {
	Name        string
	Description string
	Type        string
}

// BuildSpecRefinementPrompt は仕様洗練用のプロンプトを構築します
// 収集した情報を元に、詳細な仕様を策定するためのプロンプト
func (pb *PromptBuilder) BuildSpecRefinementPrompt(initialSpec *InitialSpec, feedback []string) string {
	var sb strings.Builder

	// システムコンテキスト
	if pb.systemContext != "" {
		sb.WriteString(fmt.Sprintf("# システムコンテキスト\n%s\n\n", pb.systemContext))
	}

	// 役割定義
	sb.WriteString(`# 役割
あなたはソフトウェアアーキテクトです。初期仕様を詳細な技術仕様に洗練させてください。

`)

	// 初期仕様
	if initialSpec != nil {
		sb.WriteString("# 初期仕様\n")
		sb.WriteString(fmt.Sprintf("- **意図**: %s\n", initialSpec.Intent))
		sb.WriteString(fmt.Sprintf("- **スコープ**: %s\n", initialSpec.Scope))
		sb.WriteString("- **主要要件**:\n")
		for _, req := range initialSpec.KeyRequirements {
			sb.WriteString(fmt.Sprintf("  - %s\n", req))
		}
		if len(initialSpec.Constraints) > 0 {
			sb.WriteString("- **制約事項**:\n")
			for _, c := range initialSpec.Constraints {
				sb.WriteString(fmt.Sprintf("  - %s\n", c))
			}
		}
		sb.WriteString("\n")
	}

	// フィードバック
	if len(feedback) > 0 {
		sb.WriteString("# フィードバック\n")
		for _, f := range feedback {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("\n")
	}

	// 出力形式
	sb.WriteString(`# 出力形式
以下のJSON形式で回答してください:

` + "```json" + `
{
  "refined_spec": {
    "name": "仕様名",
    "description": "詳細な説明",
    "components": [
      {
        "name": "コンポーネント名",
        "description": "説明",
        "interfaces": ["interface1", "interface2"],
        "dependencies": ["dep1", "dep2"]
      }
    ],
    "data_models": [
      {
        "name": "モデル名",
        "fields": [
          {"name": "field1", "type": "string", "required": true},
          {"name": "field2", "type": "int", "required": false}
        ]
      }
    ],
    "workflows": [
      {
        "name": "ワークフロー名",
        "steps": ["step1", "step2", "step3"]
      }
    ],
    "error_handling": {
      "strategies": ["retry", "fallback"],
      "logging_level": "info"
    },
    "testing_requirements": {
      "coverage_target": 80,
      "test_types": ["unit", "integration"]
    }
  },
  "implementation_notes": ["重要な実装ノート1", "重要な実装ノート2"],
  "risks": ["リスク1", "リスク2"]
}
` + "```" + `
`)

	return sb.String()
}

// InitialSpec は初期仕様です
type InitialSpec struct {
	Intent          string
	Scope           string
	KeyRequirements []string
	Constraints     []string
}

// BuildAgentDefinitionPrompt はエージェント定義用のプロンプトを構築します
// 仕様からエージェントの定義を生成するためのプロンプト
func (pb *PromptBuilder) BuildAgentDefinitionPrompt(spec *AgentSpecInput, constraints *AgentConstraints) string {
	var sb strings.Builder

	// システムコンテキスト
	if pb.systemContext != "" {
		sb.WriteString(fmt.Sprintf("# システムコンテキスト\n%s\n\n", pb.systemContext))
	}

	// 役割定義
	sb.WriteString(`# 役割
あなたはAIエージェント設計の専門家です。与えられた仕様から、効果的なAIエージェントの定義を生成してください。

`)

	// エージェント仕様入力
	if spec != nil {
		sb.WriteString("# エージェント仕様\n")
		sb.WriteString(fmt.Sprintf("- **名前**: %s\n", spec.Name))
		sb.WriteString(fmt.Sprintf("- **目的**: %s\n", spec.Purpose))
		sb.WriteString(fmt.Sprintf("- **役割**: %s\n", spec.Role))
		if len(spec.Responsibilities) > 0 {
			sb.WriteString("- **責任**:\n")
			for _, r := range spec.Responsibilities {
				sb.WriteString(fmt.Sprintf("  - %s\n", r))
			}
		}
		if len(spec.Tools) > 0 {
			sb.WriteString("- **使用ツール**:\n")
			for _, t := range spec.Tools {
				sb.WriteString(fmt.Sprintf("  - %s\n", t))
			}
		}
		sb.WriteString("\n")
	}

	// 制約条件
	if constraints != nil {
		sb.WriteString("# 制約条件\n")
		if constraints.MaxTokens > 0 {
			sb.WriteString(fmt.Sprintf("- **最大トークン数**: %d\n", constraints.MaxTokens))
		}
		if constraints.Model != "" {
			sb.WriteString(fmt.Sprintf("- **モデル**: %s\n", constraints.Model))
		}
		if len(constraints.AllowedActions) > 0 {
			sb.WriteString("- **許可アクション**:\n")
			for _, a := range constraints.AllowedActions {
				sb.WriteString(fmt.Sprintf("  - %s\n", a))
			}
		}
		if len(constraints.ForbiddenActions) > 0 {
			sb.WriteString("- **禁止アクション**:\n")
			for _, a := range constraints.ForbiddenActions {
				sb.WriteString(fmt.Sprintf("  - %s\n", a))
			}
		}
		sb.WriteString("\n")
	}

	// 出力形式
	sb.WriteString(`# 出力形式
以下のJSON形式でClaude Agent Definitionを出力してください:

` + "```json" + `
{
  "name": "エージェント名",
  "description": "エージェントの説明",
  "version": "1.0.0",
  "model": "claude-sonnet-4-6",
  "system_prompt": "システムプロンプトの内容",
  "tools": [
    {
      "name": "ツール名",
      "description": "ツールの説明",
      "input_schema": {
        "type": "object",
        "properties": {
          "param1": {"type": "string", "description": "パラメータ1"}
        },
        "required": ["param1"]
      }
    }
  ],
  "knowledge_bases": ["knowledge1", "knowledge2"],
  "execution_config": {
    "max_tokens": 4096,
    "temperature": 0.7,
    "timeout_seconds": 300
  },
  "error_handling": {
    "retry_count": 3,
    "fallback_strategy": "abort"
  },
  "metadata": {
    "author": "author-name",
    "tags": ["tag1", "tag2"]
  }
}
` + "```" + `
`)

	return sb.String()
}

// AgentSpecInput はエージェント仕様の入力です
type AgentSpecInput struct {
	Name            string
	Purpose         string
	Role            string
	Responsibilities []string
	Tools           []string
}

// AgentConstraints はエージェントの制約条件です
type AgentConstraints struct {
	MaxTokens         int
	Model             string
	AllowedActions    []string
	ForbiddenActions  []string
	TimeoutSeconds    int
}

// BuildPrompt は汎用的なプロンプトを構築します
func (pb *PromptBuilder) BuildPrompt(taskType string, input string, context map[string]interface{}) string {
	switch taskType {
	case "spec_gather":
		gatherCtx := &SpecGatherContext{}
		if ctx, ok := context["spec_context"].(*SpecGatherContext); ok {
			gatherCtx = ctx
		}
		return pb.BuildSpecGatherPrompt(input, gatherCtx)

	case "spec_refinement":
		initialSpec := &InitialSpec{}
		if spec, ok := context["initial_spec"].(*InitialSpec); ok {
			initialSpec = spec
		}
		feedback := []string{}
		if fb, ok := context["feedback"].([]string); ok {
			feedback = fb
		}
		return pb.BuildSpecRefinementPrompt(initialSpec, feedback)

	case "agent_definition":
		agentSpec := &AgentSpecInput{}
		if spec, ok := context["agent_spec"].(*AgentSpecInput); ok {
			agentSpec = spec
		}
		constraints := &AgentConstraints{}
		if c, ok := context["constraints"].(*AgentConstraints); ok {
			constraints = c
		}
		return pb.BuildAgentDefinitionPrompt(agentSpec, constraints)

	default:
		// 汎用プロンプト
		var sb strings.Builder
		if pb.systemContext != "" {
			sb.WriteString(fmt.Sprintf("# Context\n%s\n\n", pb.systemContext))
		}
		sb.WriteString(input)
		if len(context) > 0 {
			sb.WriteString("\n\n# Additional Context\n")
			for k, v := range context {
				sb.WriteString(fmt.Sprintf("- %s: %v\n", k, v))
			}
		}
		return sb.String()
	}
}

// PromptType はプロンプトタイプを表します
type PromptType string

const (
	PromptTypeSpecGather     PromptType = "spec_gather"
	PromptTypeSpecRefinement PromptType = "spec_refinement"
	PromptTypeAgentDef       PromptType = "agent_definition"
	PromptTypeGeneric        PromptType = "generic"
)
