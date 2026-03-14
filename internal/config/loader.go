package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	// DefaultConfigName はデフォルトの設定ファイル名です
	DefaultConfigName = ".agentic-shell"

	// DefaultConfigType はデフォルトの設定ファイルタイプです
	DefaultConfigType = "yaml"

	// EnvPrefix は環境変数のプレフィックスです
	EnvPrefix = "AGENTIC"
)

// Loader は設定を読み込むための構造体です
type Loader struct {
	configName string
	configType string
	configPath string
	envPrefix  string
	usedFile   string
	profile    string
	overrides  *ConfigOverrides
}

// NewLoader は新しい Loader を作成します
func NewLoader() *Loader {
	return &Loader{
		configName: DefaultConfigName,
		configType: DefaultConfigType,
		configPath: "",
		envPrefix:  EnvPrefix,
	}
}

// WithConfigName は設定ファイル名を設定します
func (l *Loader) WithConfigName(name string) *Loader {
	l.configName = name
	return l
}

// WithConfigType は設定ファイルタイプを設定します
func (l *Loader) WithConfigType(configType string) *Loader {
	l.configType = configType
	return l
}

// WithConfigPath は設定ファイルパスを設定します
func (l *Loader) WithConfigPath(path string) *Loader {
	l.configPath = path
	return l
}

// WithEnvPrefix は環境変数プレフィックスを設定します
func (l *Loader) WithEnvPrefix(prefix string) *Loader {
	l.envPrefix = prefix
	return l
}

// WithProfile は読み込み時に適用するプロファイル名を設定します。
func (l *Loader) WithProfile(profile string) *Loader {
	l.profile = profile
	return l
}

// WithCLIOverrides は CLI フラグ由来の上書きを設定します。
func (l *Loader) WithCLIOverrides(overrides *ConfigOverrides) *Loader {
	l.overrides = overrides
	return l
}

// Load はデフォルト設定と設定ファイル、環境変数を読み込んでマージします
func (l *Loader) Load() (*Config, error) {
	// デフォルト設定で初期化
	config := DefaultConfig()

	v := viper.New()

	// 基本設定
	v.SetConfigName(l.configName)
	v.SetConfigType(l.configType)

	// 設定ファイルの検索パスを追加
	if l.configPath != "" {
		v.SetConfigFile(l.configPath)
	} else {
		// ホームディレクトリ
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(home)
		}
		// カレントディレクトリ
		v.AddConfigPath(".")
		// プロジェクトルート（.git があるディレクトリ）
		if cwd, err := os.Getwd(); err == nil {
			v.AddConfigPath(cwd)
		}
	}

	// 環境変数の設定
	v.SetEnvPrefix(l.envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	bindEnvKeys(v)

	// 設定ファイルを読み込み
	if err := v.ReadInConfig(); err != nil {
		if l.configPath != "" {
			return nil, fmt.Errorf("failed to read config file %q: %w", l.configPath, err)
		}
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		l.usedFile = v.ConfigFileUsed()
	}

	// デフォルト設定とマージ
	config.Merge(loadOverrides(v))

	activeProfile := strings.TrimSpace(l.profile)
	if activeProfile == "" && v.IsSet("profile") {
		activeProfile = v.GetString("profile")
	}

	profileManager := NewProfileManager()
	if v.IsSet("profiles") {
		customProfiles := map[string]Profile{}
		if err := v.UnmarshalKey("profiles", &customProfiles); err != nil {
			return nil, fmt.Errorf("failed to decode profiles: %w", err)
		}
		if err := profileManager.Load(customProfiles); err != nil {
			return nil, fmt.Errorf("failed to load profiles: %w", err)
		}
	}
	if err := profileManager.Switch(activeProfile); err != nil {
		return nil, err
	}

	config, err := profileManager.Merge(config, activeProfile, l.overrides)
	if err != nil {
		return nil, err
	}

	// 検証
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// ConfigFileUsed は読み込まれた設定ファイルパスを返します。
func (l *Loader) ConfigFileUsed() string {
	return l.usedFile
}

// LoadFromFile は指定されたファイルから設定を読み込みます
func LoadFromFile(path string) (*Config, error) {
	loader := NewLoader().WithConfigPath(path)
	return loader.Load()
}

// LoadFromHome はホームディレクトリから設定を読み込みます
func LoadFromHome() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, DefaultConfigName+".yaml")
	loader := NewLoader().WithConfigPath(configPath)
	return loader.Load()
}

// LoadWithDefaults はデフォルト設定のみを返します（設定ファイルを読み込みません）
func LoadWithDefaults() *Config {
	return DefaultConfig()
}

// WriteConfig は設定をファイルに書き込みます
func WriteConfig(path string, config *Config) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType(DefaultConfigType)

	// 設定を viper にセット
	v.Set("llm", config.LLM)
	v.Set("output", config.Output)
	v.Set("gathering", config.Gathering)
	v.Set("generation", config.Generation)

	// ディレクトリを作成
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// ファイルに書き込み
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// WriteDefaultConfig はデフォルト設定を指定されたパスに書き込みます
func WriteDefaultConfig(path string) error {
	return WriteConfig(path, DefaultConfig())
}

// GetConfigPath はデフォルトの設定ファイルパスを返します
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, DefaultConfigName+".yaml"), nil
}

// ConfigExists は設定ファイルが存在するかどうかを返します
func ConfigExists() (bool, error) {
	path, err := GetConfigPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func bindEnvKeys(v *viper.Viper) {
	for _, key := range []string{
		"profile",
		"llm.claude_path",
		"llm.timeout",
		"llm.max_retries",
		"output.directory",
		"output.format",
		"output.overwrite",
		"gathering.confidence_threshold",
		"gathering.max_question_rounds",
		"generation.default_model",
		"generation.default_temperature",
	} {
		_ = v.BindEnv(key)
	}
}

func loadOverrides(v *viper.Viper) *ConfigOverrides {
	overrides := &ConfigOverrides{}

	if v.IsSet("llm.claude_path") {
		value := v.GetString("llm.claude_path")
		overrides.LLM.ClaudePath = &value
	}
	if v.IsSet("llm.timeout") {
		value := v.GetString("llm.timeout")
		overrides.LLM.Timeout = &value
	}
	if v.IsSet("llm.max_retries") {
		value := v.GetInt("llm.max_retries")
		overrides.LLM.MaxRetries = &value
	}
	if v.IsSet("output.directory") {
		value := v.GetString("output.directory")
		overrides.Output.Directory = &value
	}
	if v.IsSet("output.format") {
		value := v.GetString("output.format")
		overrides.Output.Format = &value
	}
	if v.IsSet("output.overwrite") {
		value := v.GetBool("output.overwrite")
		overrides.Output.Overwrite = &value
	}
	if v.IsSet("gathering.confidence_threshold") {
		value := v.GetFloat64("gathering.confidence_threshold")
		overrides.Gathering.ConfidenceThreshold = &value
	}
	if v.IsSet("gathering.max_question_rounds") {
		value := v.GetInt("gathering.max_question_rounds")
		overrides.Gathering.MaxQuestionRounds = &value
	}
	if v.IsSet("generation.default_model") {
		value := v.GetString("generation.default_model")
		overrides.Generation.DefaultModel = &value
	}
	if v.IsSet("generation.default_temperature") {
		value := v.GetFloat64("generation.default_temperature")
		overrides.Generation.DefaultTemperature = &value
	}

	return overrides
}
