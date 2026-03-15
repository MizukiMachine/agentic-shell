package spec

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildDynamicGatherPromptIncludesTranscript(t *testing.T) {
	state := NewConversationState("Refactor spec-gather to use LLM questions", nil)
	state.Confidence = 0.42
	state.AddTurn("Which behavior must stay compatible?", []string{"Fixed questions", "Output schema"}, "Fixed questions by default")

	prompt := BuildDynamicGatherPrompt(state)

	if !strings.Contains(prompt, "Refactor spec-gather to use LLM questions") {
		t.Fatal("expected initial input in prompt")
	}
	if !strings.Contains(prompt, "Current confidence: 0.42") {
		t.Fatal("expected confidence in prompt")
	}
	if !strings.Contains(prompt, "Turn 1 Question: Which behavior must stay compatible?") {
		t.Fatal("expected question transcript in prompt")
	}
	if !strings.Contains(prompt, "Turn 1 Answer: Fixed questions by default") {
		t.Fatal("expected answer transcript in prompt")
	}
}

func TestInterpreterProcessTurnUpdatesState(t *testing.T) {
	gatherer := NewGatherer(strings.NewReader(""), &bytes.Buffer{})
	spec := gatherer.buildInitialSpec("Refactor spec-gather to use LLM questions")
	state := NewConversationState("Refactor spec-gather to use LLM questions", spec)
	client := &mockLLMClient{
		responses: []TurnResponse{
			{
				CurrentUnderstanding: CurrentUnderstanding{
					CoreIntent:      "Enable optional LLM-driven follow-up questions",
					PrimaryGoal:     "Add an opt-in LLM mode with safe fallback",
					SuccessCriteria: []string{"LLM mode is opt-in", "Fallback uses fixed questions"},
				},
				Suggestions: []Suggestion{
					{Category: "Compatibility", Title: "Preserve defaults", Description: "Keep the existing fixed questions as the default path."},
				},
				NextQuestion: NextQuestion{
					Prompt:  "Which compatibility rule is most important?",
					Options: []string{"Default fixed questions", "Identical output schema"},
				},
			},
		},
	}

	interpreter := NewInterpreter(client)
	response, err := interpreter.ProcessTurn(context.Background(), state)
	if err != nil {
		t.Fatalf("ProcessTurn returned error: %v", err)
	}

	if response.NextQuestion.Prompt == "" {
		t.Fatal("expected next question prompt")
	}
	if state.LastResponse == nil {
		t.Fatal("expected state.LastResponse to be updated")
	}
	if state.AgentSpec.Intent.Goals.Primary.Main.Description != "Add an opt-in LLM mode with safe fallback" {
		t.Fatalf("unexpected primary goal: %q", state.AgentSpec.Intent.Goals.Primary.Main.Description)
	}
	if state.Confidence == 0 {
		t.Fatal("expected confidence to be recalculated")
	}
}

func TestGatherInteractiveWithLLM(t *testing.T) {
	answer := "Default flow must stay on fixed questions unless --llm is enabled, while the Go implementation keeps fallback behavior, build stability, JSON and YAML output, and automated tests."
	var output bytes.Buffer

	gatherer := NewGatherer(strings.NewReader(answer+"\n"), &output)
	gatherer.SetUseLLMQuestions(true)
	gatherer.SetMaxRounds(2)
	gatherer.SetInterpreter(NewInterpreter(&mockLLMClient{
		responses: []TurnResponse{
			{
				CurrentUnderstanding: CurrentUnderstanding{
					CoreIntent:      "Turn spec-gather into a dynamic interviewer",
					PrimaryGoal:     "Add optional LLM-generated follow-up questions with fallback",
					SuccessCriteria: []string{"The default mode stays unchanged", "LLM failures fall back safely"},
				},
				Suggestions: []Suggestion{
					{Category: "Compatibility", Title: "Keep the old path", Description: "Retain the fixed questions when LLM mode is disabled."},
				},
				NextQuestion: NextQuestion{
					Prompt:  "Which requirement is the hardest constraint?",
					Options: []string{"Backward compatibility", "Dynamic questioning", "Fallback safety"},
				},
			},
			{
				CurrentUnderstanding: CurrentUnderstanding{
					CoreIntent:      "Deliver a safe LLM-assisted gatherer refactor",
					PrimaryGoal:     "Support dynamic turns without breaking the fixed workflow",
					SuccessCriteria: []string{"`make build` passes", "`make test` passes", "Confidence reaches 90% or higher"},
				},
				Suggestions: []Suggestion{
					{Category: "Validation", Title: "Exercise both paths", Description: "Cover dynamic mode and fallback mode with tests."},
				},
				NextQuestion: NextQuestion{},
			},
		},
	}))

	spec, err := gatherer.GatherInteractive(context.Background(), "Refactor spec-gather to use dynamic LLM questions")
	if err != nil {
		t.Fatalf("GatherInteractive returned error: %v", err)
	}

	if spec.Intent.Metadata.Confidence < 0.90 {
		t.Fatalf("expected dynamic mode confidence >= 0.90, got %.2f", spec.Intent.Metadata.Confidence)
	}
	if !strings.Contains(output.String(), "=== Current Understanding ===") {
		t.Fatal("expected dynamic turn output")
	}
	if !strings.Contains(output.String(), "=== Question [1] ===") {
		t.Fatal("expected question numbering in output")
	}
}

func TestGatherInteractiveFallsBackWhenLLMFails(t *testing.T) {
	answers := strings.Join([]string{
		"The core problem is converting vague requests into implementation-ready specifications without breaking the existing workflow",
		"This matters because teams need a predictable path even when LLM generation fails in production workflows",
		"Prefer safe, reliable, reviewable changes over risky automation so fallback remains trustworthy",
		"The ideal solution is a Go CLI flow with JSON and YAML output, dynamic questions behind a flag, and strong unit test coverage",
		"It connects to broader objectives around reliable agent generation and maintainable specification tooling",
	}, "\n") + "\n"

	var output bytes.Buffer
	gatherer := NewGatherer(strings.NewReader(answers), &output)
	gatherer.SetUseLLMQuestions(true)
	gatherer.SetInterpreter(NewInterpreter(&mockLLMClient{
		executeJSONErr: fmt.Errorf("claude unavailable"),
	}))

	spec, err := gatherer.GatherInteractive(context.Background(), "Refactor spec-gather to use dynamic LLM questions")
	if err != nil {
		t.Fatalf("GatherInteractive returned error: %v", err)
	}

	if spec.Intent.Metadata.Confidence < 0.90 {
		t.Fatalf("expected fallback path to honor 0.90 confidence threshold, got %.2f", spec.Intent.Metadata.Confidence)
	}
	if !strings.Contains(output.String(), "LLM質問生成に失敗したため固定質問にフォールバックします") {
		t.Fatal("expected fallback notice in output")
	}
	if !strings.Contains(output.String(), "What is the core problem") {
		t.Fatal("expected fixed questions after fallback")
	}
}

type mockLLMClient struct {
	responses      []TurnResponse
	executeErr     error
	executeJSONErr error
	timeout        time.Duration
}

func (m *mockLLMClient) Execute(_ context.Context, _ string) (string, error) {
	return "", m.executeErr
}

func (m *mockLLMClient) ExecuteJSON(_ context.Context, _ string, target interface{}) error {
	if m.executeJSONErr != nil {
		return m.executeJSONErr
	}
	if len(m.responses) == 0 {
		return fmt.Errorf("no mock response configured")
	}

	response := m.responses[0]
	m.responses = m.responses[1:]

	typedTarget, ok := target.(*TurnResponse)
	if !ok {
		return fmt.Errorf("unexpected target type %T", target)
	}
	*typedTarget = response
	return nil
}

func (m *mockLLMClient) SetTimeout(d time.Duration) {
	m.timeout = d
}

func (m *mockLLMClient) GetTimeout() time.Duration {
	return m.timeout
}
