package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// PromptForInput は標準入力から1行読み取り、前後の空白を除去して返します。
func PromptForInput(reader *bufio.Reader, writer io.Writer, prompt string) (string, error) {
	if _, err := fmt.Fprint(writer, prompt); err != nil {
		return "", err
	}

	input, err := reader.ReadString('\n')
	if err != nil && len(input) == 0 {
		return "", err
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("入力が空です")
	}

	return input, nil
}
