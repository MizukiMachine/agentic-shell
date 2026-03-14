package skill

import "testing"

func TestCalculateSimilarityUsesTagsAndDescription(t *testing.T) {
	score, matched := CalculateSimilarity([]string{"rust", "testing", "coverage"}, SkillFile{
		Metadata: SkillMeta{
			Name:        "rust-testing",
			Description: "Run coverage reports",
			Tags:        []string{"rust", "testing"},
		},
		RawContent: "Use cargo llvm-cov for coverage.",
	})
	if score <= 0 {
		t.Fatalf("expected positive similarity score, got %v", score)
	}
	if len(matched) != 3 {
		t.Fatalf("expected all keywords to match, got %v", matched)
	}
}

func TestMatchSkillsPrioritizesCategoryBeforeTools(t *testing.T) {
	requirement := SkillMeta{
		Name:        "rust-test",
		Category:    "development",
		Description: "run tests for rust projects",
		Tools:       []string{"cargo test"},
		Tags:        []string{"rust", "testing"},
	}

	matches := MatchSkills(requirement, []SkillFile{
		{
			Path: "automation/SKILL.md",
			Metadata: SkillMeta{
				Name:        "automation-runner",
				Category:    "automation",
				Description: "run tests for rust projects",
				Tools:       []string{"cargo test"},
				Tags:        []string{"rust", "testing"},
			},
		},
		{
			Path: "development/SKILL.md",
			Metadata: SkillMeta{
				Name:        "rust-developer",
				Category:    "development",
				Description: "run tests for rust projects",
				Tags:        []string{"rust", "testing"},
			},
		},
	})

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].Skill.Path != "development/SKILL.md" {
		t.Fatalf("expected category match to rank first, got %s", matches[0].Skill.Path)
	}
}
