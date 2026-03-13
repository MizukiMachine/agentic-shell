package agent

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	types "github.com/MizukiMachine/agentic-shell/pkg/types"
)

// Generator converts AgentSpec into a Claude Code compatible agent definition.
type Generator struct{}

// NewGenerator creates a Generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate converts AgentSpec into ClaudeAgentDefinition.
func (g *Generator) Generate(ctx context.Context, spec *types.AgentSpec) (*types.ClaudeAgentDefinition, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if spec == nil {
		return nil, fmt.Errorf("spec is required")
	}
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid agent spec: %w", err)
	}

	description := firstNonEmpty(
		spec.Metadata.Description,
		spec.Intent.Goals.Primary.Main.Description,
		spec.Metadata.Name,
	)

	def := types.NewClaudeAgentDefinition(spec.Metadata.Name, description)
	def.Metadata.Version = firstNonEmpty(spec.Metadata.Version, def.Metadata.Version)
	def.Metadata.Author = spec.Metadata.Author
	def.Metadata.Description = description
	def.Metadata.CreatedAt = firstNonEmpty(spec.Metadata.CreatedAt, spec.Intent.Metadata.CreatedAt)
	def.Metadata.UpdatedAt = firstNonEmpty(spec.Metadata.UpdatedAt, spec.Intent.Metadata.CreatedAt, spec.Metadata.CreatedAt)
	def.Metadata.Tags = uniqueStrings(spec.Metadata.Tags)
	def.Metadata.Category = inferCategory(spec)
	def.Metadata.SourceIntentID = spec.Intent.Metadata.IntentID
	def.Metadata.SourceSpecID = spec.Metadata.Name
	def.Metadata.Labels = inferLabels(spec)

	def.Prompt.SystemPrompt = buildSystemPrompt(spec)
	def.Prompt.UserPrompt = spec.Intent.Goals.Primary.Main.Description
	def.Prompt.Examples = buildPromptExamples(spec)
	def.Prompt.Traits = inferTraits(spec)
	def.Prompt.CommunicationStyle = inferCommunicationStyle(spec)

	applyModelConfig(def, spec)
	applyContextConfig(def, spec)
	applySafetyConfig(def, spec)
	applyOutputConfig(def, spec)
	applyOperationalConfig(def, spec)

	def.Tools = inferTools(spec)

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("invalid generated definition: %w", err)
	}

	return def, nil
}

// RenderMarkdown renders Claude Code compatible Markdown.
func (g *Generator) RenderMarkdown(def *types.ClaudeAgentDefinition) (string, error) {
	if def == nil {
		return "", fmt.Errorf("definition is required")
	}
	if err := def.Validate(); err != nil {
		return "", fmt.Errorf("invalid definition: %w", err)
	}

	data := markdownTemplateData{
		Name:         def.Metadata.Name,
		Description:  frontmatterDescription(def),
		Model:        def.Model.ModelID,
		Tools:        toolNames(def.Tools),
		SystemPrompt: def.Prompt.SystemPrompt,
		Examples:     def.Prompt.Examples,
	}

	var buf bytes.Buffer
	if err := agentMarkdownTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}

	return buf.String(), nil
}

// MarkdownFileName returns a filesystem-safe markdown filename stem for the agent.
func MarkdownFileName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return "agent"
	}

	var builder strings.Builder
	lastHyphen := false
	for _, r := range normalized {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastHyphen = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if builder.Len() == 0 || lastHyphen {
				continue
			}
			builder.WriteByte('-')
			lastHyphen = true
		}
	}

	filename := strings.Trim(builder.String(), "-")
	if filename == "" {
		return "agent"
	}
	return filename
}

func buildSystemPrompt(spec *types.AgentSpec) string {
	var builder strings.Builder

	builder.WriteString("# Role\n")
	builder.WriteString("You are ")
	builder.WriteString(spec.Metadata.Name)
	builder.WriteString(".\n\n")
	builder.WriteString(firstNonEmpty(spec.Metadata.Description, spec.Intent.Goals.Primary.Main.Description))
	builder.WriteString("\n")

	appendBulletSection(&builder, "Mission", missionLines(spec))
	appendBulletSection(&builder, "Functional Requirements", functionalRequirementLines(spec))
	appendBulletSection(&builder, "Quality and Constraints", qualityConstraintLines(spec))
	appendBulletSection(&builder, "Capabilities", capabilityLines(spec))
	appendBulletSection(&builder, "Skills", skillLines(spec))
	appendBulletSection(&builder, "Behavior Rules", behaviorRuleLines(spec))
	appendBulletSection(&builder, "Knowledge Sources", knowledgeSourceLines(spec))
	appendBulletSection(&builder, "Execution Preferences", preferenceLines(spec))
	appendBulletSection(&builder, "Output Expectations", outputExpectationLines(spec))
	appendBulletSection(&builder, "Safety and Security", safetyLines(spec))

	return strings.TrimSpace(builder.String())
}

func missionLines(spec *types.AgentSpec) []string {
	lines := []string{}

	mainGoal := strings.TrimSpace(spec.Intent.Goals.Primary.Main.Description)
	if mainGoal != "" {
		lines = append(lines, "Primary goal: "+mainGoal)
	}

	for _, goal := range spec.Intent.Goals.Primary.Supporting {
		if goal.Description == "" {
			continue
		}
		lines = append(lines, "Supporting goal: "+goal.Description)
	}

	for _, criteria := range spec.Intent.Goals.Primary.Main.SuccessCriteria {
		if strings.TrimSpace(criteria) == "" {
			continue
		}
		lines = append(lines, "Success criterion: "+criteria)
	}

	return lines
}

func functionalRequirementLines(spec *types.AgentSpec) []string {
	lines := make([]string, 0, len(spec.Intent.Objectives.Functional)+len(spec.Intent.Objectives.NonFunctional))

	for _, requirement := range spec.Intent.Objectives.Functional {
		line := requirement.Description
		if requirement.Testable {
			line += " (testable)"
		}
		lines = append(lines, line)
		for _, criteria := range requirement.AcceptanceCriteria {
			if strings.TrimSpace(criteria) == "" {
				continue
			}
			lines = append(lines, "Acceptance: "+criteria)
		}
	}

	for _, requirement := range spec.Intent.Objectives.NonFunctional {
		line := fmt.Sprintf("%s: %s", requirement.Category, requirement.Description)
		if strings.TrimSpace(requirement.Metric) != "" {
			line += fmt.Sprintf(" [metric: %s]", requirement.Metric)
		}
		lines = append(lines, line)
	}

	return lines
}

func qualityConstraintLines(spec *types.AgentSpec) []string {
	lines := make([]string, 0, len(spec.Intent.Objectives.Quality)+len(spec.Intent.Objectives.Constraints))

	for _, quality := range spec.Intent.Objectives.Quality {
		lines = append(lines, fmt.Sprintf("%s: minimum %.0f, target %.0f", quality.Aspect, quality.MinimumScore, quality.TargetScore))
	}

	for _, constraint := range spec.Intent.Objectives.Constraints {
		line := fmt.Sprintf("%s constraint (%s): %s", constraint.Type, constraint.Impact, constraint.Description)
		if strings.TrimSpace(constraint.Workaround) != "" {
			line += " | workaround: " + constraint.Workaround
		}
		lines = append(lines, line)
	}

	return lines
}

func capabilityLines(spec *types.AgentSpec) []string {
	lines := make([]string, 0, len(spec.Capabilities))
	for _, capability := range spec.Capabilities {
		line := capability.Name
		if strings.TrimSpace(capability.Description) != "" {
			line += ": " + capability.Description
		}
		if strings.TrimSpace(capability.Level) != "" {
			line += fmt.Sprintf(" [%s]", capability.Level)
		}
		lines = append(lines, line)
	}
	return lines
}

func skillLines(spec *types.AgentSpec) []string {
	lines := make([]string, 0, len(spec.Skills))
	for _, skill := range spec.Skills {
		line := skill.Name
		if strings.TrimSpace(skill.Description) != "" {
			line += ": " + skill.Description
		}
		if strings.TrimSpace(skill.Complexity) != "" {
			line += fmt.Sprintf(" [%s]", skill.Complexity)
		}
		lines = append(lines, line)
	}
	return lines
}

func behaviorRuleLines(spec *types.AgentSpec) []string {
	lines := make([]string, 0, len(spec.BehaviorRules))
	for _, rule := range spec.BehaviorRules {
		if !rule.Enabled {
			continue
		}
		lines = append(lines, fmt.Sprintf("When %s, %s", rule.Condition, rule.Action))
	}
	return lines
}

func knowledgeSourceLines(spec *types.AgentSpec) []string {
	lines := make([]string, 0, len(spec.KnowledgeSources))
	for _, source := range spec.KnowledgeSources {
		line := fmt.Sprintf("%s (%s)", source.Name, source.Type)
		if strings.TrimSpace(source.URI) != "" {
			line += ": " + source.URI
		}
		lines = append(lines, line)
	}
	return lines
}

func preferenceLines(spec *types.AgentSpec) []string {
	preferences := spec.Intent.Preferences
	lines := []string{}

	if preferences.QualityVsSpeed.Bias != "" {
		lines = append(lines, fmt.Sprintf("Favor %s over the competing speed trade-off", preferences.QualityVsSpeed.Bias))
	}
	if preferences.CostVsPerformance.Bias != "" {
		lines = append(lines, fmt.Sprintf("Cost/performance bias: %s", preferences.CostVsPerformance.Bias))
	}
	if preferences.AutomationVsControl.Bias != "" {
		lines = append(lines, fmt.Sprintf("Automation mode: %s", preferences.AutomationVsControl.Bias))
	}
	if preferences.Risk.Tolerance != "" {
		lines = append(lines, fmt.Sprintf("Risk tolerance: %s", preferences.Risk.Tolerance))
	}
	for _, approval := range preferences.AutomationVsControl.ApprovalRequired {
		if strings.TrimSpace(approval) == "" {
			continue
		}
		lines = append(lines, "Require approval for: "+approval)
	}

	return lines
}

func outputExpectationLines(spec *types.AgentSpec) []string {
	lines := []string{
		fmt.Sprintf("Primary modality: %s", spec.Intent.Modality.Primary),
	}

	if spec.Intent.Modality.Code != nil {
		lines = append(lines, fmt.Sprintf("Code language: %s", spec.Intent.Modality.Code.Language))
		if spec.Intent.Modality.Code.IncludeTests {
			lines = append(lines, "Include tests in code-oriented outputs")
		}
	}

	if spec.Intent.Modality.Text != nil {
		lines = append(lines, fmt.Sprintf("Natural language: %s", spec.Intent.Modality.Text.Language))
		lines = append(lines, fmt.Sprintf("Tone: %s", spec.Intent.Modality.Text.Tone))
		if spec.Intent.Modality.Text.MaxLength != nil {
			lines = append(lines, fmt.Sprintf("Keep responses within approximately %d characters", *spec.Intent.Modality.Text.MaxLength))
		}
	}

	if spec.Intent.Modality.Data != nil {
		lines = append(lines, fmt.Sprintf("Structured data format: %s", spec.Intent.Modality.Data.Format))
	}

	return lines
}

func safetyLines(spec *types.AgentSpec) []string {
	lines := []string{
		fmt.Sprintf("Sandbox enabled: %t", spec.Security.SandboxEnabled),
		fmt.Sprintf("Data classification: %s", spec.Security.DataClassification),
		fmt.Sprintf("Audit logging enabled: %t", spec.Security.AuditEnabled),
		fmt.Sprintf("Encryption required: %t", spec.Security.EncryptionRequired),
	}

	for _, domain := range spec.Security.AllowedDomains {
		if strings.TrimSpace(domain) == "" {
			continue
		}
		lines = append(lines, "Allowed domain: "+domain)
	}
	for _, command := range spec.Security.AllowedCommands {
		if strings.TrimSpace(command) == "" {
			continue
		}
		lines = append(lines, "Allowed command: "+command)
	}

	return lines
}

func appendBulletSection(builder *strings.Builder, title string, lines []string) {
	lines = compactStrings(lines)
	if len(lines) == 0 {
		return
	}

	builder.WriteString("\n\n## ")
	builder.WriteString(title)
	builder.WriteString("\n")
	for _, line := range lines {
		builder.WriteString("- ")
		builder.WriteString(line)
		builder.WriteByte('\n')
	}
}

func buildPromptExamples(spec *types.AgentSpec) []types.PromptExample {
	examples := make([]types.PromptExample, 0, 4)

	for _, rule := range spec.BehaviorRules {
		if !rule.Enabled {
			continue
		}
		examples = append(examples, types.PromptExample{
			Input:  rule.Condition,
			Output: rule.Action,
		})
		if len(examples) == 4 {
			return examples
		}
	}

	for _, skill := range spec.Skills {
		for _, example := range skill.Examples {
			if strings.TrimSpace(example) == "" {
				continue
			}
			examples = append(examples, types.PromptExample{
				Input:  "Apply " + skill.Name,
				Output: example,
			})
			if len(examples) == 4 {
				return examples
			}
		}
	}

	return examples
}

func inferTraits(spec *types.AgentSpec) map[string]string {
	traits := map[string]string{}

	if spec.Intent.Preferences.QualityVsSpeed.Bias != "" {
		traits["quality_bias"] = string(spec.Intent.Preferences.QualityVsSpeed.Bias)
	}
	if spec.Intent.Preferences.CostVsPerformance.Bias != "" {
		traits["cost_bias"] = string(spec.Intent.Preferences.CostVsPerformance.Bias)
	}
	if spec.Intent.Preferences.AutomationVsControl.Bias != "" {
		traits["automation_bias"] = string(spec.Intent.Preferences.AutomationVsControl.Bias)
	}
	if spec.Intent.Preferences.Risk.Tolerance != "" {
		traits["risk_tolerance"] = string(spec.Intent.Preferences.Risk.Tolerance)
	}
	if spec.Intent.Modality.Text != nil {
		traits["response_language"] = spec.Intent.Modality.Text.Language
		traits["response_tone"] = string(spec.Intent.Modality.Text.Tone)
	}
	traits["primary_modality"] = string(spec.Intent.Modality.Primary)

	return traits
}

func inferCommunicationStyle(spec *types.AgentSpec) string {
	if spec.Intent.Modality.Text == nil {
		return "professional"
	}

	switch spec.Intent.Modality.Text.Tone {
	case types.TextToneCasual:
		return "casual"
	case types.TextToneFormal:
		return "formal"
	default:
		return "technical"
	}
}

func applyModelConfig(def *types.ClaudeAgentDefinition, spec *types.AgentSpec) {
	switch spec.Intent.Preferences.QualityVsSpeed.Bias {
	case types.QualitySpeedBiasQuality:
		def.Model.Temperature = 0.2
	case types.QualitySpeedBiasSpeed:
		def.Model.Temperature = 0.6
	default:
		def.Model.Temperature = 0.4
	}

	if spec.Intent.Modality.Text != nil && spec.Intent.Modality.Text.Tone == types.TextToneCasual {
		def.Model.Temperature = 0.7
	}
}

func applyContextConfig(def *types.ClaudeAgentDefinition, spec *types.AgentSpec) {
	def.Context.HistoryLimit = max(5, min(20, spec.Performance.MaxConcurrency*5))
	def.Context.ReservedTokens = max(1000, spec.Performance.Timeout*10)
}

func applySafetyConfig(def *types.ClaudeAgentDefinition, spec *types.AgentSpec) {
	def.Safety.ContentFiltering = true
	def.Safety.PiiHandling = "mask"
	def.Safety.AllowedContent = compactStrings(spec.Security.AllowedDomains)

	switch spec.Security.DataClassification {
	case "confidential", "restricted":
		def.Safety.BlockedContent = []string{"credentials", "secrets", "personal-data"}
	default:
		def.Safety.BlockedContent = []string{}
	}

	if spec.Intent.Modality.Text != nil && spec.Intent.Modality.Text.MaxLength != nil {
		def.Safety.MaxResponseLength = *spec.Intent.Modality.Text.MaxLength
	}
}

func applyOutputConfig(def *types.ClaudeAgentDefinition, spec *types.AgentSpec) {
	def.Output.Format = "markdown"
	def.Output.IncludeMetadata = false
	def.Output.PrettyPrint = true
	def.Output.IncludeReasoning = false

	if spec.Intent.Modality.Text != nil {
		def.Output.Language = firstNonEmpty(spec.Intent.Modality.Text.Language, def.Output.Language)
		def.Output.Tone = string(spec.Intent.Modality.Text.Tone)
		switch spec.Intent.Modality.Text.Format {
		case types.TextFormatPlain:
			def.Output.Format = "text"
		case types.TextFormatJSON:
			def.Output.Format = "json"
		default:
			def.Output.Format = "markdown"
		}
	}

	if spec.Intent.Modality.Primary == types.OutputModalityData && spec.Intent.Modality.Data != nil {
		switch spec.Intent.Modality.Data.Format {
		case types.DataFormatJSON:
			def.Output.Format = "json"
		default:
			def.Output.Format = "markdown"
		}
	}
}

func applyOperationalConfig(def *types.ClaudeAgentDefinition, spec *types.AgentSpec) {
	def.Logging.Enabled = spec.Security.AuditEnabled
	def.Logging.IncludeTrace = spec.Security.AuditEnabled
	if !spec.Security.AuditEnabled {
		def.Logging.Level = "error"
	}

	def.Metrics.Enabled = true
	def.Metrics.ExportInterval = max(30, spec.Performance.Timeout)
}

func inferCategory(spec *types.AgentSpec) string {
	if len(spec.Capabilities) > 0 && strings.TrimSpace(spec.Capabilities[0].Category) != "" {
		return spec.Capabilities[0].Category
	}
	if spec.Intent.Modality.Primary != "" {
		return string(spec.Intent.Modality.Primary)
	}
	return "general"
}

func inferLabels(spec *types.AgentSpec) map[string]string {
	labels := map[string]string{
		"communication_type": spec.Communication.Type,
		"primary_modality":   string(spec.Intent.Modality.Primary),
	}
	if spec.Security.DataClassification != "" {
		labels["data_classification"] = spec.Security.DataClassification
	}
	if spec.Intent.Modality.Text != nil && spec.Intent.Modality.Text.Language != "" {
		labels["language"] = spec.Intent.Modality.Text.Language
	}
	return labels
}

func inferTools(spec *types.AgentSpec) []types.ToolDefinition {
	selected := map[string]toolCatalogEntry{}
	addTool(selected, "Read")
	addTool(selected, "Grep")
	addTool(selected, "Glob")

	corpus := buildCorpus(spec)

	if needsWorkspaceWrite(spec, corpus) {
		addTool(selected, "Write")
		addTool(selected, "Edit")
		addTool(selected, "MultiEdit")
	}
	if needsShell(spec, corpus) {
		addTool(selected, "Bash")
	}
	if needsWebFetch(spec, corpus) {
		addTool(selected, "WebFetch")
	}
	if needsWebSearch(spec, corpus) {
		addTool(selected, "WebSearch")
	}

	tools := make([]types.ToolDefinition, 0, len(selected))
	for _, name := range orderedToolNames(selected) {
		entry := selected[name]
		tools = append(tools, types.ToolDefinition{
			Name:        entry.Name,
			Description: entry.Description,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		})
	}

	return tools
}

type toolCatalogEntry struct {
	Name        string
	Description string
}

var toolCatalog = map[string]toolCatalogEntry{
	"Read": {
		Name:        "Read",
		Description: "Read files from the workspace.",
	},
	"Grep": {
		Name:        "Grep",
		Description: "Search file contents with regular expressions.",
	},
	"Glob": {
		Name:        "Glob",
		Description: "Find files by glob pattern.",
	},
	"Write": {
		Name:        "Write",
		Description: "Create or overwrite files in the workspace.",
	},
	"Edit": {
		Name:        "Edit",
		Description: "Apply targeted edits to existing files.",
	},
	"MultiEdit": {
		Name:        "MultiEdit",
		Description: "Apply multiple edits to a single file atomically.",
	},
	"Bash": {
		Name:        "Bash",
		Description: "Run shell commands in the workspace.",
	},
	"WebFetch": {
		Name:        "WebFetch",
		Description: "Retrieve content from external URLs.",
	},
	"WebSearch": {
		Name:        "WebSearch",
		Description: "Search the web for external information.",
	},
}

var toolOrder = []string{
	"Read",
	"Grep",
	"Glob",
	"Write",
	"Edit",
	"MultiEdit",
	"Bash",
	"WebFetch",
	"WebSearch",
}

func addTool(selected map[string]toolCatalogEntry, name string) {
	entry, ok := toolCatalog[name]
	if !ok {
		return
	}
	selected[name] = entry
}

func orderedToolNames(selected map[string]toolCatalogEntry) []string {
	names := make([]string, 0, len(selected))
	for _, name := range toolOrder {
		if _, ok := selected[name]; ok {
			names = append(names, name)
		}
	}
	return names
}

func buildCorpus(spec *types.AgentSpec) string {
	parts := []string{
		spec.Metadata.Name,
		spec.Metadata.Description,
		spec.Intent.Goals.Primary.Main.Description,
		spec.Communication.Type,
		spec.Communication.Format,
	}

	for _, capability := range spec.Capabilities {
		parts = append(parts, capability.Name, capability.Description, capability.Category, capability.Level)
		parts = append(parts, capability.Keywords...)
	}
	for _, skill := range spec.Skills {
		parts = append(parts, skill.Name, skill.Description, skill.Complexity)
		parts = append(parts, skill.Prerequisites...)
		parts = append(parts, skill.Examples...)
	}
	for _, tool := range spec.Tools {
		parts = append(parts, tool.Name, tool.Description, tool.Category, tool.RiskLevel, tool.Returns)
		for _, parameter := range tool.Parameters {
			parts = append(parts, parameter.Name, parameter.Description, parameter.Type)
			parts = append(parts, parameter.Enum...)
		}
	}
	for _, source := range spec.KnowledgeSources {
		parts = append(parts, source.Name, source.Description, source.Type, source.URI)
	}
	for _, requirement := range spec.Intent.Objectives.Functional {
		parts = append(parts, requirement.Description)
		parts = append(parts, requirement.AcceptanceCriteria...)
	}
	for _, constraint := range spec.Intent.Objectives.Constraints {
		parts = append(parts, constraint.Description, constraint.Workaround)
	}

	return strings.ToLower(strings.Join(parts, " "))
}

func needsWorkspaceWrite(spec *types.AgentSpec, corpus string) bool {
	if spec.Intent.Modality.Primary == types.OutputModalityCode {
		return true
	}
	if containsAny(corpus, "code", "implement", "write", "edit", "refactor", "patch", "file", "generate") {
		return true
	}
	for _, tool := range spec.Tools {
		if tool.Category == "io" || tool.Category == "processing" {
			return true
		}
	}
	return false
}

func needsShell(spec *types.AgentSpec, corpus string) bool {
	if spec.Communication.Type == "cli" || spec.Intent.Modality.Primary == types.OutputModalityCode {
		return true
	}
	if containsAny(corpus, "build", "test", "run", "shell", "command", "cli", "deploy") {
		return true
	}
	for _, tool := range spec.Tools {
		if tool.Category == "processing" {
			return true
		}
	}
	return false
}

func needsWebFetch(spec *types.AgentSpec, corpus string) bool {
	if spec.Communication.Type == "rest" || spec.Communication.Type == "grpc" || spec.Communication.Type == "websocket" {
		return true
	}
	if containsAny(corpus, "api", "documentation", "docs", "http", "web", "url") {
		return true
	}
	for _, source := range spec.KnowledgeSources {
		if source.Type == "url" || source.Type == "api" || source.Type == "documentation" {
			return true
		}
	}
	return false
}

func needsWebSearch(spec *types.AgentSpec, corpus string) bool {
	if containsAny(corpus, "search", "research", "investigate", "discover", "latest", "current") {
		return true
	}
	for _, source := range spec.KnowledgeSources {
		if (source.Type == "url" || source.Type == "documentation") && strings.TrimSpace(source.URI) == "" {
			return true
		}
	}
	return false
}

func frontmatterDescription(def *types.ClaudeAgentDefinition) string {
	description := strings.TrimSpace(def.Metadata.Description)
	if description == "" {
		description = strings.TrimSpace(def.Prompt.SystemPrompt)
	}
	if description == "" {
		return def.Metadata.Name
	}

	description = strings.ReplaceAll(description, "\n", " ")
	description = strings.Join(strings.Fields(description), " ")
	if len(description) > 160 {
		return strings.TrimSpace(description[:157]) + "..."
	}
	return description
}

func toolNames(tools []types.ToolDefinition) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Name) == "" {
			continue
		}
		names = append(names, tool.Name)
	}
	return names
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

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
	sort.Strings(result)
	return result
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
