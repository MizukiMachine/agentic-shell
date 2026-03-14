package skill

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	types "github.com/MizukiMachine/agentic-shell/pkg/types"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultSkillOutputDir is the default target directory for generated skills.
	DefaultSkillOutputDir = ".claude/skills"
	minimumCoverageScore  = 0.34
)

// ErrSkillGenerationDeclined indicates that the user rejected generation.
var ErrSkillGenerationDeclined = errors.New("skill generation cancelled by user")

// SkillGenerator generates placeholder skills inferred from an AgentSpec.
type SkillGenerator struct {
	spec       *types.AgentSpec
	outputDir  string
	confirm    bool
	auto       bool
	input      io.Reader
	output     io.Writer
	scanSkills func(string) ([]SkillFile, error)
}

// SkillGeneratorConfig configures the SkillGenerator runtime behavior.
type SkillGeneratorConfig struct {
	OutputDir  string
	Confirm    bool
	Auto       bool
	Input      io.Reader
	Output     io.Writer
	ScanSkills func(string) ([]SkillFile, error)
}

// PlannedSkill is a generated or inferred skill candidate.
type PlannedSkill struct {
	Name        string
	Category    string
	Description string
	Tools       []string
	Tags        []string
	Sources     []string
	Path        string
	Content     string
}

// SkillGenerationPlan describes inferred and generated skills.
type SkillGenerationPlan struct {
	OutputDir      string
	ExistingSkills []SkillFile
	RequiredSkills []PlannedSkill
	MissingSkills  []PlannedSkill
	WrittenFiles   []string
}

// NewSkillGenerator creates a SkillGenerator.
func NewSkillGenerator(spec *types.AgentSpec, cfg SkillGeneratorConfig) *SkillGenerator {
	outputDir := strings.TrimSpace(cfg.OutputDir)
	if outputDir == "" {
		outputDir = DefaultSkillOutputDir
	}

	input := cfg.Input
	if input == nil {
		input = os.Stdin
	}

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	scanSkills := cfg.ScanSkills
	if scanSkills == nil {
		scanSkills = ScanSkills
	}

	return &SkillGenerator{
		spec:       spec,
		outputDir:  filepath.Clean(outputDir),
		confirm:    cfg.Confirm,
		auto:       cfg.Auto,
		input:      input,
		output:     output,
		scanSkills: scanSkills,
	}
}

// Analyze infers required skills, checks existing skills, and prepares content
// for any missing skills.
func (g *SkillGenerator) Analyze() (*SkillGenerationPlan, error) {
	if g == nil {
		return nil, fmt.Errorf("generator is required")
	}
	if g.spec == nil {
		return nil, fmt.Errorf("spec is required")
	}

	existingSkills, err := g.scanSkills(g.outputDir)
	if err != nil {
		return nil, fmt.Errorf("scan existing skills: %w", err)
	}

	requiredSkills := g.inferSkills()
	missingSkills := make([]PlannedSkill, 0, len(requiredSkills))
	for _, requiredSkill := range requiredSkills {
		if g.hasExistingCoverage(requiredSkill, existingSkills) {
			continue
		}

		requiredSkill.Path = filepath.Join(skillSlug(requiredSkill.Name), "SKILL.md")
		requiredSkill.Content, err = renderSkillTemplate(requiredSkill)
		if err != nil {
			return nil, fmt.Errorf("render %q: %w", requiredSkill.Name, err)
		}
		missingSkills = append(missingSkills, requiredSkill)
	}

	sort.Slice(requiredSkills, func(i, j int) bool {
		return requiredSkills[i].Name < requiredSkills[j].Name
	})
	sort.Slice(missingSkills, func(i, j int) bool {
		return missingSkills[i].Path < missingSkills[j].Path
	})

	return &SkillGenerationPlan{
		OutputDir:      g.outputDir,
		ExistingSkills: existingSkills,
		RequiredSkills: requiredSkills,
		MissingSkills:  missingSkills,
		WrittenFiles:   []string{},
	}, nil
}

// Generate analyzes, previews, optionally confirms, and writes missing skills.
func (g *SkillGenerator) Generate() (*SkillGenerationPlan, error) {
	if g.confirm && g.auto {
		return nil, fmt.Errorf("--confirm and --auto cannot be used together")
	}

	plan, err := g.Analyze()
	if err != nil {
		return nil, err
	}

	g.printPreview(plan)

	if len(plan.MissingSkills) == 0 {
		return plan, nil
	}

	if g.confirm && !g.auto {
		approved, err := g.promptForConfirmation()
		if err != nil {
			return nil, err
		}
		if !approved {
			return nil, ErrSkillGenerationDeclined
		}
	}

	writtenFiles, err := g.writeSkills(plan.MissingSkills)
	if err != nil {
		return nil, err
	}
	plan.WrittenFiles = writtenFiles

	return plan, nil
}

func (g *SkillGenerator) inferSkills() []PlannedSkill {
	candidates := map[string]*PlannedSkill{}
	metadataTags := uniqueStrings(append([]string{}, g.spec.Metadata.Tags...))

	addCandidate := func(name, category, description string, tools, tags, sources []string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}

		key := skillSlug(name)
		if key == "" {
			return
		}

		normalizedCategory := normalizeGeneratedCategory(category, name+" "+description)
		mergedTags := uniqueStrings(append(append([]string{}, metadataTags...), tags...))
		if len(mergedTags) == 0 {
			mergedTags = tokenize(name + " " + description)
		}

		existing, ok := candidates[key]
		if !ok {
			candidates[key] = &PlannedSkill{
				Name:        name,
				Category:    normalizedCategory,
				Description: placeholderDescription(name, description, normalizedCategory),
				Tools:       uniqueStrings(tools),
				Tags:        mergedTags,
				Sources:     uniqueStrings(sources),
			}
			return
		}

		if existing.Description == "" && description != "" {
			existing.Description = placeholderDescription(name, description, normalizedCategory)
		}
		if existing.Category == "" || existing.Category == "general" {
			existing.Category = normalizedCategory
		}
		existing.Tools = uniqueStrings(append(existing.Tools, tools...))
		existing.Tags = uniqueStrings(append(existing.Tags, mergedTags...))
		existing.Sources = uniqueStrings(append(existing.Sources, sources...))
	}

	for _, skill := range g.spec.Skills {
		addCandidate(
			skill.Name,
			inferCategoryFromText(skill.Name+" "+skill.Description),
			skill.Description,
			nil,
			append(tokenize(strings.Join(skill.Examples, " ")), tokenize(skill.Description)...),
			[]string{"spec.skills"},
		)
	}

	for _, capability := range g.spec.Capabilities {
		addCandidate(
			capability.Name,
			capability.Category,
			capability.Description,
			nil,
			append([]string{}, capability.Keywords...),
			[]string{"spec.capabilities"},
		)
	}

	for _, inferred := range inferToolSkills(g.spec.Tools) {
		addCandidate(
			inferred.Name,
			inferred.Category,
			inferred.Description,
			inferred.Tools,
			inferred.Tags,
			[]string{"spec.tools"},
		)
	}

	for _, inferred := range inferIntentSkills(g.spec) {
		addCandidate(
			inferred.Name,
			inferred.Category,
			inferred.Description,
			inferred.Tools,
			inferred.Tags,
			[]string{"spec.intent"},
		)
	}

	result := make([]PlannedSkill, 0, len(candidates))
	for _, candidate := range candidates {
		candidate.Category = normalizeGeneratedCategory(candidate.Category, candidate.Name+" "+candidate.Description)
		candidate.Description = placeholderDescription(candidate.Name, candidate.Description, candidate.Category)
		candidate.Tags = uniqueStrings(candidate.Tags)
		candidate.Tools = uniqueStrings(candidate.Tools)
		candidate.Sources = uniqueStrings(candidate.Sources)
		result = append(result, *candidate)
	}

	return result
}

func (g *SkillGenerator) hasExistingCoverage(requiredSkill PlannedSkill, existingSkills []SkillFile) bool {
	if len(existingSkills) == 0 {
		return false
	}

	requirement := SkillMeta{
		Name:        requiredSkill.Name,
		Category:    requiredSkill.Category,
		Description: requiredSkill.Description,
		Tools:       requiredSkill.Tools,
		Tags:        requiredSkill.Tags,
	}

	matches := MatchSkills(requirement, existingSkills)
	if len(matches) == 0 {
		return false
	}

	topMatch := matches[0]
	if normalizePhrase(requiredSkill.Name) == normalizePhrase(topMatch.Skill.Metadata.Name) {
		return true
	}

	return topMatch.Score >= minimumCoverageScore
}

func (g *SkillGenerator) printPreview(plan *SkillGenerationPlan) {
	if g.output == nil || plan == nil {
		return
	}

	fmt.Fprintf(g.output, "Skill generation preview\n")
	fmt.Fprintf(g.output, "  output directory: %s\n", plan.OutputDir)
	fmt.Fprintf(g.output, "  existing skills scanned: %d\n", len(plan.ExistingSkills))
	fmt.Fprintf(g.output, "  inferred required skills: %d\n", len(plan.RequiredSkills))
	fmt.Fprintf(g.output, "  missing skills to generate: %d\n", len(plan.MissingSkills))

	for _, missing := range plan.MissingSkills {
		fmt.Fprintf(g.output, "  - %s [%s] -> %s\n", missing.Name, missing.Category, missing.Path)
	}
}

func (g *SkillGenerator) promptForConfirmation() (bool, error) {
	if g.output != nil {
		fmt.Fprint(g.output, "Generate these skill files? [y/N]: ")
	}

	reader := bufio.NewReader(g.input)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func (g *SkillGenerator) writeSkills(skills []PlannedSkill) ([]string, error) {
	writtenFiles := make([]string, 0, len(skills))

	for _, plannedSkill := range skills {
		targetPath := filepath.Join(g.outputDir, plannedSkill.Path)
		if _, err := os.Stat(targetPath); err == nil {
			return nil, fmt.Errorf("skill file already exists: %s", targetPath)
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat %s: %w", targetPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", filepath.Dir(targetPath), err)
		}
		if err := os.WriteFile(targetPath, []byte(plannedSkill.Content), 0644); err != nil {
			return nil, fmt.Errorf("write %s: %w", targetPath, err)
		}

		writtenFiles = append(writtenFiles, targetPath)
	}

	sort.Strings(writtenFiles)
	return writtenFiles, nil
}

func renderSkillTemplate(skill PlannedSkill) (string, error) {
	type frontMatter struct {
		Name        string   `yaml:"name"`
		Category    string   `yaml:"category"`
		Description string   `yaml:"description"`
		Tools       []string `yaml:"tools,omitempty"`
		Tags        []string `yaml:"tags,omitempty"`
	}

	frontMatterText, err := yaml.Marshal(frontMatter{
		Name:        skill.Name,
		Category:    skill.Category,
		Description: skill.Description,
		Tools:       uniqueStrings(skill.Tools),
		Tags:        uniqueStrings(skill.Tags),
	})
	if err != nil {
		return "", err
	}

	template := templateForCategory(skill.Category)

	content := fmt.Sprintf(`---
%s---

# %s

## Purpose

- %s
- Replace these placeholders with project-specific instructions before production use.

## Inputs

- Describe the user requests this skill should handle.
- List required repositories, files, services, or credentials.

## Workflow

%s

## Validation

%s

## Examples

- Example request: <describe a representative task>
- Expected response: <describe the expected deliverable>

## Notes

- Generated from AgentSpec sources: %s
- Add concrete guardrails, fallback behavior, and escalation rules.
`,
		string(frontMatterText),
		skill.Name,
		skill.Description,
		bulletBlock(template.Workflow),
		bulletBlock(template.Validation),
		strings.Join(skill.Sources, ", "),
	)

	return strings.TrimSpace(content) + "\n", nil
}

type categoryTemplate struct {
	Workflow   []string
	Validation []string
}

func templateForCategory(category string) categoryTemplate {
	switch normalizeGeneratedCategory(category, category) {
	case "development":
		return categoryTemplate{
			Workflow: []string{
				"Inspect the relevant code paths, interfaces, and tests before editing.",
				"Capture assumptions, edge cases, and backward-compatibility risks in the instructions.",
				"Specify how implementation changes and tests should be updated together.",
			},
			Validation: []string{
				"Define the commands or checks that prove the change is correct.",
				"Document expected failure modes and how to surface them to the user.",
			},
		}
	case "security":
		return categoryTemplate{
			Workflow: []string{
				"Identify trust boundaries, sensitive inputs, and privilege requirements before acting.",
				"Require the least-privilege approach and explicit justification for elevated actions.",
				"Call out auditability, secrets handling, and rollback expectations.",
			},
			Validation: []string{
				"Verify controls are preserved and document evidence for each security-sensitive change.",
				"List rejection conditions for unsafe requests or missing approvals.",
			},
		}
	case "automation":
		return categoryTemplate{
			Workflow: []string{
				"Describe deterministic execution steps, inputs, and expected outputs.",
				"Document idempotency, retries, and timeout behavior for automated tasks.",
				"Define when to stop automation and request user intervention.",
			},
			Validation: []string{
				"Add success checks for each automated step and explicit recovery actions for failures.",
				"Specify logging or progress signals that make the workflow observable.",
			},
		}
	default:
		return categoryTemplate{
			Workflow: []string{
				"Describe the core task sequence and decision points.",
				"Document the information required before acting and any important constraints.",
				"List fallback behavior for incomplete context or partial failures.",
			},
			Validation: []string{
				"Explain how to verify the result and what evidence should be returned.",
				"Document when the skill should stop and ask for clarification.",
			},
		}
	}
}

func bulletBlock(lines []string) string {
	if len(lines) == 0 {
		return "- Add category-specific workflow steps."
	}

	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, "- "+strings.TrimSpace(line))
	}
	return strings.Join(result, "\n")
}

func placeholderDescription(name, description, category string) string {
	switch {
	case strings.TrimSpace(description) != "":
		return strings.TrimSpace(description)
	case strings.TrimSpace(category) != "" && category != "general":
		return fmt.Sprintf("Placeholder %s skill for %s", category, name)
	default:
		return fmt.Sprintf("Placeholder skill for %s", name)
	}
}

func normalizeGeneratedCategory(category, corpus string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "development", "dev", "engineering":
		return "development"
	case "security", "compliance":
		return "security"
	case "automation", "operations", "ops":
		return "automation"
	case "research", "analysis":
		return "research"
	case "communication", "processing":
		return inferCategoryFromText(corpus)
	case "":
		return inferCategoryFromText(corpus)
	default:
		return strings.ToLower(strings.TrimSpace(category))
	}
}

func inferCategoryFromText(text string) string {
	tokens := tokenSet(text)
	switch {
	case hasAnyToken(tokens, "security", "secure", "audit", "auth", "authorization", "authentication", "permission", "permissions", "secret", "secrets", "encryption", "vulnerability"):
		return "security"
	case hasAnyToken(tokens, "automate", "automation", "workflow", "pipeline", "deploy", "release", "sync", "orchestrate", "cli"):
		return "automation"
	case hasAnyToken(tokens, "research", "investigate", "compare", "evaluate", "analysis"):
		return "research"
	case hasAnyToken(tokens, "code", "review", "test", "refactor", "build", "debug", "implementation", "develop", "development", "documentation"):
		return "development"
	default:
		return "general"
	}
}

func inferToolSkills(tools []types.Tool) []PlannedSkill {
	skills := []PlannedSkill{}
	for _, tool := range tools {
		corpus := strings.ToLower(strings.Join([]string{tool.Name, tool.Description, tool.Category}, " "))
		tokens := tokenSet(corpus)
		switch {
		case hasAnyToken(tokens, "bash", "shell", "terminal", "command", "cli", "exec", "execute"):
			skills = append(skills, PlannedSkill{
				Name:        "CLI Automation",
				Category:    "automation",
				Description: "Use command-line tooling safely and deterministically.",
				Tools:       []string{tool.Name},
				Tags:        []string{"cli", "automation"},
			})
		case hasAnyToken(tokens, "http", "https", "web", "api", "request", "fetch", "rest", "graphql"):
			skills = append(skills, PlannedSkill{
				Name:        "API Integration",
				Category:    "automation",
				Description: "Integrate with external APIs and web resources reliably.",
				Tools:       []string{tool.Name},
				Tags:        []string{"api", "integration"},
			})
		case hasAnyToken(tokens, "read", "write", "edit", "file", "filesystem"):
			skills = append(skills, PlannedSkill{
				Name:        "File Operations",
				Category:    "development",
				Description: "Read and update repository files with clear validation steps.",
				Tools:       []string{tool.Name},
				Tags:        []string{"files", "repository"},
			})
		case hasAnyToken(tokens, "git", "repo", "repository", "diff", "commit"):
			skills = append(skills, PlannedSkill{
				Name:        "Repository Analysis",
				Category:    "development",
				Description: "Inspect repository history, diffs, and local changes accurately.",
				Tools:       []string{tool.Name},
				Tags:        []string{"git", "review"},
			})
		case hasAnyToken(tokens, "security", "audit", "auth", "secret", "permission"):
			skills = append(skills, PlannedSkill{
				Name:        "Security Review",
				Category:    "security",
				Description: "Assess security-sensitive operations and guardrails before execution.",
				Tools:       []string{tool.Name},
				Tags:        []string{"security", "audit"},
			})
		}
	}

	return skills
}

func inferIntentSkills(spec *types.AgentSpec) []PlannedSkill {
	corpusParts := []string{
		spec.Metadata.Name,
		spec.Metadata.Description,
		spec.Intent.Goals.Primary.Main.Description,
	}

	for _, goal := range spec.Intent.Goals.Primary.Supporting {
		corpusParts = append(corpusParts, goal.Description)
	}
	for _, requirement := range spec.Intent.Objectives.Functional {
		corpusParts = append(corpusParts, requirement.Description)
		corpusParts = append(corpusParts, requirement.AcceptanceCriteria...)
	}
	for _, requirement := range spec.Intent.Objectives.NonFunctional {
		corpusParts = append(corpusParts, requirement.Description)
		if requirement.Target != nil {
			corpusParts = append(corpusParts, fmt.Sprint(requirement.Target))
		}
	}
	for _, quality := range spec.Intent.Objectives.Quality {
		corpusParts = append(corpusParts, quality.Description)
	}
	for _, constraint := range spec.Intent.Objectives.Constraints {
		corpusParts = append(corpusParts, constraint.Description, constraint.Workaround)
	}

	tokens := tokenSet(strings.Join(corpusParts, " "))
	result := []PlannedSkill{}

	if hasAnyToken(tokens, "review", "diff", "regression", "quality", "refactor") {
		result = append(result, PlannedSkill{
			Name:        "Code Review",
			Category:    "development",
			Description: "Inspect changes critically and surface the highest-value findings first.",
			Tags:        []string{"review", "quality"},
		})
	}
	if hasAnyToken(tokens, "test", "testing", "validation", "coverage", "assert") {
		result = append(result, PlannedSkill{
			Name:        "Test Automation",
			Category:    "development",
			Description: "Design and validate test coverage for the requested change.",
			Tags:        []string{"testing", "validation"},
		})
	}
	if hasAnyToken(tokens, "security", "auth", "permission", "permissions", "secret", "encryption", "audit", "vulnerability") {
		result = append(result, PlannedSkill{
			Name:        "Security Review",
			Category:    "security",
			Description: "Evaluate security constraints, approvals, and sensitive data handling.",
			Tags:        []string{"security", "audit"},
		})
	}
	if hasAnyToken(tokens, "automate", "automation", "workflow", "pipeline", "deploy", "release", "orchestrate") {
		result = append(result, PlannedSkill{
			Name:        "Workflow Automation",
			Category:    "automation",
			Description: "Coordinate repeatable automation steps with clear checkpoints.",
			Tags:        []string{"automation", "workflow"},
		})
	}
	if hasAnyToken(tokens, "document", "documentation", "docs", "summarize", "summary") {
		result = append(result, PlannedSkill{
			Name:        "Documentation",
			Category:    "development",
			Description: "Turn technical context into clear written guidance and summaries.",
			Tags:        []string{"documentation", "communication"},
		})
	}

	return result
}

func tokenSet(text string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, token := range tokenize(text) {
		set[token] = struct{}{}
	}
	return set
}

func hasAnyToken(tokens map[string]struct{}, values ...string) bool {
	for _, value := range values {
		for _, token := range tokenize(value) {
			if _, ok := tokens[token]; ok {
				return true
			}
		}
	}
	return false
}

func skillSlug(text string) string {
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
