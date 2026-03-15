package spec

import "strings"

// ConversationState は動的質問の対話状態を保持します。
type ConversationState struct {
	InitialInput string             `json:"initial_input"`
	Turns        []ConversationTurn `json:"turns"`
	Confidence   float64            `json:"confidence"`
	LastResponse *TurnResponse      `json:"last_response,omitempty"`
	AgentSpec    *AgentSpec         `json:"-"`
}

// ConversationTurn は 1 問 1 答の履歴です。
type ConversationTurn struct {
	Question string   `json:"question"`
	Options  []string `json:"options,omitempty"`
	Answer   string   `json:"answer"`
}

// NewConversationState は初期対話状態を生成します。
func NewConversationState(initialInput string, spec *AgentSpec) *ConversationState {
	return &ConversationState{
		InitialInput: strings.TrimSpace(initialInput),
		AgentSpec:    spec,
	}
}

// AddTurn は新しい対話履歴を追加します。
func (s *ConversationState) AddTurn(question string, options []string, answer string) {
	if s == nil {
		return
	}

	s.Turns = append(s.Turns, ConversationTurn{
		Question: strings.TrimSpace(question),
		Options:  trimNonEmptyStrings(options),
		Answer:   strings.TrimSpace(answer),
	})
}

// LastTurn は最後の対話履歴を返します。
func (s *ConversationState) LastTurn() *ConversationTurn {
	if s == nil || len(s.Turns) == 0 {
		return nil
	}
	return &s.Turns[len(s.Turns)-1]
}
