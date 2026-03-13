package config

import (
	"fmt"
	"os"
	"path/filepath"

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

	// 設定ファイルを読み込み（存在しなくてもエラーにしない）
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// 設定ファイルが見つからない場合はデフォルト設定を使用
	}

	// 読み込んだ設定を構造体にマップ
	var fileConfig Config
	if err := v.Unmarshal(&fileConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// デフォルト設定とマージ
	config.Merge(&fileConfig)

	// 検証
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
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
