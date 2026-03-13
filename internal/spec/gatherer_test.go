package spec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestGenerateStepBackQuestions(t *testing.T) {
	questions := generateStepBackQuestions("implement an interactive gatherer")

	if len(questions) != 5 {
		t.Fatalf("expected 5 questions, got %d", len(questions))
	}

	if !strings.Contains(questions[0], "interactive gatherer") {
		t.Fatalf("unexpected first question: %q", questions[0])
	}
}

func TestCalculateConfidence(t *testing.T) {
	gatherer := NewGatherer(strings.NewReader(""), &bytes.Buffer{})
	spec := gatherer.buildInitialSpec("Implement interactive specification gathering")

	initial := calculateConfidence(spec)
	if initial >= ConfidenceThreshold {
		t.Fatalf("expected initial confidence below threshold, got %.2f", initial)
	}

	gatherer.applyCoreProblem(spec, "We need a concrete problem statement for implementation")
	gatherer.applyBiggerPicture(spec, "This keeps the agent authoring workflow aligned across the project")
	gatherer.applyPrinciples(spec, "Prefer quality, safety, human review, and predictable validation")
	gatherer.applyIdealSolution(spec, "Ideal solution provides interactive prompts plus JSON and YAML export")
	gatherer.applyBroaderObjectives(spec, "It should support broader automation and reusable agent specifications")

	final := calculateConfidence(spec)
	if final < ConfidenceThreshold {
		t.Fatalf("expected confidence above threshold, got %.2f", final)
	}
}

func TestValidateConfidence(t *testing.T) {
	if err := ValidateConfidence(0.90); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := ValidateConfidence(0.80); err == nil {
		t.Fatal("expected threshold validation error")
	}
}

func TestValidateSynchronizesConfidence(t *testing.T) {
	gatherer := NewGatherer(strings.NewReader(""), &bytes.Buffer{})
	spec := gatherer.buildInitialSpec("Implement interactive specification gathering")

	gatherer.applyCoreProblem(spec, "We need a concrete problem statement for implementation")
	gatherer.applyBiggerPicture(spec, "This keeps the workflow aligned across the project")
	gatherer.applyPrinciples(spec, "Prefer quality, safety, human review, and predictable validation")
	gatherer.applyIdealSolution(spec, "The solution should provide interactive prompts plus JSON and YAML export")
	gatherer.applyBroaderObjectives(spec, "It should support broader automation and reusable agent specifications")

	spec.Intent.Metadata.Confidence = 0.01

	if err := Validate(spec); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	expected := calculateConfidence(spec)
	if spec.Intent.Metadata.Confidence != expected {
		t.Fatalf("confidence mismatch: got %.2f, want %.2f", spec.Intent.Metadata.Confidence, expected)
	}
}

func TestGatherInteractive(t *testing.T) {
	answers := strings.Join([]string{
		"The core problem is turning vague requests into implementation-ready agent specifications",
		"This matters because the broader workflow needs reliable, repeatable requirements capture",
		"Prefer quality, safety, human review, and explicit validation over raw speed",
		"The ideal solution is an interactive Go component that emits JSON and YAML outputs with validation",
		"It connects to the broader objective of reusable agent definitions and automation",
	}, "\n") + "\n"

	var output bytes.Buffer
	gatherer := NewGatherer(strings.NewReader(answers), &output)

	spec, err := gatherer.GatherInteractive(context.Background(), "Implement the spec gatherer feature")
	if err != nil {
		t.Fatalf("GatherInteractive returned error: %v", err)
	}

	if err := Validate(spec); err != nil {
		t.Fatalf("returned spec should be valid: %v", err)
	}
	if spec.Intent.Metadata.Confidence < ConfidenceThreshold {
		t.Fatalf("expected confidence >= %.2f, got %.2f", ConfidenceThreshold, spec.Intent.Metadata.Confidence)
	}
	if len(spec.Capabilities) == 0 || len(spec.Skills) == 0 || len(spec.Tools) == 0 {
		t.Fatal("expected solution details to be populated")
	}
	if !strings.Contains(output.String(), "What is the core problem") {
		t.Fatal("expected interactive questions to be written to output")
	}
}

func TestGatherInteractiveMaxRounds(t *testing.T) {
	answers := "\n\n\n\n\n"
	gatherer := NewGatherer(strings.NewReader(answers), &bytes.Buffer{})

	_, err := gatherer.GatherInteractive(context.Background(), "Implement the spec gatherer feature")
	if err == nil {
		t.Fatal("expected confidence error after max rounds")
	}
	if !strings.Contains(err.Error(), "below threshold") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGatherInteractiveContextCancel(t *testing.T) {
	reader := &blockingReader{unblock: make(chan struct{})}
	gatherer := NewGatherer(reader, &bytes.Buffer{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		_, err := gatherer.GatherInteractive(ctx, "Implement the spec gatherer feature")
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		close(reader.unblock)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		close(reader.unblock)
		t.Fatal("GatherInteractive did not return promptly after cancellation")
	}
}

func TestGatherInteractiveEOFError(t *testing.T) {
	gatherer := NewGatherer(strings.NewReader(""), &bytes.Buffer{})

	_, err := gatherer.GatherInteractive(context.Background(), "Implement the spec gatherer feature")
	if err == nil {
		t.Fatal("expected EOF error")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "unexpectedly") {
		t.Fatalf("expected contextual EOF message, got: %v", err)
	}
}

func TestApplyCoreProblemSynchronizesAllGoals(t *testing.T) {
	gatherer := NewGatherer(strings.NewReader(""), &bytes.Buffer{})
	spec := gatherer.buildInitialSpec("Implement interactive specification gathering")

	gatherer.applyCoreProblem(spec, "Document the exact problem before coding starts")

	if len(spec.Intent.Goals.AllGoals) == 0 {
		t.Fatal("expected all goals to contain the primary goal")
	}
	if spec.Intent.Goals.AllGoals[0].Description != spec.Intent.Goals.Primary.Main.Description {
		t.Fatalf("all goals main description mismatch: got %q, want %q", spec.Intent.Goals.AllGoals[0].Description, spec.Intent.Goals.Primary.Main.Description)
	}
	if len(spec.Intent.Goals.AllGoals[0].SuccessCriteria) != len(spec.Intent.Goals.Primary.Main.SuccessCriteria) {
		t.Fatal("expected success criteria to stay synchronized")
	}
}

func TestContainsAnyUsesWordBoundaries(t *testing.T) {
	if containsAny("ongoing goal refinement", "go") {
		t.Fatal("expected partial word match to be ignored")
	}
	if !containsAny("the go package should be documented", "go") {
		t.Fatal("expected exact word match to succeed")
	}
	if !containsAny("command line workflow", "command line") {
		t.Fatal("expected phrase match to succeed")
	}
}

type blockingReader struct {
	unblock chan struct{}
}

func (r *blockingReader) Read(_ []byte) (int, error) {
	<-r.unblock
	return 0, io.EOF
}
