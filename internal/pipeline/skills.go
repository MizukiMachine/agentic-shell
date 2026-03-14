package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	skillpkg "github.com/MizukiMachine/agentic-shell/internal/skill"
	"gopkg.in/yaml.v3"
)

// ScanSkills scans a directory for existing skill definitions.
func ScanSkills(env *Envelope, dir string) error {
	if env == nil {
		return fmt.Errorf("envelope is required")
	}

	result := &SkillScanResult{
		Directory: dir,
		Skills:    []SkillInfo{},
	}

	skills, err := skillpkg.ScanSkills(dir)
	if err != nil {
		return err
	}
	for _, skill := range skills {
		result.Skills = append(result.Skills, mapSkillInfo(skill))
	}

	sort.Slice(result.Skills, func(i, j int) bool {
		return result.Skills[i].Path < result.Skills[j].Path
	})

	env.SkillScan = result
	return nil
}

// MatchSkills compares required skills against scanned skills.
func MatchSkills(env *Envelope) error {
	if env == nil || env.Extraction == nil {
		return fmt.Errorf("extraction result is required")
	}
	if env.SkillScan == nil {
		return fmt.Errorf("skill scan result is required")
	}

	scannedSkills := mapSkillFiles(env.SkillScan.Skills)
	result := &MatchResult{
		Matches:       []RequirementMatch{},
		MissingSkills: []SkillRequirement{},
	}

	for _, requirement := range env.Extraction.SkillRequirements {
		match := RequirementMatch{
			RequirementID:   requirement.ID,
			RequirementName: requirement.Name,
			Description:     requirement.Description,
			Matches:         []MatchedSkill{},
		}

		reqMeta := skillpkg.SkillMeta{
			Name:        requirement.Name,
			Description: requirement.Description,
			Tags:        append([]string{}, requirement.Keywords...),
		}

		for _, matched := range skillpkg.MatchSkills(reqMeta, scannedSkills) {
			match.Matches = append(match.Matches, MatchedSkill{
				Name:    matched.Skill.Metadata.Name,
				Path:    matched.Skill.Path,
				Score:   matched.Score,
				Reasons: matchedKeywordReasons(requirement.Name, matched),
			})
		}

		sort.Slice(match.Matches, func(i, j int) bool {
			if match.Matches[i].Score == match.Matches[j].Score {
				return match.Matches[i].Path < match.Matches[j].Path
			}
			return match.Matches[i].Score > match.Matches[j].Score
		})
		if len(match.Matches) > 3 {
			match.Matches = match.Matches[:3]
		}

		if len(match.Matches) == 0 || match.Matches[0].Score < 0.34 {
			match.Missing = true
			match.MissingReason = "no existing skill reached the minimum similarity threshold"
			result.MissingSkills = append(result.MissingSkills, requirement)
			match.Matches = nil
		}

		result.Matches = append(result.Matches, match)
	}

	env.Match = result
	return nil
}

// GenerateMissingSkills creates placeholder skill files for unmatched requirements.
func GenerateMissingSkills(env *Envelope) error {
	if env == nil || env.Match == nil {
		return fmt.Errorf("match result is required")
	}

	result := &SkillGenerationResult{
		Status:        "noop",
		MissingSkills: append([]SkillRequirement{}, env.Match.MissingSkills...),
		Files:         []GeneratedFile{},
	}

	if len(env.Match.MissingSkills) == 0 {
		result.Summary = "all required skills are already covered"
		env.SkillGen = result
		return nil
	}

	result.Status = "placeholder-generated"
	result.Summary = fmt.Sprintf("%d missing skills require follow-up generation", len(env.Match.MissingSkills))

	for _, skill := range env.Match.MissingSkills {
		content, err := renderSkillPlaceholder(skill)
		if err != nil {
			return fmt.Errorf("render placeholder for %q: %w", skill.Name, err)
		}

		filename := filepath.Join(slugify(skill.Name), "SKILL.md")
		result.Files = append(result.Files, GeneratedFile{
			Path:    filename,
			Content: content,
		})
	}

	env.SkillGen = result
	return nil
}

// WriteGeneratedFiles writes generated files to disk.
func WriteGeneratedFiles(env *Envelope, baseDir, skillsDir string, overwrite bool) error {
	if env == nil || env.SkillGen == nil {
		return fmt.Errorf("skill generation result is required")
	}

	result := &OutputResult{
		BaseDir:      baseDir,
		WrittenFiles: []string{},
		SkippedFiles: []string{},
	}

	for _, file := range env.SkillGen.Files {
		targetPath := filepath.Join(baseDir, skillsDir, file.Path)
		if !overwrite {
			if _, err := os.Stat(targetPath); err == nil {
				result.SkippedFiles = append(result.SkippedFiles, targetPath)
				continue
			} else if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("stat %s: %w", targetPath, err)
			}
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(targetPath), err)
		}
		if err := os.WriteFile(targetPath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", targetPath, err)
		}

		result.WrittenFiles = append(result.WrittenFiles, targetPath)
	}

	env.Output = result
	return nil
}

func mapSkillInfo(skillFile skillpkg.SkillFile) SkillInfo {
	name := firstNonEmpty(skillFile.Metadata.Name, trimExtension(filepath.Base(skillFile.Path)))

	return SkillInfo{
		Name:        name,
		Category:    skillFile.Metadata.Category,
		Description: skillFile.Metadata.Description,
		Path:        skillFile.Path,
		Tools:       append([]string{}, skillFile.Metadata.Tools...),
		Tags:        append([]string{}, skillFile.Metadata.Tags...),
		Keywords: uniqueStrings(append(
			tokenize(name+" "+skillFile.Metadata.Description+" "+skillFile.RawContent),
			skillFile.Metadata.Tags...,
		)),
	}
}

func mapSkillFiles(skills []SkillInfo) []skillpkg.SkillFile {
	result := make([]skillpkg.SkillFile, 0, len(skills))
	for _, skill := range skills {
		result = append(result, skillpkg.SkillFile{
			Path: skill.Path,
			Metadata: skillpkg.SkillMeta{
				Name:        skill.Name,
				Category:    skill.Category,
				Description: skill.Description,
				Tools:       append([]string{}, skill.Tools...),
				Tags:        append([]string{}, skill.Tags...),
			},
			RawContent: strings.Join(skill.Keywords, " "),
		})
	}
	return result
}

func matchedKeywordReasons(requirementName string, matched skillpkg.SkillMatch) []string {
	reasons := make([]string, 0, len(matched.MatchedKeywords)+2)
	for _, keyword := range matched.MatchedKeywords {
		reasons = append(reasons, "shared keyword: "+keyword)
	}
	if slugify(requirementName) == slugify(matched.Skill.Metadata.Name) {
		reasons = append(reasons, "exact normalized name match")
	}
	if strings.Contains(slugify(matched.Skill.Metadata.Name), slugify(requirementName)) || strings.Contains(slugify(requirementName), slugify(matched.Skill.Metadata.Name)) {
		reasons = append(reasons, "partial normalized name match")
	}
	return uniqueStrings(reasons)
}

func renderSkillPlaceholder(skill SkillRequirement) (string, error) {
	type skillFrontMatter struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}

	frontMatter, err := yaml.Marshal(skillFrontMatter{
		Name:        skill.Name,
		Description: fmt.Sprintf("Placeholder skill generated for %s", skill.Name),
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(fmt.Sprintf(`
---
%s---

# %s

## Purpose

- Cover the missing requirement: %s
- Replace this placeholder with a fully authored skill definition.

## Expected Workflow

- Define the concrete inputs and outputs.
- Document validation and failure handling.
- Add examples that demonstrate the skill in context.

## Notes

- Generated by the agentic-shell placeholder pipeline.
`, string(frontMatter), skill.Name, skill.Description)) + "\n", nil
}
