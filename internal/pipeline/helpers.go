package pipeline

import (
	"encoding/json"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

func yamlLikeMarshal(value interface{}) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func tokenize(text string) []string {
	var builder strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(r)
		default:
			builder.WriteByte(' ')
		}
	}

	parts := strings.Fields(builder.String())
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 2 {
			continue
		}
		filtered = append(filtered, part)
	}
	return uniqueStrings(filtered)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func slugify(text string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(r)
			lastDash = false
		case !lastDash:
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func DecodeEnvelope(data []byte) (*Envelope, bool, error) {
	var probe map[string]interface{}
	if err := json.Unmarshal(data, &probe); err == nil && looksLikeEnvelope(probe) {
		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return nil, false, err
		}
		return &env, true, nil
	}

	probe = map[string]interface{}{}
	if err := yaml.Unmarshal(data, &probe); err == nil && looksLikeEnvelope(probe) {
		var env Envelope
		if err := yaml.Unmarshal(data, &env); err != nil {
			return nil, false, err
		}
		return &env, true, nil
	}

	return nil, false, nil
}

func looksLikeEnvelope(probe map[string]interface{}) bool {
	if len(probe) == 0 {
		return false
	}

	for _, key := range []string{"documents", "extraction", "skill_scan", "match", "skill_gen", "output"} {
		if _, ok := probe[key]; ok {
			return true
		}
	}
	return false
}
