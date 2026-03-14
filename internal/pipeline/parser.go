package pipeline

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFiles parses one or more specification files and appends them to an envelope.
func ParseFiles(env *Envelope, paths []string) (*Envelope, error) {
	if env == nil {
		env = &Envelope{}
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		doc, err := ParseDocument(filepath.Base(path), data, detectFormat(path, data))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		doc.Source = path
		env.Documents = append(env.Documents, doc)
	}

	return env, nil
}

// ParseStdin parses raw stdin bytes as a single document.
func ParseStdin(env *Envelope, source string, data []byte) (*Envelope, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return env, nil
	}
	if env == nil {
		env = &Envelope{}
	}

	doc, err := ParseDocument(source, data, detectFormat(source, data))
	if err != nil {
		return nil, err
	}
	env.Documents = append(env.Documents, doc)

	return env, nil
}

// ParseDocument parses a single YAML or Markdown document.
func ParseDocument(source string, data []byte, format string) (ParsedDocument, error) {
	switch format {
	case "yaml":
		return parseYAMLDocument(source, data)
	default:
		return parseMarkdownDocument(source, data)
	}
}

func parseYAMLDocument(source string, data []byte) (ParsedDocument, error) {
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ParsedDocument{}, err
	}

	structured := normalizeYAMLValue(raw)
	root, _ := structured.(map[string]interface{})
	metadata := map[string]interface{}{}
	title := guessTitleFromStructured(root)
	summary := ""
	sections := []Section{}

	if value, ok := root["metadata"].(map[string]interface{}); ok {
		metadata = value
		if title == "" {
			if name, ok := value["name"].(string); ok {
				title = strings.TrimSpace(name)
			}
		}
		if summary == "" {
			if description, ok := value["description"].(string); ok {
				summary = strings.TrimSpace(description)
			}
		}
	}

	keys := make([]string, 0, len(root))
	for key := range root {
		if key == "metadata" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		content := renderStructuredValue(root[key])
		sections = append(sections, Section{
			Heading: key,
			Level:   1,
			Content: content,
			Bullets: extractBullets(content),
		})
	}

	return ParsedDocument{
		Source:     source,
		Format:     "yaml",
		Title:      firstNonEmpty(title, trimExtension(source)),
		Summary:    summary,
		Metadata:   metadata,
		Sections:   sections,
		Structured: root,
		Raw:        string(data),
	}, nil
}

func parseMarkdownDocument(source string, data []byte) (ParsedDocument, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	metadata, body := splitFrontMatter(text)
	lines := strings.Split(body, "\n")

	title := firstNonEmpty(stringValue(metadata["title"]), firstMarkdownHeading(lines, 1), trimExtension(source))
	sections := []Section{}
	summary := ""

	var current *Section
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			level := headingLevel(trimmed)
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			if current != nil {
				finalizeSection(current)
				sections = append(sections, *current)
			}
			current = &Section{Heading: heading, Level: level}
			if level == 1 && title == "" {
				title = heading
			}
			continue
		}

		if current == nil {
			current = &Section{Heading: "Overview", Level: 1}
		}
		if current.Content != "" {
			current.Content += "\n"
		}
		current.Content += line
	}

	if current != nil {
		finalizeSection(current)
		sections = append(sections, *current)
	}

	if summary == "" {
		summary = firstParagraph(lines)
	}
	if summary == "" {
		summary = stringValue(metadata["description"])
	}

	return ParsedDocument{
		Source:   source,
		Format:   "markdown",
		Title:    title,
		Summary:  summary,
		Metadata: metadata,
		Sections: sections,
		Raw:      text,
	}, nil
}

func splitFrontMatter(text string) (map[string]interface{}, string) {
	metadata := map[string]interface{}{}
	if !strings.HasPrefix(text, "---\n") {
		return metadata, text
	}

	parts := strings.SplitN(text, "\n---\n", 2)
	if len(parts) != 2 {
		return metadata, text
	}

	if err := yaml.Unmarshal([]byte(parts[0][4:]), &metadata); err != nil {
		return map[string]interface{}{}, text
	}

	return metadata, parts[1]
}

func detectFormat(source string, data []byte) string {
	switch strings.ToLower(filepath.Ext(source)) {
	case ".yaml", ".yml", ".json":
		return "yaml"
	case ".md", ".markdown":
		return "markdown"
	}

	var probe interface{}
	if err := yaml.Unmarshal(data, &probe); err == nil {
		switch probe.(type) {
		case map[string]interface{}, map[interface{}]interface{}, []interface{}:
			return "yaml"
		}
	}

	return "markdown"
}

func normalizeYAMLValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			result[key] = normalizeYAMLValue(child)
		}
		return result
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, child := range typed {
			result[fmt.Sprintf("%v", key)] = normalizeYAMLValue(child)
		}
		return result
	case []interface{}:
		result := make([]interface{}, 0, len(typed))
		for _, child := range typed {
			result = append(result, normalizeYAMLValue(child))
		}
		return result
	default:
		return typed
	}
}

func renderStructuredValue(value interface{}) string {
	data, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return strings.TrimSpace(string(data))
}

func guessTitleFromStructured(root map[string]interface{}) string {
	if root == nil {
		return ""
	}
	if metadata, ok := root["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			return strings.TrimSpace(name)
		}
	}
	if title, ok := root["title"].(string); ok {
		return strings.TrimSpace(title)
	}
	return ""
}

func finalizeSection(section *Section) {
	section.Content = strings.TrimSpace(section.Content)
	section.Bullets = extractBullets(section.Content)
}

func firstMarkdownHeading(lines []string, level int) string {
	prefix := strings.Repeat("#", level) + " "
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

func firstParagraph(lines []string) string {
	var builder strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if builder.Len() > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(trimmed)
	}
	return strings.TrimSpace(builder.String())
}

func headingLevel(line string) int {
	level := 0
	for _, r := range line {
		if r == '#' {
			level++
			continue
		}
		break
	}
	if level == 0 {
		return 1
	}
	return level
}

func extractBullets(content string) []string {
	var bullets []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "- "), strings.HasPrefix(trimmed, "* "), strings.HasPrefix(trimmed, "+ "):
			bullets = append(bullets, strings.TrimSpace(trimmed[2:]))
		}
	}
	return bullets
}

func trimExtension(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func stringValue(value interface{}) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
