// Package config は agentic-shell の設定管理を提供します
// Viper ベースの構造化設定システム
package config

import (
	"fmt"
	"time"
)

// Config は agentic-shell のメイン設定構造体です
type Config struct {
	LLM        LLMConfig        `mapstructure:"llm" yaml:"llm"`
	Output     OutputConfig     `mapstructure:"output" yaml:"output"`
	Gathering  GatheringConfig  `mapstructure:"gathering" yaml:"gathering"`
	Generation GenerationConfig `mapstructure:"generation" yaml:"generation"`
}

// ConfigOverrides は設定の上書き値を表します。
// nil は「未設定」、ポインタ値はゼロ値を含めて「明示設定」を表します。
type ConfigOverrides struct {
	LLM        LLMConfigOverrides        `mapstructure:"llm" yaml:"llm"`
	Output     OutputConfigOverrides     `mapstructure:"output" yaml:"output"`
	Gathering  GatheringConfigOverrides  `mapstructure:"gathering" yaml:"gathering"`
	Generation GenerationConfigOverrides `mapstructure:"generation" yaml:"generation"`
}

// LLMConfigOverrides は LLM 設定の上書き値です。
type LLMConfigOverrides struct {
	ClaudePath *string `mapstructure:"claude_path" yaml:"claude_path"`
	Timeout    *string `mapstructure:"timeout" yaml:"timeout"`
	MaxRetries *int    `mapstructure:"max_retries" yaml:"max_retries"`
}

// OutputConfigOverrides は出力設定の上書き値です。
type OutputConfigOverrides struct {
	Directory *string `mapstructure:"directory" yaml:"directory"`
	Format    *string `mapstructure:"format" yaml:"format"`
	Overwrite *bool   `mapstructure:"overwrite" yaml:"overwrite"`
}

// GatheringConfigOverrides は情報収集設定の上書き値です。
type GatheringConfigOverrides struct {
	ConfidenceThreshold *float64 `mapstructure:"confidence_threshold" yaml:"confidence_threshold"`
	MaxQuestionRounds   *int     `mapstructure:"max_question_rounds" yaml:"max_question_rounds"`
}

// GenerationConfigOverrides は生成設定の上書き値です。
type GenerationConfigOverrides struct {
	DefaultModel       *string  `mapstructure:"default_model" yaml:"default_model"`
	DefaultTemperature *float64 `mapstructure:"default_temperature" yaml:"default_temperature"`
}

// LLMConfig は LLM 関連の設定です
type LLMConfig struct {
	// ClaudePath は Claude CLI のパスです
	ClaudePath string `mapstructure:"claude_path" yaml:"claude_path"`

	// Timeout は LLM リクエストのタイムアウト時間です
	Timeout string `mapstructure:"timeout" yaml:"timeout"`

	// MaxRetries は最大リトライ回数です
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries"`
}

// GetTimeout はタイムアウト設定を time.Duration として返します
func (c *LLMConfig) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}

// Validate は LLMConfig を検証します
func (c *LLMConfig) Validate() error {
	if c.ClaudePath == "" {
		return fmt.Errorf("llm.claude_path is required")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("llm.max_retries must be non-negative, got: %d", c.MaxRetries)
	}
	if _, err := c.GetTimeout(); err != nil {
		return fmt.Errorf("llm.timeout is invalid: %w", err)
	}
	return nil
}

// OutputConfig は出力関連の設定です
type OutputConfig struct {
	// Directory は生成されたファイルの出力ディレクトリです
	Directory string `mapstructure:"directory" yaml:"directory"`

	// Format は出力フォーマットです (markdown, yaml, json)
	Format string `mapstructure:"format" yaml:"format"`

	// Overwrite は既存ファイルを上書きするかどうかです
	Overwrite bool `mapstructure:"overwrite" yaml:"overwrite"`
}

// Validate は OutputConfig を検証します
func (c *OutputConfig) Validate() error {
	if c.Directory == "" {
		return fmt.Errorf("output.directory is required")
	}
	validFormats := map[string]bool{"markdown": true, "yaml": true, "json": true}
	if !validFormats[c.Format] {
		return fmt.Errorf("output.format must be one of [markdown, yaml, json], got: %s", c.Format)
	}
	return nil
}

// GatheringConfig は情報収集関連の設定です
type GatheringConfig struct {
	// ConfidenceThreshold は情報収集の信頼度閾値です (0.0-1.0)
	ConfidenceThreshold float64 `mapstructure:"confidence_threshold" yaml:"confidence_threshold"`

	// MaxQuestionRounds は最大質問ラウンド数です
	MaxQuestionRounds int `mapstructure:"max_question_rounds" yaml:"max_question_rounds"`
}

// Validate は GatheringConfig を検証します
func (c *GatheringConfig) Validate() error {
	if c.ConfidenceThreshold < 0 || c.ConfidenceThreshold > 1 {
		return fmt.Errorf("gathering.confidence_threshold must be between 0 and 1, got: %f", c.ConfidenceThreshold)
	}
	if c.MaxQuestionRounds < 1 {
		return fmt.Errorf("gathering.max_question_rounds must be at least 1, got: %d", c.MaxQuestionRounds)
	}
	return nil
}

// GenerationConfig は生成関連の設定です
type GenerationConfig struct {
	// DefaultModel はデフォルトで使用するモデルです
	DefaultModel string `mapstructure:"default_model" yaml:"default_model"`

	// DefaultTemperature はデフォルトの temperature です
	DefaultTemperature float64 `mapstructure:"default_temperature" yaml:"default_temperature"`
}

// Validate は GenerationConfig を検証します
func (c *GenerationConfig) Validate() error {
	if c.DefaultModel == "" {
		return fmt.Errorf("generation.default_model is required")
	}
	if c.DefaultTemperature < 0 || c.DefaultTemperature > 1 {
		return fmt.Errorf("generation.default_temperature must be between 0 and 1, got: %f", c.DefaultTemperature)
	}
	return nil
}

// Validate は Config 全体を検証します
func (c *Config) Validate() error {
	if err := c.LLM.Validate(); err != nil {
		return fmt.Errorf("llm: %w", err)
	}
	if err := c.Output.Validate(); err != nil {
		return fmt.Errorf("output: %w", err)
	}
	if err := c.Gathering.Validate(); err != nil {
		return fmt.Errorf("gathering: %w", err)
	}
	if err := c.Generation.Validate(); err != nil {
		return fmt.Errorf("generation: %w", err)
	}
	return nil
}

// DefaultConfig はデフォルト設定を返します
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			ClaudePath: "claude",
			Timeout:    "2m",
			MaxRetries: 3,
		},
		Output: OutputConfig{
			Directory: ".claude/agents",
			Format:    "markdown",
			Overwrite: false,
		},
		Gathering: GatheringConfig{
			ConfidenceThreshold: 0.85,
			MaxQuestionRounds:   5,
		},
		Generation: GenerationConfig{
			DefaultModel:       "claude-sonnet-4-6",
			DefaultTemperature: 0.7,
		},
	}
}

// Merge は上書き設定をこの設定にマージします。
func (c *Config) Merge(other *ConfigOverrides) {
	if other == nil {
		return
	}

	if other.LLM.ClaudePath != nil {
		c.LLM.ClaudePath = *other.LLM.ClaudePath
	}
	if other.LLM.Timeout != nil {
		c.LLM.Timeout = *other.LLM.Timeout
	}
	if other.LLM.MaxRetries != nil {
		c.LLM.MaxRetries = *other.LLM.MaxRetries
	}
	if other.Output.Directory != nil {
		c.Output.Directory = *other.Output.Directory
	}
	if other.Output.Format != nil {
		c.Output.Format = *other.Output.Format
	}
	if other.Output.Overwrite != nil {
		c.Output.Overwrite = *other.Output.Overwrite
	}
	if other.Gathering.ConfidenceThreshold != nil {
		c.Gathering.ConfidenceThreshold = *other.Gathering.ConfidenceThreshold
	}
	if other.Gathering.MaxQuestionRounds != nil {
		c.Gathering.MaxQuestionRounds = *other.Gathering.MaxQuestionRounds
	}
	if other.Generation.DefaultModel != nil {
		c.Generation.DefaultModel = *other.Generation.DefaultModel
	}
	if other.Generation.DefaultTemperature != nil {
		c.Generation.DefaultTemperature = *other.Generation.DefaultTemperature
	}
}
