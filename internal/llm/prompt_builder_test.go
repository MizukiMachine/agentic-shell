package llm

import (
	"strings"
	"testing"
)

func TestNewPromptBuilder(t *testing.T) {
	pb := NewPromptBuilder()
	if pb == nil {
		t.Error("NewPromptBuilder() returned nil")
	}
}

func TestSetSystemContext(t *testing.T) {
	pb := NewPromptBuilder()
	result := pb.SetSystemContext("test context")

	// メソッドチェーン確認
	if result != pb {
		t.Error("SetSystemContext should return *PromptBuilder for chaining")
	}

	if pb.systemContext != "test context" {
		t.Errorf("systemContext = %v, want 'test context'", pb.systemContext)
	}
}

func TestBuildSpecGatherPrompt(t *testing.T) {
	tests := []struct {
		name         string
		systemCtx    string
		userInput    string
		context      *SpecGatherContext
		wantContains []string
	}{
		{
			name:         "basic prompt without context",
			systemCtx:    "",
			userInput:    "ユーザー管理機能を作りたい",
			context:      nil,
			wantContains: []string{"ユーザー管理機能を作りたい", "understood_intent", "key_requirements", "JSON"},
		},
		{
			name:         "prompt with system context",
			systemCtx:    "あなたは専門のAIアシスタントです",
			userInput:    "APIを開発したい",
			context:      nil,
			wantContains: []string{"専門のAIアシスタントです", "APIを開発したい", "JSON"},
		},
		{
			name:      "prompt with full context",
			systemCtx: "",
			userInput: "ログイン機能を実装したい",
			context: &SpecGatherContext{
				KnownIntents: []string{"認証", "認可"},
				ExistingSpecs: []SpecInfo{
					{Name: "User", Description: "ユーザー情報", Type: "model"},
				},
				Questions: []string{"OAuth対応が必要ですか？"},
			},
			wantContains: []string{"認証", "認可", "User", "OAuth対応が必要ですか？", "JSON"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder()
			if tt.systemCtx != "" {
				pb.SetSystemContext(tt.systemCtx)
			}

			result := pb.BuildSpecGatherPrompt(tt.userInput, tt.context)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("prompt should contain %q", want)
				}
			}
		})
	}
}

func TestBuildSpecRefinementPrompt(t *testing.T) {
	tests := []struct {
		name         string
		initialSpec  *InitialSpec
		feedback     []string
		wantContains []string
	}{
		{
			name:         "nil spec",
			initialSpec:  nil,
			feedback:     nil,
			wantContains: []string{"ソフトウェアアーキテクト", "JSON", "refined_spec"},
		},
		{
			name: "with initial spec only",
			initialSpec: &InitialSpec{
				Intent:          "ユーザー登録機能",
				Scope:           "MVP",
				KeyRequirements: []string{"メール認証", "パスワードハッシュ"},
				Constraints:     []string{"GDPR対応"},
			},
			feedback: nil,
			wantContains: []string{"ユーザー登録機能", "MVP", "メール認証", "GDPR対応", "JSON"},
		},
		{
			name: "with spec and feedback",
			initialSpec: &InitialSpec{
				Intent:          "API開発",
				Scope:           "Phase 1",
				KeyRequirements: []string{"RESTful", "JSON"},
			},
			feedback: []string{"エラーハンドリングを強化してください", "バリデーションを追加してください"},
			wantContains: []string{"API開発", "エラーハンドリング", "バリデーション", "JSON"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder()
			result := pb.BuildSpecRefinementPrompt(tt.initialSpec, tt.feedback)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("prompt should contain %q", want)
				}
			}
		})
	}
}

func TestBuildAgentDefinitionPrompt(t *testing.T) {
	tests := []struct {
		name         string
		spec         *AgentSpecInput
		constraints  *AgentConstraints
		wantContains []string
	}{
		{
			name:         "nil spec and constraints",
			spec:         nil,
			constraints:  nil,
			wantContains: []string{"AIエージェント", "JSON", "system_prompt", "tools"},
		},
		{
			name: "with spec only",
			spec: &AgentSpecInput{
				Name:             "CodeReviewer",
				Purpose:          "コード品質の自動レビュー",
				Role:             "レビュアー",
				Responsibilities: []string{"静的解析", "バグ検出"},
				Tools:            []string{"Bash", "Read"},
			},
			constraints: nil,
			wantContains: []string{"CodeReviewer", "コード品質の自動レビュー", "静的解析", "Bash", "JSON"},
		},
		{
			name: "with spec and constraints",
			spec: &AgentSpecInput{
				Name:     "DeployAgent",
				Purpose:  "自動デプロイ",
				Role:     "デプロイヤー",
				Tools:    []string{"Bash", "Write"},
			},
			constraints: &AgentConstraints{
				MaxTokens:        4096,
				Model:            "claude-sonnet-4-6",
				AllowedActions:   []string{"deploy", "rollback"},
				ForbiddenActions: []string{"delete_all"},
			},
			wantContains: []string{"DeployAgent", "claude-sonnet-4-6", "deploy", "delete_all", "JSON"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder()
			result := pb.BuildAgentDefinitionPrompt(tt.spec, tt.constraints)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("prompt should contain %q", want)
				}
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	pb := NewPromptBuilder()

	t.Run("spec_gather type", func(t *testing.T) {
		result := pb.BuildPrompt("spec_gather", "test input", nil)
		if !strings.Contains(result, "test input") {
			t.Error("spec_gather prompt should contain user input")
		}
	})

	t.Run("spec_refinement type", func(t *testing.T) {
		result := pb.BuildPrompt("spec_refinement", "", map[string]interface{}{
			"initial_spec": &InitialSpec{Intent: "test intent"},
		})
		if !strings.Contains(result, "test intent") {
			t.Error("spec_refinement prompt should contain intent")
		}
	})

	t.Run("agent_definition type", func(t *testing.T) {
		result := pb.BuildPrompt("agent_definition", "", map[string]interface{}{
			"agent_spec": &AgentSpecInput{Name: "TestAgent"},
		})
		if !strings.Contains(result, "TestAgent") {
			t.Error("agent_definition prompt should contain agent name")
		}
	})

	t.Run("generic type", func(t *testing.T) {
		pb := NewPromptBuilder().SetSystemContext("system context")
		result := pb.BuildPrompt("unknown", "generic input", map[string]interface{}{
			"key": "value",
		})
		if !strings.Contains(result, "generic input") {
			t.Error("generic prompt should contain input")
		}
		if !strings.Contains(result, "system context") {
			t.Error("generic prompt should contain system context")
		}
	})
}

func TestPromptTypeConstants(t *testing.T) {
	tests := []struct {
		constant PromptType
		expected string
	}{
		{PromptTypeSpecGather, "spec_gather"},
		{PromptTypeSpecRefinement, "spec_refinement"},
		{PromptTypeAgentDef, "agent_definition"},
		{PromptTypeGeneric, "generic"},
	}

	for _, tt := range tests {
		t.Run(string(tt.constant), func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("constant = %v, want %v", tt.constant, tt.expected)
			}
		})
	}
}

func TestSpecGatherContext(t *testing.T) {
	ctx := &SpecGatherContext{
		KnownIntents:  []string{"intent1", "intent2"},
		ExistingSpecs: []SpecInfo{{Name: "spec1", Description: "desc1", Type: "type1"}},
		Questions:     []string{"q1", "q2"},
		RelatedAgents: []string{"agent1"},
	}

	if len(ctx.KnownIntents) != 2 {
		t.Errorf("KnownIntents length = %d, want 2", len(ctx.KnownIntents))
	}
	if len(ctx.ExistingSpecs) != 1 {
		t.Errorf("ExistingSpecs length = %d, want 1", len(ctx.ExistingSpecs))
	}
}

func TestInitialSpec(t *testing.T) {
	spec := &InitialSpec{
		Intent:          "test intent",
		Scope:           "MVP",
		KeyRequirements: []string{"req1", "req2"},
		Constraints:     []string{"const1"},
	}

	if spec.Intent != "test intent" {
		t.Errorf("Intent = %v, want 'test intent'", spec.Intent)
	}
	if len(spec.KeyRequirements) != 2 {
		t.Errorf("KeyRequirements length = %d, want 2", len(spec.KeyRequirements))
	}
}

func TestAgentSpecInput(t *testing.T) {
	spec := &AgentSpecInput{
		Name:             "TestAgent",
		Purpose:          "Testing",
		Role:             "Tester",
		Responsibilities: []string{"r1", "r2"},
		Tools:            []string{"tool1"},
	}

	if spec.Name != "TestAgent" {
		t.Errorf("Name = %v, want 'TestAgent'", spec.Name)
	}
	if len(spec.Responsibilities) != 2 {
		t.Errorf("Responsibilities length = %d, want 2", len(spec.Responsibilities))
	}
}

func TestAgentConstraints(t *testing.T) {
	constraints := &AgentConstraints{
		MaxTokens:        4096,
		Model:            "claude-sonnet-4-6",
		AllowedActions:   []string{"read", "write"},
		ForbiddenActions: []string{"delete"},
		TimeoutSeconds:   300,
	}

	if constraints.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %v, want 4096", constraints.MaxTokens)
	}
	if constraints.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %v, want 'claude-sonnet-4-6'", constraints.Model)
	}
}

// Benchmark tests
func BenchmarkBuildSpecGatherPrompt(b *testing.B) {
	pb := NewPromptBuilder()
	ctx := &SpecGatherContext{
		KnownIntents: []string{"intent1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pb.BuildSpecGatherPrompt("test input", ctx)
	}
}

func BenchmarkBuildSpecRefinementPrompt(b *testing.B) {
	pb := NewPromptBuilder()
	spec := &InitialSpec{
		Intent:          "test",
		KeyRequirements: []string{"r1", "r2"},
	}
	feedback := []string{"feedback1"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pb.BuildSpecRefinementPrompt(spec, feedback)
	}
}

func BenchmarkBuildAgentDefinitionPrompt(b *testing.B) {
	pb := NewPromptBuilder()
	spec := &AgentSpecInput{
		Name: "Test",
		Tools: []string{"t1"},
	}
	constraints := &AgentConstraints{
		MaxTokens: 4096,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pb.BuildAgentDefinitionPrompt(spec, constraints)
	}
}

// Integration test for prompt building workflow
func TestPromptBuildingWorkflow(t *testing.T) {
	pb := NewPromptBuilder().SetSystemContext("Test system context")

	// Step 1: Spec gathering
	gatherPrompt := pb.BuildPrompt("spec_gather", "Create a user authentication system", nil)
	if !strings.Contains(gatherPrompt, "authentication") {
		t.Error("gather prompt should mention authentication")
	}

	// Step 2: Spec refinement
	refinePrompt := pb.BuildPrompt("spec_refinement", "", map[string]interface{}{
		"initial_spec": &InitialSpec{
			Intent:          "Authentication",
			Scope:           "MVP",
			KeyRequirements: []string{"OAuth", "JWT"},
		},
	})
	if !strings.Contains(refinePrompt, "OAuth") {
		t.Error("refine prompt should mention OAuth")
	}

	// Step 3: Agent definition
	agentPrompt := pb.BuildPrompt("agent_definition", "", map[string]interface{}{
		"agent_spec": &AgentSpecInput{
			Name:    "AuthAgent",
			Purpose: "Handle authentication",
			Tools:   []string{"Bash", "Read"},
		},
		"constraints": &AgentConstraints{
			MaxTokens: 4096,
			Model:     "claude-sonnet-4-6",
		},
	})
	if !strings.Contains(agentPrompt, "AuthAgent") {
		t.Error("agent prompt should mention AuthAgent")
	}
}
