package spec

import "strings"

// TurnResponse は LLM が返す 1 ターン分の応答です。
type TurnResponse struct {
	CurrentUnderstanding CurrentUnderstanding `json:"current_understanding"`
	Suggestions          []Suggestion         `json:"suggestions"`
	NextQuestion         NextQuestion         `json:"next_question"`
}

// CurrentUnderstanding は現在の解釈を表します。
type CurrentUnderstanding struct {
	CoreIntent      string   `json:"core_intent"`
	PrimaryGoal     string   `json:"primary_goal"`
	SuccessCriteria []string `json:"success_criteria"`
}

// Suggestion は仕様整理のための提案です。
type Suggestion struct {
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// NextQuestion は次にユーザーへ確認する質問です。
type NextQuestion struct {
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

func (r *TurnResponse) normalize() {
	if r == nil {
		return
	}

	r.CurrentUnderstanding.CoreIntent = strings.TrimSpace(r.CurrentUnderstanding.CoreIntent)
	r.CurrentUnderstanding.PrimaryGoal = strings.TrimSpace(r.CurrentUnderstanding.PrimaryGoal)
	r.CurrentUnderstanding.SuccessCriteria = trimNonEmptyStrings(r.CurrentUnderstanding.SuccessCriteria)
	r.NextQuestion.Prompt = strings.TrimSpace(r.NextQuestion.Prompt)
	r.NextQuestion.Options = trimNonEmptyStrings(r.NextQuestion.Options)

	normalized := make([]Suggestion, 0, len(r.Suggestions))
	for _, suggestion := range r.Suggestions {
		suggestion.Category = strings.TrimSpace(suggestion.Category)
		suggestion.Title = strings.TrimSpace(suggestion.Title)
		suggestion.Description = strings.TrimSpace(suggestion.Description)
		if suggestion.Category == "" && suggestion.Title == "" && suggestion.Description == "" {
			continue
		}
		normalized = append(normalized, suggestion)
	}
	r.Suggestions = normalized
}

func trimNonEmptyStrings(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}
