package pipeline

import types "github.com/MizukiMachine/agentic-shell/pkg/types"

// Envelope is the shared payload passed between pipeline subcommands.
type Envelope struct {
	Documents  []ParsedDocument       `json:"documents,omitempty" yaml:"documents,omitempty"`
	Extraction *ExtractionResult      `json:"extraction,omitempty" yaml:"extraction,omitempty"`
	SkillScan  *SkillScanResult       `json:"skill_scan,omitempty" yaml:"skill_scan,omitempty"`
	Match      *MatchResult           `json:"match,omitempty" yaml:"match,omitempty"`
	SkillGen   *SkillGenerationResult `json:"skill_gen,omitempty" yaml:"skill_gen,omitempty"`
	Output     *OutputResult          `json:"output,omitempty" yaml:"output,omitempty"`
}

// ParsedDocument is the structured representation of an input specification.
type ParsedDocument struct {
	Source     string                 `json:"source" yaml:"source"`
	Format     string                 `json:"format" yaml:"format"`
	Title      string                 `json:"title,omitempty" yaml:"title,omitempty"`
	Summary    string                 `json:"summary,omitempty" yaml:"summary,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Sections   []Section              `json:"sections,omitempty" yaml:"sections,omitempty"`
	Structured map[string]interface{} `json:"structured,omitempty" yaml:"structured,omitempty"`
	Raw        string                 `json:"raw,omitempty" yaml:"raw,omitempty"`
}

// Section represents a parsed markdown section or a structured YAML subsection.
type Section struct {
	Heading string   `json:"heading" yaml:"heading"`
	Level   int      `json:"level" yaml:"level"`
	Content string   `json:"content,omitempty" yaml:"content,omitempty"`
	Bullets []string `json:"bullets,omitempty" yaml:"bullets,omitempty"`
}

// Requirement captures an extracted requirement from the parsed spec.
type Requirement struct {
	ID          string   `json:"id" yaml:"id"`
	Category    string   `json:"category" yaml:"category"`
	Description string   `json:"description" yaml:"description"`
	Required    bool     `json:"required" yaml:"required"`
	Keywords    []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Source      string   `json:"source,omitempty" yaml:"source,omitempty"`
}

// SkillRequirement captures a capability/skill need inferred from the spec.
type SkillRequirement struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Keywords    []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Required    bool     `json:"required" yaml:"required"`
	Source      string   `json:"source,omitempty" yaml:"source,omitempty"`
}

// ExtractionResult is the output of the extract subcommand.
type ExtractionResult struct {
	Summary           string             `json:"summary,omitempty" yaml:"summary,omitempty"`
	AgentSpec         *types.AgentSpec   `json:"agent_spec,omitempty" yaml:"agent_spec,omitempty"`
	Requirements      []Requirement      `json:"requirements,omitempty" yaml:"requirements,omitempty"`
	SkillRequirements []SkillRequirement `json:"skill_requirements,omitempty" yaml:"skill_requirements,omitempty"`
}

// SkillInfo describes a discovered skill file.
type SkillInfo struct {
	Name        string   `json:"name" yaml:"name"`
	Category    string   `json:"category,omitempty" yaml:"category,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Path        string   `json:"path" yaml:"path"`
	Tools       []string `json:"tools,omitempty" yaml:"tools,omitempty"`
	Tags        []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Keywords    []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`
}

// SkillScanResult is the output of the skill-scan subcommand.
type SkillScanResult struct {
	Directory string      `json:"directory" yaml:"directory"`
	Skills    []SkillInfo `json:"skills,omitempty" yaml:"skills,omitempty"`
}

// MatchedSkill represents a ranked match between a requirement and a skill.
type MatchedSkill struct {
	Name    string   `json:"name" yaml:"name"`
	Path    string   `json:"path" yaml:"path"`
	Score   float64  `json:"score" yaml:"score"`
	Reasons []string `json:"reasons,omitempty" yaml:"reasons,omitempty"`
}

// RequirementMatch contains the result of matching one required skill.
type RequirementMatch struct {
	RequirementID   string         `json:"requirement_id" yaml:"requirement_id"`
	RequirementName string         `json:"requirement_name" yaml:"requirement_name"`
	Description     string         `json:"description,omitempty" yaml:"description,omitempty"`
	Matches         []MatchedSkill `json:"matches,omitempty" yaml:"matches,omitempty"`
	Missing         bool           `json:"missing" yaml:"missing"`
	MissingReason   string         `json:"missing_reason,omitempty" yaml:"missing_reason,omitempty"`
}

// MatchResult is the output of the match subcommand.
type MatchResult struct {
	Matches       []RequirementMatch `json:"matches,omitempty" yaml:"matches,omitempty"`
	MissingSkills []SkillRequirement `json:"missing_skills,omitempty" yaml:"missing_skills,omitempty"`
}

// GeneratedFile is a file planned by the skill-gen stage.
type GeneratedFile struct {
	Path    string `json:"path" yaml:"path"`
	Content string `json:"content" yaml:"content"`
}

// SkillGenerationResult is the output of the placeholder skill-gen stage.
type SkillGenerationResult struct {
	Status        string             `json:"status" yaml:"status"`
	Summary       string             `json:"summary,omitempty" yaml:"summary,omitempty"`
	MissingSkills []SkillRequirement `json:"missing_skills,omitempty" yaml:"missing_skills,omitempty"`
	Files         []GeneratedFile    `json:"files,omitempty" yaml:"files,omitempty"`
}

// OutputResult summarizes files written by the output stage.
type OutputResult struct {
	BaseDir      string   `json:"base_dir" yaml:"base_dir"`
	WrittenFiles []string `json:"written_files,omitempty" yaml:"written_files,omitempty"`
	SkippedFiles []string `json:"skipped_files,omitempty" yaml:"skipped_files,omitempty"`
}
