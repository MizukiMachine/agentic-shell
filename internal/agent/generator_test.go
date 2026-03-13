package agent

import (
	"context"
	"strings"
	"testing"

	types "github.com/MizukiMachine/agentic-shell/pkg/types"
)

func TestGeneratorGenerate(t *testing.T) {
	spec := validAgentSpec()

	def, err := NewGenerator().Generate(context.Background(), spec)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if def.Metadata.Name != spec.Metadata.Name {
		t.Fatalf("expected name %q, got %q", spec.Metadata.Name, def.Metadata.Name)
	}
	if def.Metadata.Category != "development" {
		t.Fatalf("expected category %q, got %q", "development", def.Metadata.Category)
	}
	if def.Prompt.CommunicationStyle != "technical" {
		t.Fatalf("expected communication style technical, got %q", def.Prompt.CommunicationStyle)
	}
	if !strings.Contains(def.Prompt.SystemPrompt, "## Mission") {
		t.Fatalf("expected system prompt to contain mission section, got:\n%s", def.Prompt.SystemPrompt)
	}

	names := toolNames(def.Tools)
	for _, required := range []string{"Read", "Grep", "Glob", "Write", "Edit", "MultiEdit", "Bash", "WebFetch"} {
		if !containsTool(names, required) {
			t.Fatalf("expected inferred tool %q in %v", required, names)
		}
	}
}

func TestGeneratorRenderMarkdown(t *testing.T) {
	spec := validAgentSpec()

	def, err := NewGenerator().Generate(context.Background(), spec)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	markdown, err := NewGenerator().RenderMarkdown(def)
	if err != nil {
		t.Fatalf("RenderMarkdown returned error: %v", err)
	}

	if !strings.Contains(markdown, "---\nname: \"code reviewer\"") {
		t.Fatalf("expected frontmatter name, got:\n%s", markdown)
	}
	if !strings.Contains(markdown, "tools:\n  - \"Read\"") {
		t.Fatalf("expected tools frontmatter, got:\n%s", markdown)
	}
	if !strings.Contains(markdown, "## Examples") {
		t.Fatalf("expected examples section, got:\n%s", markdown)
	}
	if !strings.Contains(markdown, "## Safety and Security") {
		t.Fatalf("expected system prompt body, got:\n%s", markdown)
	}
}

func TestMarkdownFileName(t *testing.T) {
	if got := MarkdownFileName("  Code Reviewer v2  "); got != "code-reviewer-v2" {
		t.Fatalf("expected sanitized file name, got %q", got)
	}
}

func validAgentSpec() *types.AgentSpec {
	spec := types.NewAgentSpec("code reviewer", "1.2.3")
	spec.Metadata.Author = "tester"
	spec.Metadata.Description = "Review code changes and produce actionable feedback."
	spec.Metadata.CreatedAt = "2026-03-13T00:00:00Z"
	spec.Metadata.Tags = []string{"review", "go"}
	spec.Capabilities = []types.Capability{
		{
			ID:          "cap-1",
			Name:        "Code Review",
			Description: "Inspect code, explain risks, and recommend fixes",
			Category:    "development",
			Level:       "expert",
			Keywords:    []string{"code", "review", "quality"},
		},
	}
	spec.Skills = []types.Skill{
		{
			ID:          "skill-1",
			Name:        "Static Analysis",
			Description: "Trace regressions and validate behavior changes",
			Examples:    []string{"Summarize the regression risk and propose a concrete patch."},
			Complexity:  "high",
		},
	}
	spec.BehaviorRules = []types.BehaviorRule{
		{
			ID:        "rule-1",
			Name:      "Escalate risky changes",
			Condition: "the change may break backward compatibility",
			Action:    "call out the risk before presenting lower-severity notes",
			Priority:  1,
			Enabled:   true,
		},
	}
	spec.KnowledgeSources = []types.KnowledgeSource{
		{
			ID:       "ks-1",
			Type:     "documentation",
			Name:     "Go documentation",
			URI:      "https://go.dev/doc/",
			Priority: 1,
		},
	}
	spec.Communication = types.CommunicationProtocol{
		Type:           "cli",
		Format:         "json",
		AllowedMethods: []string{"exec"},
	}
	spec.Performance = types.PerformanceConfig{
		MaxConcurrency: 1,
		Timeout:        60,
		RetryCount:     2,
		RetryDelay:     500,
		Priority:       "normal",
	}
	spec.Security = types.SecurityConfig{
		SandboxEnabled:     true,
		DataClassification: "internal",
		AuditEnabled:       true,
		EncryptionRequired: false,
		AllowedDomains:     []string{"go.dev"},
		AllowedCommands:    []string{"go test ./..."},
	}
	spec.Intent = types.IntentSpace{
		Metadata: types.IntentMetadata{
			IntentID:   "intent-123",
			Source:     types.IntentSourceUser,
			CreatedAt:  "2026-03-13T00:00:00Z",
			Confidence: 0.9,
			Version:    1,
		},
		Goals: types.GoalsDimension{
			Primary: types.PrimaryGoals{
				Main: types.Goal{
					ID:              "goal-1",
					Type:            types.GoalTypePrimary,
					Description:     "Review code changes and report the highest-value findings first.",
					Priority:        types.GoalPriorityHigh,
					Measurable:      true,
					SuccessCriteria: []string{"Findings are specific and actionable"},
				},
			},
			Secondary: types.SecondaryGoals{
				Goals: []types.Goal{},
			},
			Implicit: types.ImplicitGoals{
				Inferred:   []types.Goal{},
				Confidence: 0,
				Source:     "",
			},
			AllGoals: []types.Goal{
				{
					ID:          "goal-1",
					Type:        types.GoalTypePrimary,
					Description: "Review code changes and report the highest-value findings first.",
					Priority:    types.GoalPriorityHigh,
					Measurable:  true,
				},
			},
		},
		Preferences: types.PreferencesDimension{
			QualityVsSpeed: types.QualitySpeedPreference{
				Bias:             types.QualitySpeedBiasQuality,
				QualityThreshold: 85,
				SpeedMultiplier:  1,
			},
			CostVsPerformance: types.CostPerformancePreference{
				Bias:             types.CostPerformanceBiasBalanced,
				PerformanceFloor: 50,
				Elasticity:       1,
			},
			AutomationVsControl: types.AutomationControlPreference{
				Bias:                 types.AutomationControlBiasSemiAuto,
				ApprovalRequired:     []string{"destructive edits"},
				AutoApproveThreshold: 80,
			},
			Risk: types.RiskPreference{
				Tolerance:           types.RiskToleranceModerate,
				MaxRiskScore:        30,
				RequiresReviewAbove: 20,
			},
			CustomTradeOffs: []types.TradeOff{},
		},
		Objectives: types.ObjectivesDimension{
			Functional: []types.FunctionalRequirement{
				{
					ID:                 "fr-1",
					Description:        "Inspect diffs and identify correctness, reliability, and test coverage issues.",
					Priority:           types.GoalPriorityHigh,
					AcceptanceCriteria: []string{"Each finding includes impact and rationale"},
					Testable:           true,
				},
			},
			NonFunctional: []types.NonFunctionalRequirement{
				{
					ID:          "nfr-1",
					Category:    types.NFCategoryReliability,
					Description: "Keep the review deterministic and evidence-based.",
					Metric:      "false-positive rate",
					Target:      "low",
				},
			},
			Quality: []types.QualityRequirement{
				{
					ID:           "qr-1",
					Aspect:       types.QualityAspectCodeQuality,
					Description:  "Prioritize correctness over stylistic nitpicks.",
					MinimumScore: 80,
					TargetScore:  95,
					Mandatory:    true,
				},
			},
			Constraints: []types.Constraint{
				{
					ID:          "c-1",
					Type:        types.ConstraintTypeTechnical,
					Description: "Operate within the checked-out repository without external build systems.",
					Impact:      types.ConstraintImpactLimiting,
				},
			},
		},
		Modality: types.ModalityDimension{
			Primary:   types.OutputModalityCode,
			Secondary: []types.OutputModality{types.OutputModalityText},
			Code: &types.CodeModality{
				Language:     "go",
				Style:        types.CodeStyleDocumented,
				IncludeTests: true,
				IncludeTypes: true,
			},
			Text: &types.TextModality{
				Format:   types.TextFormatMarkdown,
				Language: "ja",
				Tone:     types.TextToneTechnical,
			},
		},
	}

	return spec
}

func containsTool(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
