package spec

import (
	"context"
	"fmt"
	"strings"

	"github.com/MizukiMachine/agentic-shell/internal/llm"
)

// Interpreter は LLM を使って現在の対話を解釈します。
type Interpreter struct {
	client     llm.Client
	calculator *ConfidenceCalculator
	mutator    *Gatherer
}

// NewInterpreter は Interpreter を生成します。
func NewInterpreter(client llm.Client) *Interpreter {
	return &Interpreter{
		client:     client,
		calculator: &ConfidenceCalculator{},
		mutator:    NewGatherer(nil, nil),
	}
}

// ProcessTurn は対話履歴を元に次ターン用の応答を生成します。
func (i *Interpreter) ProcessTurn(ctx context.Context, state *ConversationState) (*TurnResponse, error) {
	if state == nil {
		return nil, fmt.Errorf("conversation state is required")
	}
	if state.AgentSpec == nil {
		return nil, fmt.Errorf("agent spec is required")
	}
	if i == nil || i.client == nil {
		return nil, fmt.Errorf("llm client is required")
	}

	prompt := BuildDynamicGatherPrompt(state)
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("dynamic gather prompt is empty")
	}

	var response TurnResponse
	if err := i.client.ExecuteJSON(ctx, prompt, &response); err != nil {
		return nil, fmt.Errorf("execute dynamic gather prompt: %w", err)
	}

	response.normalize()
	if err := i.applyTurnResponse(state.AgentSpec, &response); err != nil {
		return nil, err
	}

	state.Confidence = i.calculator.Calculate(state.AgentSpec)
	state.AgentSpec.Intent.Metadata.Confidence = state.Confidence
	state.LastResponse = &response

	return &response, nil
}

func (i *Interpreter) applyTurnResponse(spec *AgentSpec, response *TurnResponse) error {
	if spec == nil {
		return fmt.Errorf("agent spec is required")
	}
	if response == nil {
		return fmt.Errorf("turn response is required")
	}

	coreProblem := response.CurrentUnderstanding.PrimaryGoal
	if coreProblem == "" {
		coreProblem = response.CurrentUnderstanding.CoreIntent
	}
	if coreProblem != "" {
		i.mutator.applyCoreProblem(spec, coreProblem)
	}

	if response.CurrentUnderstanding.CoreIntent != "" {
		spec.Metadata.Description = mergeNarrative(
			spec.Metadata.Description,
			"Core intent: "+response.CurrentUnderstanding.CoreIntent,
		)
	}

	for _, criterion := range response.CurrentUnderstanding.SuccessCriteria {
		spec.Intent.Goals.Primary.Main.SuccessCriteria = appendUnique(
			spec.Intent.Goals.Primary.Main.SuccessCriteria,
			criterion,
		)
		if len(spec.Intent.Objectives.Functional) > 0 {
			spec.Intent.Objectives.Functional[0].AcceptanceCriteria = appendUnique(
				spec.Intent.Objectives.Functional[0].AcceptanceCriteria,
				criterion,
			)
		}
	}
	i.mutator.syncPrimaryMainGoal(spec)

	for _, suggestion := range response.Suggestions {
		description := suggestion.Description
		switch {
		case description == "" && suggestion.Title != "":
			description = suggestion.Title
		case description == "":
			description = suggestion.Category
		}
		if strings.TrimSpace(description) == "" {
			continue
		}

		spec.Metadata.Description = mergeNarrative(
			spec.Metadata.Description,
			fmt.Sprintf("Suggestion [%s]: %s", nonEmpty(suggestion.Category, "general"), description),
		)
		spec.Metadata.Tags = appendUnique(spec.Metadata.Tags, extractKeywords(description)...)
	}

	return nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
