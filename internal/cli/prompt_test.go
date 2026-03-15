package cli

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestPromptForInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("  feature request  \n"))
	var output bytes.Buffer

	got, err := PromptForInput(reader, &output, "input: ")
	if err != nil {
		t.Fatalf("PromptForInput returned error: %v", err)
	}

	if got != "feature request" {
		t.Fatalf("expected trimmed input, got %q", got)
	}
	if output.String() != "input: " {
		t.Fatalf("expected prompt to be written, got %q", output.String())
	}
}

func TestPromptForInputRejectsEmptyInput(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("   \n"))

	_, err := PromptForInput(reader, &bytes.Buffer{}, "input: ")
	if err == nil {
		t.Fatal("expected empty input error")
	}
}
