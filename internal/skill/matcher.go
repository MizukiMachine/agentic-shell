package skill

import (
	"sort"
	"strings"
	"unicode"
)

const (
	categoryWeight = 0.5
	keywordWeight  = 0.35
	toolWeight     = 0.15
)

// MatchSkills ranks skills against a structured requirement.
func MatchSkills(requirements SkillMeta, skills []SkillFile) []SkillMatch {
	req := normalizeSkillMeta(requirements)
	matches := make([]SkillMatch, 0, len(skills))

	for _, skillFile := range skills {
		score, matchedKeywords := scoreMatch(req, skillFile)
		if score <= 0 {
			continue
		}
		matches = append(matches, SkillMatch{
			Skill:           skillFile,
			Score:           score,
			MatchedKeywords: matchedKeywords,
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Skill.Path < matches[j].Skill.Path
		}
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// CalculateSimilarity compares requirement keywords to a skill's metadata.
func CalculateSimilarity(keywords []string, skillFile SkillFile) (float64, []string) {
	required := uniqueStrings(append([]string{}, keywords...))
	skillKeywords := skillKeywordSet(skillFile)
	if len(required) == 0 || len(skillKeywords) == 0 {
		return 0, nil
	}

	keywordSet := make(map[string]struct{}, len(skillKeywords))
	for _, keyword := range skillKeywords {
		keywordSet[keyword] = struct{}{}
	}

	matched := make([]string, 0, len(required))
	for _, keyword := range required {
		if _, ok := keywordSet[keyword]; ok {
			matched = append(matched, keyword)
		}
	}
	if len(matched) == 0 {
		return 0, nil
	}

	return float64(len(matched)) / float64(len(required)), matched
}

func scoreMatch(requirement SkillMeta, skillFile SkillFile) (float64, []string) {
	totalWeight := 0.0
	score := 0.0

	if strings.TrimSpace(requirement.Category) != "" {
		totalWeight += categoryWeight
		score += categoryWeight * categorySimilarity(requirement.Category, skillFile.Metadata.Category)
	}

	keywords := requirementKeywordSet(requirement)
	keywordScore, matched := CalculateSimilarity(keywords, skillFile)
	if len(keywords) > 0 {
		totalWeight += keywordWeight
		score += keywordWeight * keywordScore
	}

	if len(requirement.Tools) > 0 {
		totalWeight += toolWeight
		score += toolWeight * toolSimilarity(requirement.Tools, skillFile.Metadata.Tools)
	}

	if totalWeight == 0 {
		return 0, nil
	}

	return score / totalWeight, matched
}

func requirementKeywordSet(requirement SkillMeta) []string {
	return uniqueStrings(append(
		tokenize(requirement.Name+" "+requirement.Description),
		tokenize(strings.Join(requirement.Tags, " "))...,
	))
}

func skillKeywordSet(skillFile SkillFile) []string {
	return uniqueStrings(append(
		tokenize(skillFile.Metadata.Name+" "+skillFile.Metadata.Description+" "+skillFile.RawContent),
		tokenize(strings.Join(skillFile.Metadata.Tags, " "))...,
	))
}

func categorySimilarity(required, actual string) float64 {
	required = normalizePhrase(required)
	actual = normalizePhrase(actual)
	switch {
	case required == "" || actual == "":
		return 0
	case required == actual:
		return 1
	case strings.Contains(required, actual), strings.Contains(actual, required):
		return 0.5
	default:
		return 0
	}
}

func toolSimilarity(requiredTools, actualTools []string) float64 {
	required := normalizePhrases(requiredTools)
	actual := normalizePhrases(actualTools)
	if len(required) == 0 || len(actual) == 0 {
		return 0
	}

	actualSet := make(map[string]struct{}, len(actual))
	for _, tool := range actual {
		actualSet[tool] = struct{}{}
	}

	matches := 0
	for _, tool := range required {
		if _, ok := actualSet[tool]; ok {
			matches++
		}
	}

	return float64(matches) / float64(len(required))
}

func normalizePhrases(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if phrase := normalizePhrase(value); phrase != "" {
			normalized = append(normalized, phrase)
		}
	}
	return uniqueStrings(normalized)
}

func normalizePhrase(value string) string {
	return strings.Join(tokenize(value), " ")
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
