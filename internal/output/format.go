package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// OutputFormatter は値を任意の出力形式へ変換します。
type OutputFormatter interface {
	Name() string
	Extension() string
	Format(value interface{}) ([]byte, error)
}

// MarkdownMarshaler は Markdown へシリアライズ可能な型です。
type MarkdownMarshaler interface {
	MarshalMarkdown() ([]byte, error)
}

// NewFormatter は名前から OutputFormatter を返します。
func NewFormatter(name string) (OutputFormatter, error) {
	switch normalize(name) {
	case "json":
		return jsonFormatter{}, nil
	case "yaml":
		return yamlFormatter{}, nil
	case "markdown":
		return markdownFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", name)
	}
}

type jsonFormatter struct{}

func (jsonFormatter) Name() string {
	return "json"
}

func (jsonFormatter) Extension() string {
	return ".json"
}

func (jsonFormatter) Format(value interface{}) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}

type yamlFormatter struct{}

func (yamlFormatter) Name() string {
	return "yaml"
}

func (yamlFormatter) Extension() string {
	return ".yaml"
}

func (yamlFormatter) Format(value interface{}) ([]byte, error) {
	return yaml.Marshal(value)
}

type markdownFormatter struct{}

func (markdownFormatter) Name() string {
	return "markdown"
}

func (markdownFormatter) Extension() string {
	return ".md"
}

func (markdownFormatter) Format(value interface{}) ([]byte, error) {
	switch typed := value.(type) {
	case []byte:
		return typed, nil
	case string:
		return []byte(typed), nil
	case fmt.Stringer:
		return []byte(typed.String()), nil
	case MarkdownMarshaler:
		return typed.MarshalMarkdown()
	default:
		return nil, fmt.Errorf("value of type %T cannot be formatted as markdown", value)
	}
}

func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
