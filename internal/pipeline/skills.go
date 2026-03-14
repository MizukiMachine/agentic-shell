package pipeline

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			env.SkillScan = result
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("skills path is not a directory: %s", dir)
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !isSkillFile(dir, path) {
			return nil
		}

		skill, err := parseSkillFile(dir, path)
		if err != nil {
			return err
		}
		result.Skills = append(result.Skills, skill)
		return nil
	})
	if err != nil {
		return err
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

		reqTokens := uniqueStrings(append(tokenize(requirement.Name+" "+requirement.Description), requirement.Keywords...))
		for _, skill := range env.SkillScan.Skills {
			score, reasons := scoreSkillMatch(reqTokens, requirement.Name, skill)
			if score <= 0 {
				continue
			}
			match.Matches = append(match.Matches, MatchedSkill{
				Name:    skill.Name,
				Path:    skill.Path,
				Score:   score,
				Reasons: reasons,
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

func parseSkillFile(rootDir, path string) (SkillInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SkillInfo{}, err
	}

	doc, err := ParseDocument(path, data, detectFormat(path, data))
	if err != nil {
		return SkillInfo{}, err
	}

	name := firstNonEmpty(
		stringValue(doc.Metadata["name"]),
		doc.Title,
		trimExtension(filepath.Base(path)),
	)
	description := firstNonEmpty(
		stringValue(doc.Metadata["description"]),
		doc.Summary,
	)

	relative, err := filepath.Rel(rootDir, path)
	if err != nil {
		relative = path
	}

	return SkillInfo{
		Name:        name,
		Description: description,
		Path:        filepath.ToSlash(relative),
		Keywords:    uniqueStrings(tokenize(name + " " + description + " " + doc.Raw)),
	}, nil
}

func scoreSkillMatch(requirementTokens []string, requirementName string, skill SkillInfo) (float64, []string) {
	skillTokens := uniqueStrings(append(tokenize(skill.Name+" "+skill.Description), skill.Keywords...))
	if len(requirementTokens) == 0 || len(skillTokens) == 0 {
		return 0, nil
	}

	skillSet := make(map[string]struct{}, len(skillTokens))
	for _, token := range skillTokens {
		skillSet[token] = struct{}{}
	}

	overlap := 0
	reasons := []string{}
	for _, token := range requirementTokens {
		if _, ok := skillSet[token]; ok {
			overlap++
			reasons = append(reasons, "shared keyword: "+token)
		}
	}

	score := float64(overlap) / float64(len(requirementTokens))
	if slugify(requirementName) == slugify(skill.Name) {
		score += 0.5
		reasons = append(reasons, "exact normalized name match")
	}
	if strings.Contains(slugify(skill.Name), slugify(requirementName)) || strings.Contains(slugify(requirementName), slugify(skill.Name)) {
		score += 0.2
		reasons = append(reasons, "partial normalized name match")
	}
	if score > 1 {
		score = 1
	}

	return score, uniqueStrings(reasons)
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

func isSkillFile(rootDir, path string) bool {
	lower := strings.ToLower(filepath.Base(path))
	switch {
	case lower == "skill.md":
		return true
	case strings.HasSuffix(lower, ".skill"):
		return true
	case lower == "readme.md", lower == "readme.markdown":
		return false
	}

	ext := strings.ToLower(filepath.Ext(lower))
	if ext != ".md" && ext != ".markdown" {
		return false
	}

	relative, err := filepath.Rel(rootDir, path)
	if err != nil {
		return false
	}

	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 2 {
		return false
	}

	parent := strings.ToLower(parts[len(parts)-2])
	name := strings.ToLower(trimExtension(parts[len(parts)-1]))
	return name == parent
}
