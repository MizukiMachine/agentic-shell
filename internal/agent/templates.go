package agent

import (
	"strconv"
	"strings"
	"text/template"

	types "github.com/MizukiMachine/agentic-shell/pkg/types"
)

type markdownTemplateData struct {
	Name         string
	Description  string
	Model        string
	Tools        []string
	SystemPrompt string
	Examples     []types.PromptExample
}

var agentMarkdownTemplate = template.Must(template.New("agent-markdown").Funcs(template.FuncMap{
	"yamlString": strconv.Quote,
	"trimSpace":  strings.TrimSpace,
}).Parse(`---
name: {{ yamlString .Name }}
description: {{ yamlString .Description }}
{{- if .Model }}
model: {{ yamlString .Model }}
{{- end }}
{{- if .Tools }}
tools:
{{- range .Tools }}
  - {{ yamlString . }}
{{- end }}
{{- end }}
---

{{ trimSpace .SystemPrompt }}{{- if .Examples }}

## Examples
{{- range .Examples }}

### Input
{{ .Input }}

### Output
{{ .Output }}
{{- end }}
{{- end }}
`))
