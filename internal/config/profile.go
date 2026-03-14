package config

import (
	"fmt"
	"sort"
	"strings"
)

// Profile は設定プロファイルを表します。
type Profile struct {
	Name         string          `mapstructure:"name" yaml:"name"`
	Settings     ConfigOverrides `mapstructure:"settings" yaml:"settings"`
	Tools        []string        `mapstructure:"tools" yaml:"tools"`
	OutputFormat string          `mapstructure:"output_format" yaml:"output_format"`
}

// ProfileManager は利用可能なプロファイルと選択状態を管理します。
type ProfileManager struct {
	current  string
	profiles map[string]Profile
}

// NewProfileManager はビルトインプロファイル込みの ProfileManager を返します。
func NewProfileManager() *ProfileManager {
	profiles := map[string]Profile{}
	for name, profile := range builtInProfiles() {
		profiles[name] = profile
	}
	return &ProfileManager{profiles: profiles}
}

// Load はカスタムプロファイルを取り込みます。
func (m *ProfileManager) Load(profiles map[string]Profile) error {
	if m == nil {
		return fmt.Errorf("profile manager is nil")
	}

	for name, profile := range profiles {
		profileName := normalizeProfileName(name)
		if profileName == "" {
			profileName = normalizeProfileName(profile.Name)
		}
		if profileName == "" {
			return fmt.Errorf("profile name is required")
		}

		profile.Name = profileName
		profile.OutputFormat = normalizeOutputFormat(profile.OutputFormat)
		if err := validateProfile(profile); err != nil {
			return err
		}
		m.profiles[profileName] = profile
	}

	return nil
}

// Switch は現在プロファイルを切り替えます。空文字列で解除します。
func (m *ProfileManager) Switch(name string) error {
	if m == nil {
		return fmt.Errorf("profile manager is nil")
	}

	normalized := normalizeProfileName(name)
	if normalized == "" {
		m.current = ""
		return nil
	}
	if _, ok := m.profiles[normalized]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	m.current = normalized
	return nil
}

// Merge は base -> named profile -> CLI overrides の順で設定を合成します。
func (m *ProfileManager) Merge(base *Config, profileName string, cliOverrides *ConfigOverrides) (*Config, error) {
	if base == nil {
		return nil, fmt.Errorf("base config is required")
	}

	merged := *base
	selected := normalizeProfileName(profileName)
	if selected == "" {
		selected = m.current
	}

	if selected != "" {
		profile, ok := m.profiles[selected]
		if !ok {
			return nil, fmt.Errorf("profile %q not found", profileName)
		}
		merged.Merge(&profile.Settings)
		if profile.OutputFormat != "" {
			merged.Output.Format = profile.OutputFormat
		}
	}

	merged.Merge(cliOverrides)
	if err := merged.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &merged, nil
}

// Current は現在のプロファイル名を返します。
func (m *ProfileManager) Current() string {
	if m == nil {
		return ""
	}
	return m.current
}

// Profile は名前からプロファイルを取得します。
func (m *ProfileManager) Profile(name string) (Profile, bool) {
	if m == nil {
		return Profile{}, false
	}
	profile, ok := m.profiles[normalizeProfileName(name)]
	return profile, ok
}

// Names は利用可能なプロファイル名一覧を返します。
func (m *ProfileManager) Names() []string {
	if m == nil {
		return nil
	}

	names := make([]string, 0, len(m.profiles))
	for name := range m.profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func builtInProfiles() map[string]Profile {
	return map[string]Profile{
		"dev": {
			Name: "dev",
			Settings: ConfigOverrides{
				LLM: LLMConfigOverrides{
					MaxRetries: profileIntPtr(1),
				},
				Output: OutputConfigOverrides{
					Overwrite: profileBoolPtr(true),
				},
				Gathering: GatheringConfigOverrides{
					ConfidenceThreshold: profileFloat64Ptr(0.70),
				},
				Generation: GenerationConfigOverrides{
					DefaultTemperature: profileFloat64Ptr(0.85),
				},
			},
			Tools:        []string{"read", "write", "bash"},
			OutputFormat: "markdown",
		},
		"prod": {
			Name: "prod",
			Settings: ConfigOverrides{
				LLM: LLMConfigOverrides{
					MaxRetries: profileIntPtr(5),
				},
				Output: OutputConfigOverrides{
					Overwrite: profileBoolPtr(false),
				},
				Gathering: GatheringConfigOverrides{
					ConfidenceThreshold: profileFloat64Ptr(0.95),
				},
				Generation: GenerationConfigOverrides{
					DefaultTemperature: profileFloat64Ptr(0.30),
				},
			},
			Tools:        []string{"read"},
			OutputFormat: "json",
		},
	}
}

func validateProfile(profile Profile) error {
	if profile.OutputFormat == "" {
		return nil
	}

	validFormats := map[string]bool{
		"markdown": true,
		"yaml":     true,
		"json":     true,
	}
	if !validFormats[profile.OutputFormat] {
		return fmt.Errorf("profile %q has invalid output_format: %s", profile.Name, profile.OutputFormat)
	}

	return nil
}

func normalizeProfileName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeOutputFormat(format string) string {
	return strings.ToLower(strings.TrimSpace(format))
}

func profileIntPtr(v int) *int {
	return &v
}

func profileFloat64Ptr(v float64) *float64 {
	return &v
}

func profileBoolPtr(v bool) *bool {
	return &v
}
