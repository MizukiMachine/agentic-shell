package spec

import (
	"fmt"
	"strings"
)

func Validate(spec *AgentSpec) error {
	if err := ValidateRequiredFields(spec); err != nil {
		return err
	}
	if err := spec.Validate(); err != nil {
		return fmt.Errorf("invalid agent spec: %w", err)
	}
	if err := ValidateConfidence(calculateConfidence(spec)); err != nil {
		return err
	}
	return nil
}

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

func ValidateConfidence(confidence float64) error {
	switch {
	case confidence < 0 || confidence > 1:
		return fmt.Errorf("confidence must be between 0 and 1, got %.2f", confidence)
	case confidence < ConfidenceThreshold:
		return fmt.Errorf("confidence %.2f is below threshold %.2f", confidence, ConfidenceThreshold)
	default:
		return nil
	}
}
