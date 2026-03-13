// Package types provides core type definitions for the agentic-shell.
// This file contains tests for AgentSpec types.
package types

import (
	"strings"
	"testing"
)

func TestAgentSpecMetadataValidation(t *testing.T) {
	tests := []struct {
		name     string
		metadata AgentSpecMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: AgentSpecMetadata{
				Name:    "test-agent",
				Version: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			metadata: AgentSpecMetadata{
				Version: "1.0.0",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			metadata: AgentSpecMetadata{
				Name: "test-agent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCapabilityValidation(t *testing.T) {
	tests := []struct {
		name    string
		cap     Capability
		wantErr bool
	}{
		{
			name: "valid capability",
			cap: Capability{
				ID:          "cap-1",
				Name:        "Code Generation",
				Description: "Generate code from specifications",
				Category:    "development",
				Level:       "expert",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			cap: Capability{
				Name:        "Code Generation",
				Description: "Generate code from specifications",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			cap: Capability{
				ID:          "cap-1",
				Description: "Generate code from specifications",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cap.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSkillValidation(t *testing.T) {
	skill := Skill{
		ID:          "skill-1",
		Name:        "Go Programming",
		Description: "Write Go code",
		Complexity:  "medium",
	}

	if err := skill.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestToolValidation(t *testing.T) {
	tool := Tool{
		ID:          "tool-1",
		Name:        "File Writer",
		Description: "Write files to disk",
		Category:    "io",
		Parameters: []ToolParameter{
			{
				Name:        "path",
				Type:        "string",
				Description: "File path",
				Required:    true,
			},
			{
				Name:        "content",
				Type:        "string",
				Description: "File content",
				Required:    true,
			},
		},
		RiskLevel: "medium",
	}

	if err := tool.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestToolParameterValidation(t *testing.T) {
	tests := []struct {
		name    string
		param   ToolParameter
		wantErr bool
	}{
		{
			name: "valid parameter",
			param: ToolParameter{
				Name:        "path",
				Type:        "string",
				Description: "File path",
				Required:    true,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			param: ToolParameter{
				Type:        "string",
				Description: "File path",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			param: ToolParameter{
				Name:        "path",
				Description: "File path",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.param.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewAgentSpec(t *testing.T) {
	spec := NewAgentSpec("test-agent", "1.0.0")

	if spec.Metadata.Name != "test-agent" {
		t.Errorf("expected name 'test-agent', got '%s'", spec.Metadata.Name)
	}
	if spec.Metadata.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", spec.Metadata.Version)
	}

	// Check default values
	if spec.Communication.Type != "rest" {
		t.Errorf("expected default communication type 'rest', got '%s'", spec.Communication.Type)
	}
	if spec.Performance.MaxConcurrency != 1 {
		t.Errorf("expected default max_concurrency 1, got %d", spec.Performance.MaxConcurrency)
	}
	if !spec.Security.SandboxEnabled {
		t.Error("expected sandbox_enabled to be true by default")
	}
}

func TestAgentSpecJSONRoundTrip(t *testing.T) {
	spec := &AgentSpec{
		Metadata: AgentSpecMetadata{
			Name:        "json-test-agent",
			Version:     "2.0.0",
			Author:      "test-author",
			Description: "Test agent for JSON serialization",
			Tags:        []string{"test", "json"},
		},
		Intent: IntentSpace{
			Metadata: IntentMetadata{
				IntentID:   "intent-1",
				Source:     IntentSourceUser,
				Confidence: 0.9,
				Version:    1,
			},
			Goals: GoalsDimension{
				Primary: PrimaryGoals{
					Main: Goal{
						ID:          "goal-1",
						Type:        GoalTypePrimary,
						Description: "Main goal",
						Priority:    GoalPriorityHigh,
						Measurable:  true,
					},
				},
			},
			Preferences: PreferencesDimension{
				QualityVsSpeed: QualitySpeedPreference{
					Bias:             QualitySpeedBiasBalanced,
					QualityThreshold: 50,
				},
			},
			Modality: ModalityDimension{
				Primary: OutputModalityCode,
			},
		},
		Capabilities: []Capability{
			{
				ID:          "cap-1",
				Name:        "Testing",
				Description: "Run tests",
				Category:    "quality",
				Level:       "intermediate",
			},
		},
		Skills: []Skill{
			{
				ID:          "skill-1",
				Name:        "Go",
				Description: "Go programming",
				Complexity:  "high",
			},
		},
		Tools: []Tool{
			{
				ID:          "tool-1",
				Name:        "Shell",
				Description: "Execute shell commands",
				Category:    "execution",
				RiskLevel:   "high",
			},
		},
		Communication: CommunicationProtocol{
			Type:   "grpc",
			Format: "protobuf",
		},
		Performance: PerformanceConfig{
			MaxConcurrency: 5,
			Timeout:        60,
			Priority:       "high",
		},
		Security: SecurityConfig{
			SandboxEnabled:     true,
			DataClassification: "confidential",
		},
	}

	// Serialize to JSON
	data, err := spec.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Deserialize from JSON
	var decoded AgentSpec
	if err := decoded.FromJSON(data); err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	// Verify fields
	if decoded.Metadata.Name != spec.Metadata.Name {
		t.Errorf("name mismatch: got '%s', want '%s'", decoded.Metadata.Name, spec.Metadata.Name)
	}
	if len(decoded.Capabilities) != len(spec.Capabilities) {
		t.Errorf("capabilities count mismatch")
	}
	if decoded.Communication.Type != spec.Communication.Type {
		t.Errorf("communication type mismatch")
	}
}

func TestAgentSpecToYAML(t *testing.T) {
	spec := NewAgentSpec("yaml-test-agent", "1.0.0")
	spec.Metadata.Description = "YAML serialization test"
	spec.Intent = IntentSpace{
		Metadata: IntentMetadata{
			IntentID:   "intent-yaml",
			Source:     IntentSourceUser,
			Confidence: 0.9,
			Version:    1,
		},
		Goals: GoalsDimension{
			Primary: PrimaryGoals{
				Main: Goal{
					ID:          "goal-yaml",
					Type:        GoalTypePrimary,
					Description: "Serialize to YAML",
					Priority:    GoalPriorityHigh,
					Measurable:  true,
				},
			},
		},
		Preferences: PreferencesDimension{
			QualityVsSpeed: QualitySpeedPreference{
				SpeedMultiplier: 1.0,
			},
			CostVsPerformance: CostPerformancePreference{
				Elasticity: 1.0,
			},
		},
		Objectives: ObjectivesDimension{
			Functional: []FunctionalRequirement{
				{
					ID:          "fr-yaml",
					Description: "Support YAML output",
					Priority:    GoalPriorityHigh,
					Testable:    true,
				},
			},
		},
		Modality: ModalityDimension{
			Primary: OutputModalityData,
			Data: &DataModality{
				Format:     DataFormatYAML,
				Validation: true,
			},
		},
	}

	yaml, err := spec.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML returned error: %v", err)
	}
	if yaml == "" {
		t.Fatal("expected YAML output to be non-empty")
	}
	if !strings.Contains(yaml, "metadata:") || !strings.Contains(yaml, "name: \"yaml-test-agent\"") {
		t.Fatalf("unexpected YAML output: %s", yaml)
	}
}

func TestAgentSpecFullValidation(t *testing.T) {
	spec := &AgentSpec{
		Metadata: AgentSpecMetadata{
			Name:    "full-test-agent",
			Version: "1.0.0",
		},
		Intent: IntentSpace{
			Metadata: IntentMetadata{
				IntentID:   "intent-full",
				Source:     IntentSourceUser,
				Confidence: 0.8,
				Version:    1,
			},
			Goals: GoalsDimension{
				Primary: PrimaryGoals{
					Main: Goal{
						ID:          "goal-1",
						Type:        GoalTypePrimary,
						Description: "Main goal",
						Priority:    GoalPriorityHigh,
						Measurable:  true,
					},
				},
			},
			Preferences: PreferencesDimension{
				QualityVsSpeed: QualitySpeedPreference{
					Bias:             QualitySpeedBiasBalanced,
					QualityThreshold: 50,
					SpeedMultiplier:  1.0,
				},
			},
			Modality: ModalityDimension{
				Primary: OutputModalityCode,
			},
		},
		Capabilities: []Capability{
			{
				ID:          "cap-1",
				Name:        "Core",
				Description: "Core capability",
				Category:    "core",
				Level:       "expert",
			},
		},
		Skills: []Skill{
			{
				ID:          "skill-1",
				Name:        "Primary",
				Description: "Primary skill",
				Complexity:  "medium",
			},
		},
		Tools: []Tool{
			{
				ID:          "tool-1",
				Name:        "Basic",
				Description: "Basic tool",
				Category:    "utility",
				RiskLevel:   "low",
			},
		},
		BehaviorRules: []BehaviorRule{
			{
				ID:        "rule-1",
				Name:      "Safety check",
				Condition: "before_delete",
				Action:    "confirm",
				Priority:  100,
				Enabled:   true,
			},
		},
		KnowledgeSources: []KnowledgeSource{
			{
				ID:       "ks-1",
				Type:     "documentation",
				Name:     "API Docs",
				Priority: 1,
			},
		},
		Communication: CommunicationProtocol{
			Type:   "rest",
			Format: "json",
		},
		Performance: PerformanceConfig{
			MaxConcurrency: 2,
			Timeout:        30,
			Priority:       "normal",
		},
		Security: SecurityConfig{
			SandboxEnabled:     true,
			DataClassification: "internal",
		},
	}

	if err := spec.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}
}

func TestPerformanceConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  PerformanceConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: PerformanceConfig{
				MaxConcurrency: 5,
				Timeout:        30,
			},
			wantErr: false,
		},
		{
			name: "invalid concurrency",
			config: PerformanceConfig{
				MaxConcurrency: 0,
				Timeout:        30,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: PerformanceConfig{
				MaxConcurrency: 5,
				Timeout:        0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
