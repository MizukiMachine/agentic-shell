package skill

// SkillMeta captures the structured metadata stored in a skill file.
type SkillMeta struct {
	Name        string   `yaml:"name"`
	Category    string   `yaml:"category"`
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools"`
	Tags        []string `yaml:"tags"`
}

// SkillFile represents a skill file discovered on disk.
type SkillFile struct {
	Path       string
	Metadata   SkillMeta
	RawContent string
}

// SkillMatch is the ranked result of matching a requirement against a skill.
type SkillMatch struct {
	Skill           SkillFile
	Score           float64
	MatchedKeywords []string
}
