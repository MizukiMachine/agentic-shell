package config

import "testing"

func TestProfileManagerSupportsBuiltInProfiles(t *testing.T) {
	manager := NewProfileManager()

	if err := manager.Switch("dev"); err != nil {
		t.Fatalf("expected dev profile to be available: %v", err)
	}
	if manager.Current() != "dev" {
		t.Fatalf("expected current profile to be dev, got %q", manager.Current())
	}

	if err := manager.Switch("prod"); err != nil {
		t.Fatalf("expected prod profile to be available: %v", err)
	}
}

func TestProfileManagerLoadAndMergeOrder(t *testing.T) {
	manager := NewProfileManager()
	err := manager.Load(map[string]Profile{
		"custom": {
			Settings: ConfigOverrides{
				Output: OutputConfigOverrides{
					Directory: stringPtr("./profile-output"),
				},
				Generation: GenerationConfigOverrides{
					DefaultModel: stringPtr("custom-model"),
				},
			},
			Tools:        []string{"read", "grep"},
			OutputFormat: "yaml",
		},
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	base := DefaultConfig()
	base.Output.Directory = "./base-output"
	base.Generation.DefaultModel = "base-model"

	merged, err := manager.Merge(base, "custom", &ConfigOverrides{
		Output: OutputConfigOverrides{
			Directory: stringPtr("./cli-output"),
			Format:    stringPtr("json"),
		},
	})
	if err != nil {
		t.Fatalf("Merge returned error: %v", err)
	}

	if merged.Output.Directory != "./cli-output" {
		t.Fatalf("expected CLI override to win, got %q", merged.Output.Directory)
	}
	if merged.Output.Format != "json" {
		t.Fatalf("expected CLI format override to win, got %q", merged.Output.Format)
	}
	if merged.Generation.DefaultModel != "custom-model" {
		t.Fatalf("expected profile setting to override base, got %q", merged.Generation.DefaultModel)
	}
	if merged.LLM.Provider != "glm" {
		t.Fatalf("expected base config values to be preserved, got %q", merged.LLM.Provider)
	}
}

func TestProfileManagerSwitchUnknownProfile(t *testing.T) {
	manager := NewProfileManager()

	if err := manager.Switch("missing"); err == nil {
		t.Fatal("expected unknown profile error")
	}
}

func TestProfileManagerProfileLookup(t *testing.T) {
	manager := NewProfileManager()
	if err := manager.Load(map[string]Profile{
		"custom": {
			Tools:        []string{"read"},
			OutputFormat: "markdown",
		},
	}); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	profile, ok := manager.Profile("custom")
	if !ok {
		t.Fatal("expected custom profile to be available")
	}
	if profile.Name != "custom" {
		t.Fatalf("expected normalized profile name, got %q", profile.Name)
	}
	if profile.OutputFormat != "markdown" {
		t.Fatalf("expected output format markdown, got %q", profile.OutputFormat)
	}
}

func stringPtr(v string) *string {
	return &v
}
