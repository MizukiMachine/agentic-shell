package skill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var defaultCache = NewSkillCache()

type cacheEntry struct {
	modTime time.Time
	skill   SkillFile
}

// SkillCache caches parsed skill files by path and modification time.
type SkillCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

// NewSkillCache creates a cache for scanned skill files.
func NewSkillCache() *SkillCache {
	return &SkillCache{
		entries: map[string]cacheEntry{},
	}
}

// ScanSkills scans a skill directory recursively and parses skill metadata.
func ScanSkills(dir string) ([]SkillFile, error) {
	return defaultCache.ScanSkills(dir)
}

// ScanSkills scans a skill directory recursively and reuses cached entries
// when file modification times have not changed.
func (c *SkillCache) ScanSkills(dir string) ([]SkillFile, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("dir is required")
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SkillFile{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skills path is not a directory: %s", dir)
	}

	skills := []SkillFile{}
	live := map[string]struct{}{}

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

		info, err := d.Info()
		if err != nil {
			return err
		}

		live[path] = struct{}{}
		if skill, ok := c.get(path, info.ModTime()); ok {
			skills = append(skills, skill)
			return nil
		}

		skill, err := parseSkillFile(dir, path)
		if err != nil {
			return err
		}
		c.put(path, info.ModTime(), skill)
		skills = append(skills, skill)
		return nil
	})
	if err != nil {
		return nil, err
	}

	c.prune(dir, live)
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Path < skills[j].Path
	})

	return skills, nil
}

// ParseSkillMetadata extracts skill metadata from a Markdown frontmatter block
// or, for compatibility, from a YAML-only skill file.
func ParseSkillMetadata(data []byte) (SkillMeta, error) {
	var meta SkillMeta

	frontMatter, ok := extractFrontMatter(string(data))
	if ok {
		if err := yaml.Unmarshal([]byte(frontMatter), &meta); err != nil {
			return SkillMeta{}, err
		}
		return normalizeSkillMeta(meta), nil
	}

	if err := yaml.Unmarshal(data, &meta); err == nil {
		return normalizeSkillMeta(meta), nil
	}

	return SkillMeta{}, nil
}

func (c *SkillCache) get(path string, modTime time.Time) (SkillFile, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[path]
	if !ok || !entry.modTime.Equal(modTime) {
		return SkillFile{}, false
	}
	return entry.skill, true
}

func (c *SkillCache) put(path string, modTime time.Time, skill SkillFile) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[path] = cacheEntry{
		modTime: modTime,
		skill:   skill,
	}
}

func (c *SkillCache) prune(root string, live map[string]struct{}) {
	prefix := filepath.Clean(root) + string(filepath.Separator)

	c.mu.Lock()
	defer c.mu.Unlock()

	for path := range c.entries {
		if path == filepath.Clean(root) || strings.HasPrefix(path, prefix) {
			if _, ok := live[path]; !ok {
				delete(c.entries, path)
			}
		}
	}
}

func parseSkillFile(rootDir, path string) (SkillFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SkillFile{}, err
	}

	meta, err := ParseSkillMetadata(data)
	if err != nil {
		return SkillFile{}, fmt.Errorf("parse %s: %w", path, err)
	}

	relative, err := filepath.Rel(rootDir, path)
	if err != nil {
		relative = path
	}
	relative = filepath.ToSlash(relative)

	if strings.TrimSpace(meta.Name) == "" {
		meta.Name = fallbackSkillName(relative)
	}

	return SkillFile{
		Path:       relative,
		Metadata:   meta,
		RawContent: string(data),
	}, nil
}

func normalizeSkillMeta(meta SkillMeta) SkillMeta {
	meta.Name = strings.TrimSpace(meta.Name)
	meta.Category = strings.TrimSpace(meta.Category)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.Tools = uniqueStrings(meta.Tools)
	meta.Tags = uniqueStrings(meta.Tags)
	return meta
}

func extractFrontMatter(text string) (string, bool) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return "", false
	}

	rest := normalized[4:]
	if idx := strings.Index(rest, "\n---\n"); idx >= 0 {
		return rest[:idx], true
	}
	if strings.HasSuffix(rest, "\n---") {
		return strings.TrimSuffix(rest, "\n---"), true
	}
	return "", false
}

func fallbackSkillName(relativePath string) string {
	base := filepath.Base(relativePath)
	if strings.EqualFold(base, "skill.md") {
		return filepath.Base(filepath.Dir(relativePath))
	}
	return trimExtension(base)
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

func trimExtension(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
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
