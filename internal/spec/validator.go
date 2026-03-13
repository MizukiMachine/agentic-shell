package spec

import (
	"fmt"
	"strings"
)

// Validate は AgentSpec 全体を検証し、算出した信頼度を Metadata に反映します。
func Validate(spec *AgentSpec) error {
	return ValidateWithThreshold(spec, ConfidenceThreshold)
}

// ValidateWithThreshold は AgentSpec 全体を検証し、算出した信頼度を Metadata に反映します。
func ValidateWithThreshold(spec *AgentSpec, threshold float64) error {
	if err := ValidateRequiredFields(spec); err != nil {
		return err
	}
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("invalid agent spec: %w", err)
	}
	confidence := calculateConfidence(spec)
	spec.Intent.Metadata.Confidence = confidence
	if err := ValidateConfidenceWithThreshold(confidence, threshold); err != nil {
		return err
	}
	return nil
}

// ValidateRequiredFields は最小限必要な必須フィールドの有無を検証します。
func ValidateRequiredFields(spec *AgentSpec) error {
	if spec == nil {
		return fmt.Errorf("spec is required")
	}

	requiredFields := map[string]string{
		"metadata.name":                         spec.Metadata.Name,
		"metadata.version":                      spec.Metadata.Version,
		"metadata.description":                  spec.Metadata.Description,
		"intent.metadata.intent_id":             spec.Intent.Metadata.IntentID,
		"intent.goals.primary.main.id":          spec.Intent.Goals.Primary.Main.ID,
		"intent.goals.primary.main.description": spec.Intent.Goals.Primary.Main.Description,
		"communication.type":                    spec.Communication.Type,
		"communication.format":                  spec.Communication.Format,
		"security.data_classification":          spec.Security.DataClassification,
	}

	for field, value := range requiredFields {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("required field %s is missing", field)
		}
	}

	if len(spec.Intent.Objectives.Functional) == 0 {
		return fmt.Errorf("required field intent.objectives.functional is missing")
	}
	if strings.TrimSpace(string(spec.Intent.Modality.Primary)) == "" {
		return fmt.Errorf("required field intent.modality.primary is missing")
	}

	return nil
}

// ValidateConfidence は信頼度が許容範囲かつ閾値以上かを検証します。
func ValidateConfidence(confidence float64) error {
	return ValidateConfidenceWithThreshold(confidence, ConfidenceThreshold)
}

// ValidateConfidenceWithThreshold は信頼度が許容範囲かつ指定閾値以上かを検証します。
func ValidateConfidenceWithThreshold(confidence, threshold float64) error {
	switch {
	case confidence < 0 || confidence > 1:
		return fmt.Errorf("confidence must be between 0 and 1, got %.2f", confidence)
	case confidence < threshold:
		return fmt.Errorf("confidence %.2f is below threshold %.2f", confidence, threshold)
	default:
		return nil
	}
}
