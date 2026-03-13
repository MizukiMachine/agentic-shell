package spec

import (
	"math"
	"strings"
	"unicode/utf8"
)

const ConfidenceThreshold = 0.85

type ConfidenceCalculator struct{}

func (c *ConfidenceCalculator) Calculate(spec *AgentSpec) float64 {
	if spec == nil {
		return 0
	}

	score := 0.0

	score += scoreMetadata(spec)
	score += scoreGoals(spec)
	score += scorePreferences(spec)
	score += scoreObjectives(spec)
	score += scoreExecution(spec)
	score += scoreSolution(spec)

	return math.Max(0, math.Min(1, score))
}

func calculateConfidence(spec *AgentSpec) float64 {
	return (&ConfidenceCalculator{}).Calculate(spec)
}

func scoreMetadata(spec *AgentSpec) float64 {
	score := 0.0
	if strings.TrimSpace(spec.Metadata.Name) != "" {
		score += 0.05
	}
	if isRichText(spec.Metadata.Description) {
		score += 0.08
	} else if strings.TrimSpace(spec.Metadata.Description) != "" {
		score += 0.04
	}
	if len(spec.Metadata.Tags) > 0 {
		score += 0.05
	}
	return score
}

func scoreGoals(spec *AgentSpec) float64 {
	score := 0.0
	mainGoal := spec.Intent.Goals.Primary.Main

	if isRichText(mainGoal.Description) {
		score += 0.12
	} else if strings.TrimSpace(mainGoal.Description) != "" {
		score += 0.06
	}
	if len(mainGoal.SuccessCriteria) > 0 {
		score += 0.06
	}
	if len(spec.Intent.Goals.Primary.Supporting) > 0 || len(spec.Intent.Goals.AllGoals) > 1 {
		score += 0.04
	}
	return score
}

func scorePreferences(spec *AgentSpec) float64 {
	score := 0.0
	prefs := spec.Intent.Preferences

	if len(prefs.CustomTradeOffs) > 0 {
		score += 0.08
	}
	if strings.TrimSpace(string(prefs.QualityVsSpeed.Bias)) != "" ||
		strings.TrimSpace(string(prefs.CostVsPerformance.Bias)) != "" {
		score += 0.04
	}
	if strings.TrimSpace(string(prefs.AutomationVsControl.Bias)) != "" ||
		len(prefs.AutomationVsControl.ApprovalRequired) > 0 {
		score += 0.02
	}
	if strings.TrimSpace(string(prefs.Risk.Tolerance)) != "" {
		score += 0.02
	}
	return score
}

func scoreObjectives(spec *AgentSpec) float64 {
	score := 0.0
	if len(spec.Intent.Objectives.Functional) > 0 {
		score += 0.10
	}
	if len(spec.Intent.Objectives.NonFunctional) > 0 {
		score += 0.05
	}
	if len(spec.Intent.Objectives.Quality) > 0 {
		score += 0.03
	}
	if len(spec.Intent.Objectives.Constraints) > 0 {
		score += 0.02
	}
	return score
}

func scoreExecution(spec *AgentSpec) float64 {
	score := 0.0
	if strings.TrimSpace(spec.Communication.Type) != "" &&
		strings.TrimSpace(spec.Communication.Format) != "" {
		score += 0.03
	}
	if spec.Performance.MaxConcurrency >= 1 && spec.Performance.Timeout >= 1 {
		score += 0.03
	}
	if strings.TrimSpace(spec.Security.DataClassification) != "" {
		score += 0.03
	}
	if strings.TrimSpace(string(spec.Intent.Modality.Primary)) != "" {
		score += 0.03
	}
	return score
}

func scoreSolution(spec *AgentSpec) float64 {
	score := 0.0
	if len(spec.Capabilities) > 0 {
		score += 0.05
	}
	if len(spec.Skills) > 0 {
		score += 0.04
	}
	if len(spec.Tools) > 0 {
		score += 0.03
	}
	return score
}

func isRichText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if len(strings.Fields(trimmed)) >= 6 {
		return true
	}
	return utf8.RuneCountInString(trimmed) >= 24
}
