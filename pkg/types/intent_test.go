// Package types provides core type definitions for the agentic-shell.
// This file contains tests for IntentSpace types.
package types

import (
	"encoding/json"
	"testing"
)

func TestGoalValidation(t *testing.T) {
	tests := []struct {
		name    string
		goal    Goal
		wantErr bool
	}{
		{
			name: "valid goal",
			goal: Goal{
				ID:          "goal-1",
				Type:        GoalTypePrimary,
				Description: "Test goal",
				Priority:    GoalPriorityHigh,
				Measurable:  true,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			goal: Goal{
				Type:        GoalTypePrimary,
				Description: "Test goal",
				Priority:    GoalPriorityHigh,
			},
			wantErr: true,
		},
		{
			name: "missing description",
			goal: Goal{
				ID:       "goal-1",
				Type:     GoalTypePrimary,
				Priority: GoalPriorityHigh,
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			goal: Goal{
				ID:          "goal-1",
				Type:        "invalid",
				Description: "Test goal",
				Priority:    GoalPriorityHigh,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.goal.Validate()
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

func TestIntentMetadataValidation(t *testing.T) {
	tests := []struct {
		name     string
		metadata IntentMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: IntentMetadata{
				IntentID:   "intent-1",
				Source:     IntentSourceUser,
				Confidence: 0.8,
				Version:    1,
			},
			wantErr: false,
		},
		{
			name: "missing intent ID",
			metadata: IntentMetadata{
				Source:     IntentSourceUser,
				Confidence: 0.8,
			},
			wantErr: true,
		},
		{
			name: "invalid confidence",
			metadata: IntentMetadata{
				IntentID:   "intent-1",
				Source:     IntentSourceUser,
				Confidence: 1.5,
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

func TestIntentSpaceJSONSerialization(t *testing.T) {
	intent := &IntentSpace{
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
				Bias:             QualitySpeedBiasQuality,
				QualityThreshold: 80,
				SpeedMultiplier:  1.0,
			},
		},
		Modality: ModalityDimension{
			Primary: OutputModalityCode,
			Code: &CodeModality{
				Language: "go",
				Style:    CodeStyleDocumented,
			},
		},
	}

	// Test JSON serialization
	data, err := intent.ToJSON()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Test JSON deserialization
	var decoded IntentSpace
	if err := decoded.FromJSON(data); err != nil {
		t.Fatalf("failed to deserialize: %v", err)
	}

	// Verify fields
	if decoded.Metadata.IntentID != intent.Metadata.IntentID {
		t.Errorf("intent ID mismatch: got %s, want %s", decoded.Metadata.IntentID, intent.Metadata.IntentID)
	}
	if decoded.Goals.Primary.Main.ID != intent.Goals.Primary.Main.ID {
		t.Errorf("goal ID mismatch: got %s, want %s", decoded.Goals.Primary.Main.ID, intent.Goals.Primary.Main.ID)
	}
}

func TestValidateIntent(t *testing.T) {
	intent := &IntentSpace{
		Metadata: IntentMetadata{
			IntentID:   "intent-1",
			Source:     IntentSourceUser,
			Confidence: 0.3, // Low confidence to trigger suggestion
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
			Primary: OutputModalityText,
		},
	}

	result := ValidateIntent(intent)

	if !result.Valid {
		t.Errorf("expected valid intent, got invalid: %v", result.Errors)
	}

	// Should have suggestions for low confidence
	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions for low confidence")
	}
}

func TestPreferencesValidation(t *testing.T) {
	tests := []struct {
		name    string
		pref    PreferencesDimension
		wantErr bool
	}{
		{
			name: "valid preferences",
			pref: PreferencesDimension{
				QualityVsSpeed: QualitySpeedPreference{
					Bias:             QualitySpeedBiasQuality,
					QualityThreshold: 80,
					SpeedMultiplier:  1.0,
				},
				CostVsPerformance: CostPerformancePreference{
					Bias:             CostPerformanceBiasBalanced,
					PerformanceFloor: 50,
				},
				AutomationVsControl: AutomationControlPreference{
					Bias: AutomationControlBiasSemiAuto,
				},
				Risk: RiskPreference{
					Tolerance: RiskToleranceModerate,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid quality threshold",
			pref: PreferencesDimension{
				QualityVsSpeed: QualitySpeedPreference{
					QualityThreshold: 150, // Invalid: > 100
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pref.Validate()
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

func TestModalityValidation(t *testing.T) {
	tests := []struct {
		name     string
		modality ModalityDimension
		wantErr  bool
	}{
		{
			name: "valid code modality",
			modality: ModalityDimension{
				Primary: OutputModalityCode,
				Code: &CodeModality{
					Language: "go",
					Style:    CodeStyleConcise,
				},
			},
			wantErr: false,
		},
		{
			name: "missing code language",
			modality: ModalityDimension{
				Primary: OutputModalityCode,
				Code: &CodeModality{
					Style: CodeStyleConcise,
				},
			},
			wantErr: true,
		},
		{
			name: "valid text modality",
			modality: ModalityDimension{
				Primary: OutputModalityText,
				Text: &TextModality{
					Format:   TextFormatMarkdown,
					Language: "en",
					Tone:     TextToneTechnical,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.modality.Validate()
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

func TestObjectivesValidation(t *testing.T) {
	obj := ObjectivesDimension{
		Functional: []FunctionalRequirement{
			{
				ID:                 "fr-1",
				Description:        "Functional requirement 1",
				Priority:           GoalPriorityCritical,
				AcceptanceCriteria: []string{"AC1", "AC2"},
				Testable:           true,
			},
		},
		NonFunctional: []NonFunctionalRequirement{
			{
				ID:          "nfr-1",
				Category:    NFCategoryPerformance,
				Description: "Response time under 100ms",
				Metric:      "latency_ms",
				Target:      100,
			},
		},
		Quality: []QualityRequirement{
			{
				ID:           "qr-1",
				Aspect:       QualityAspectTestCoverage,
				Description:  "Test coverage above 80%",
				MinimumScore: 80,
				TargetScore:  90,
				Mandatory:    true,
			},
		},
	}

	if err := obj.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIntentSpaceFullValidation(t *testing.T) {
	intent := &IntentSpace{
		Metadata: IntentMetadata{
			IntentID:   "intent-full-test",
			Source:     IntentSourceUser,
			Confidence: 0.85,
			Version:    1,
		},
		Goals: GoalsDimension{
			Primary: PrimaryGoals{
				Main: Goal{
					ID:          "goal-main",
					Type:        GoalTypePrimary,
					Description: "Primary goal for testing",
					Priority:    GoalPriorityCritical,
					Measurable:  true,
				},
				Supporting: []Goal{
					{
						ID:          "goal-support-1",
						Type:        GoalTypeSecondary,
						Description: "Supporting goal",
						Priority:    GoalPriorityMedium,
						Measurable:  true,
					},
				},
			},
			Secondary: SecondaryGoals{
				Goals: []Goal{
					{
						ID:          "goal-sec-1",
						Type:        GoalTypeSecondary,
						Description: "Secondary goal",
						Priority:    GoalPriorityLow,
						Measurable:  false,
					},
				},
			},
			Implicit: ImplicitGoals{
				Inferred: []Goal{
					{
						ID:          "goal-implicit-1",
						Type:        GoalTypeImplicit,
						Description: "Inferred goal",
						Priority:    GoalPriorityLow,
						Measurable:  false,
					},
				},
				Confidence: 0.7,
				Source:     "context",
			},
		},
		Preferences: PreferencesDimension{
			QualityVsSpeed: QualitySpeedPreference{
				Bias:             QualitySpeedBiasQuality,
				QualityThreshold: 90,
				SpeedMultiplier:  1.0,
				AllowDegradation: false,
			},
			CostVsPerformance: CostPerformancePreference{
				Bias:             CostPerformanceBiasPerformance,
				PerformanceFloor: 80,
				Elasticity:       1.5,
			},
			AutomationVsControl: AutomationControlPreference{
				Bias:                 AutomationControlBiasSemiAuto,
				ApprovalRequired:     []string{"deploy", "delete"},
				AutoApproveThreshold: 95,
			},
			Risk: RiskPreference{
				Tolerance:           RiskToleranceModerate,
				MaxRiskScore:        50,
				RequiresReviewAbove: 30,
			},
		},
		Objectives: ObjectivesDimension{
			Functional: []FunctionalRequirement{
				{
					ID:                 "func-1",
					Description:        "Core functionality",
					Priority:           GoalPriorityHigh,
					AcceptanceCriteria: []string{"Works correctly", "Handles errors"},
					Testable:           true,
				},
			},
			NonFunctional: []NonFunctionalRequirement{
				{
					ID:          "nf-1",
					Category:    NFCategoryReliability,
					Description: "99.9% uptime",
					Metric:      "availability_percent",
					Target:      99.9,
				},
			},
			Quality: []QualityRequirement{
				{
					ID:           "qual-1",
					Aspect:       QualityAspectCodeQuality,
					Description:  "Clean code standards",
					MinimumScore: 80,
					TargetScore:  95,
					Mandatory:    true,
				},
			},
			Constraints: []Constraint{
				{
					ID:          "const-1",
					Type:        ConstraintTypeTechnical,
					Description: "Must work on Linux",
					Impact:      ConstraintImpactBlocking,
				},
			},
		},
		Modality: ModalityDimension{
			Primary:   OutputModalityCode,
			Secondary: []OutputModality{OutputModalityText},
			Code: &CodeModality{
				Language:     "go",
				Framework:    "stdlib",
				Style:        CodeStyleDocumented,
				IncludeTests: true,
				IncludeTypes: true,
			},
			Text: &TextModality{
				Format:   TextFormatMarkdown,
				Language: "en",
				Tone:     TextToneTechnical,
			},
		},
	}

	// Validate
	if err := intent.Validate(); err != nil {
		t.Errorf("validation failed: %v", err)
	}

	// JSON round-trip
	data, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded IntentSpace
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify key fields
	if decoded.Metadata.IntentID != intent.Metadata.IntentID {
		t.Errorf("IntentID mismatch")
	}
	if decoded.Goals.Primary.Main.ID != intent.Goals.Primary.Main.ID {
		t.Errorf("Main goal ID mismatch")
	}
	if decoded.Modality.Code.Language != intent.Modality.Code.Language {
		t.Errorf("Code language mismatch")
	}
}
