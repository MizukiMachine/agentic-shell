package pipeline

import (
	"fmt"
	"sort"
	"strings"
	"time"

	specpkg "github.com/MizukiMachine/agentic-shell/internal/spec"
	types "github.com/MizukiMachine/agentic-shell/pkg/types"
)

// Extract populates AgentSpec, requirements, and skill requirements from parsed documents.
func Extract(env *Envelope) error {
	if env == nil || len(env.Documents) == 0 {
		return fmt.Errorf("parsed documents are required")
	}

	spec := materializeAgentSpec(env.Documents)
	requirements := deriveRequirements(env.Documents, spec)
	skillRequirements := deriveSkillRequirements(env.Documents, spec, requirements)

	env.Extraction = &ExtractionResult{
		Summary:           summarizeDocuments(env.Documents),
		AgentSpec:         spec,
		Requirements:      requirements,
		SkillRequirements: skillRequirements,
	}

	return nil
}

func materializeAgentSpec(documents []ParsedDocument) *types.AgentSpec {
	for _, doc := range documents {
		if doc.Format != "yaml" || len(doc.Structured) == 0 {
			continue
		}

		data, err := yamlLikeMarshal(doc.Structured)
		if err != nil {
			continue
		}

		spec := &types.AgentSpec{}
		if err := spec.FromJSON(data); err == nil && strings.TrimSpace(spec.Metadata.Name) != "" {
			_ = specpkg.ValidateWithThreshold(spec, 0)
			return spec
		}
	}

	return buildAgentSpecFromDocuments(documents)
}

func buildAgentSpecFromDocuments(documents []ParsedDocument) *types.AgentSpec {
	now := time.Now().UTC().Format(time.RFC3339)
	title := firstNonEmpty(documents[0].Title, "generated-agent")
	name := slugify(title)
	if name == "" {
		name = "generated-agent"
	}

	spec := types.NewAgentSpec(name, "1.0.0")
	spec.Metadata.Name = name
	spec.Metadata.Description = firstNonEmpty(summarizeDocuments(documents), title)
	spec.Metadata.CreatedAt = now
	spec.Metadata.UpdatedAt = now
	spec.Intent.Metadata.IntentID = "intent-" + name
	spec.Intent.Metadata.CreatedAt = now
	spec.Intent.Goals.Primary.Main = types.Goal{
		ID:          "goal-main",
		Type:        types.GoalTypePrimary,
		Description: firstNonEmpty(spec.Metadata.Description, title),
		Priority:    types.GoalPriorityHigh,
		Measurable:  true,
	}
	spec.Intent.Goals.AllGoals = []types.Goal{spec.Intent.Goals.Primary.Main}
	spec.Intent.Objectives.Functional = nil
	spec.Intent.Modality.Primary = types.OutputModalityText
	spec.Intent.Modality.Text = &types.TextModality{
		Format:   types.TextFormatMarkdown,
		Language: "en",
		Tone:     types.TextToneTechnical,
	}
	spec.Communication.Type = inferCommunicationType(documents)
	spec.Communication.Format = inferCommunicationFormat(spec.Communication.Type)

	requirements := deriveRequirements(documents, nil)
	for idx, requirement := range requirements {
		switch requirement.Category {
		case "functional":
			spec.Intent.Objectives.Functional = append(spec.Intent.Objectives.Functional, types.FunctionalRequirement{
				ID:                 requirement.ID,
				Description:        requirement.Description,
				Priority:           types.GoalPriorityHigh,
				AcceptanceCriteria: []string{"Requirement is satisfied by the resulting skill or workflow"},
				Testable:           true,
			})
		case "constraint":
			spec.Intent.Objectives.Constraints = append(spec.Intent.Objectives.Constraints, types.Constraint{
				ID:          requirement.ID,
				Type:        types.ConstraintTypeTechnical,
				Description: requirement.Description,
				Impact:      types.ConstraintImpactAdvisory,
			})
		case "quality":
			spec.Intent.Objectives.NonFunctional = append(spec.Intent.Objectives.NonFunctional, types.NonFunctionalRequirement{
				ID:          requirement.ID,
				Category:    types.NFCategoryMaintainability,
				Description: requirement.Description,
				Metric:      "explicit requirement",
			})
		case "tool":
			spec.Tools = append(spec.Tools, types.Tool{
				ID:          requirement.ID,
				Name:        requirement.Description,
				Description: requirement.Description,
				Category:    "processing",
				RiskLevel:   "low",
			})
		}

		if idx == 0 {
			spec.Intent.Goals.Primary.Main.Description = requirement.Description
		}
	}

	if len(spec.Intent.Objectives.Functional) == 0 {
		spec.Intent.Objectives.Functional = []types.FunctionalRequirement{
			{
				ID:                 "fr-1",
				Description:        spec.Metadata.Description,
				Priority:           types.GoalPriorityHigh,
				AcceptanceCriteria: []string{"Result is documented and reproducible"},
				Testable:           true,
			},
		}
	}

	for _, skillReq := range deriveSkillRequirements(documents, spec, requirements) {
		spec.Skills = append(spec.Skills, types.Skill{
			ID:          skillReq.ID,
			Name:        skillReq.Name,
			Description: skillReq.Description,
			Complexity:  "medium",
		})
	}

	if len(spec.Capabilities) == 0 {
		spec.Capabilities = []types.Capability{
			{
				ID:          "cap-1",
				Name:        title,
				Description: spec.Metadata.Description,
				Category:    "analysis",
				Level:       "intermediate",
				Keywords:    uniqueStrings(tokenize(spec.Metadata.Description)),
			},
		}
	}

	_ = specpkg.ValidateWithThreshold(spec, 0)

	return spec
}

func deriveRequirements(documents []ParsedDocument, spec *types.AgentSpec) []Requirement {
	if spec != nil && strings.TrimSpace(spec.Metadata.Name) != "" && len(spec.Intent.Objectives.Functional) > 0 {
		return requirementsFromAgentSpec(spec)
	}

	var requirements []Requirement
	counter := 1
	for _, doc := range documents {
		for _, section := range doc.Sections {
			category := categorizeSection(section.Heading)
			lines := section.Bullets
			if len(lines) == 0 && strings.TrimSpace(section.Content) != "" {
				lines = []string{strings.TrimSpace(section.Content)}
			}

			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				requirements = append(requirements, Requirement{
					ID:          fmt.Sprintf("req-%d", counter),
					Category:    category,
					Description: strings.TrimSpace(line),
					Required:    true,
					Keywords:    uniqueStrings(tokenize(section.Heading + " " + line)),
					Source:      doc.Source,
				})
				counter++
			}
		}
	}

	if len(requirements) == 0 {
		requirements = append(requirements, Requirement{
			ID:          "req-1",
			Category:    "functional",
			Description: summarizeDocuments(documents),
			Required:    true,
			Keywords:    uniqueStrings(tokenize(summarizeDocuments(documents))),
			Source:      documents[0].Source,
		})
	}

	return requirements
}

func requirementsFromAgentSpec(spec *types.AgentSpec) []Requirement {
	var requirements []Requirement
	counter := 1

	for _, requirement := range spec.Intent.Objectives.Functional {
		requirements = append(requirements, Requirement{
			ID:          firstNonEmpty(requirement.ID, fmt.Sprintf("req-%d", counter)),
			Category:    "functional",
			Description: requirement.Description,
			Required:    true,
			Keywords:    uniqueStrings(tokenize(requirement.Description)),
			Source:      spec.Metadata.Name,
		})
		counter++
	}

	for _, requirement := range spec.Intent.Objectives.NonFunctional {
		requirements = append(requirements, Requirement{
			ID:          firstNonEmpty(requirement.ID, fmt.Sprintf("req-%d", counter)),
			Category:    "quality",
			Description: requirement.Description,
			Required:    true,
			Keywords:    uniqueStrings(tokenize(requirement.Description)),
			Source:      spec.Metadata.Name,
		})
		counter++
	}

	for _, constraint := range spec.Intent.Objectives.Constraints {
		requirements = append(requirements, Requirement{
			ID:          fmt.Sprintf("req-%d", counter),
			Category:    "constraint",
			Description: constraint.Description,
			Required:    true,
			Keywords:    uniqueStrings(tokenize(constraint.Description)),
			Source:      spec.Metadata.Name,
		})
		counter++
	}

	for _, tool := range spec.Tools {
		requirements = append(requirements, Requirement{
			ID:          firstNonEmpty(tool.ID, fmt.Sprintf("req-%d", counter)),
			Category:    "tool",
			Description: tool.Name,
			Required:    true,
			Keywords:    uniqueStrings(tokenize(tool.Name + " " + tool.Description)),
			Source:      spec.Metadata.Name,
		})
		counter++
	}

	return requirements
}

func deriveSkillRequirements(documents []ParsedDocument, spec *types.AgentSpec, requirements []Requirement) []SkillRequirement {
	if spec != nil && len(spec.Skills) > 0 {
		skillRequirements := make([]SkillRequirement, 0, len(spec.Skills))
		for _, skill := range spec.Skills {
			skillRequirements = append(skillRequirements, SkillRequirement{
				ID:          firstNonEmpty(skill.ID, slugify(skill.Name)),
				Name:        firstNonEmpty(skill.Name, skill.Description),
				Description: skill.Description,
				Keywords:    uniqueStrings(tokenize(skill.Name + " " + skill.Description)),
				Required:    true,
				Source:      spec.Metadata.Name,
			})
		}
		return dedupeSkillRequirements(skillRequirements)
	}

	var skillRequirements []SkillRequirement
	for _, doc := range documents {
		for _, section := range doc.Sections {
			if !strings.Contains(strings.ToLower(section.Heading), "skill") {
				continue
			}
			lines := section.Bullets
			if len(lines) == 0 && strings.TrimSpace(section.Content) != "" {
				lines = []string{section.Content}
			}
			for _, line := range lines {
				name := line
				if before, _, ok := strings.Cut(line, ":"); ok {
					name = before
				}
				skillRequirements = append(skillRequirements, SkillRequirement{
					ID:          slugify(name),
					Name:        strings.TrimSpace(name),
					Description: strings.TrimSpace(line),
					Keywords:    uniqueStrings(tokenize(line)),
					Required:    true,
					Source:      doc.Source,
				})
			}
		}
	}

	if len(skillRequirements) == 0 {
		for _, requirement := range requirements {
			for _, inferred := range inferSkillNames(requirement.Description) {
				skillRequirements = append(skillRequirements, SkillRequirement{
					ID:          slugify(inferred),
					Name:        inferred,
					Category:    requirement.Category,
					Description: requirement.Description,
					Tools:       inferredSkillTools(requirement),
					Keywords:    uniqueStrings(append(tokenize(inferred), requirement.Keywords...)),
					Required:    true,
					Source:      requirement.Source,
				})
			}
		}
	}

	if len(skillRequirements) == 0 {
		summary := summarizeDocuments(documents)
		skillRequirements = append(skillRequirements, SkillRequirement{
			ID:          "general-analysis",
			Name:        "general-analysis",
			Description: summary,
			Keywords:    uniqueStrings(tokenize(summary)),
			Required:    true,
			Source:      documents[0].Source,
		})
	}

	return dedupeSkillRequirements(skillRequirements)
}

func inferSkillNames(text string) []string {
	lower := strings.ToLower(text)
	seen := map[string]struct{}{}
	var names []string

	mappings := map[string]string{
		"review":   "code-review",
		"test":     "test-automation",
		"document": "documentation",
		"docs":     "documentation",
		"security": "security-audit",
		"yaml":     "structured-output",
		"json":     "structured-output",
		"cli":      "cli-automation",
		"terminal": "cli-automation",
		"pipeline": "pipeline-design",
		"extract":  "requirements-extraction",
		"match":    "skill-matching",
		"generate": "skill-generation",
		"output":   "artifact-output",
		"markdown": "markdown-authoring",
	}

	for token, name := range mappings {
		if strings.Contains(lower, token) {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names
}

func dedupeSkillRequirements(items []SkillRequirement) []SkillRequirement {
	seen := map[string]SkillRequirement{}
	order := []string{}
	for _, item := range items {
		key := slugify(firstNonEmpty(item.Name, item.ID))
		if key == "" {
			continue
		}
		item.ID = firstNonEmpty(item.ID, key)
		item.Name = firstNonEmpty(item.Name, key)
		item.Category = strings.TrimSpace(item.Category)
		item.Tools = uniqueStrings(item.Tools)
		item.Keywords = uniqueStrings(item.Keywords)
		if _, ok := seen[key]; !ok {
			order = append(order, key)
			seen[key] = item
			continue
		}

		existing := seen[key]
		existing.Category = firstNonEmpty(existing.Category, item.Category)
		existing.Tools = uniqueStrings(append(existing.Tools, item.Tools...))
		existing.Keywords = uniqueStrings(append(existing.Keywords, item.Keywords...))
		existing.Description = firstNonEmpty(existing.Description, item.Description)
		existing.Source = firstNonEmpty(existing.Source, item.Source)
		seen[key] = existing
	}

	result := make([]SkillRequirement, 0, len(order))
	for _, key := range order {
		result = append(result, seen[key])
	}
	return result
}

func inferredSkillTools(requirement Requirement) []string {
	if requirement.Category != "tool" {
		return nil
	}
	return []string{requirement.Description}
}

func summarizeDocuments(documents []ParsedDocument) string {
	var parts []string
	for _, doc := range documents {
		if doc.Summary != "" {
			parts = append(parts, doc.Summary)
			continue
		}
		if doc.Title != "" {
			parts = append(parts, doc.Title)
		}
	}
	return strings.TrimSpace(strings.Join(uniqueStrings(parts), " "))
}

func categorizeSection(heading string) string {
	lower := strings.ToLower(strings.TrimSpace(heading))
	switch {
	case strings.Contains(lower, "skill"):
		return "skill"
	case strings.Contains(lower, "tool"):
		return "tool"
	case strings.Contains(lower, "constraint"), strings.Contains(lower, "limit"):
		return "constraint"
	case strings.Contains(lower, "quality"), strings.Contains(lower, "non-functional"):
		return "quality"
	default:
		return "functional"
	}
}

func inferCommunicationType(documents []ParsedDocument) string {
	corpus := strings.ToLower(summarizeDocuments(documents))
	if strings.Contains(corpus, "cli") || strings.Contains(corpus, "terminal") || strings.Contains(corpus, "command") {
		return "cli"
	}
	return "rest"
}

func inferCommunicationFormat(protocol string) string {
	if protocol == "cli" {
		return "text"
	}
	return "json"
}
