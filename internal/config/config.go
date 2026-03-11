package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the full application configuration.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	Facebook FacebookConfig `yaml:"facebook"`
	Server   ServerConfig   `yaml:"server"`
	App      AppConfig      `yaml:"app"`
}

// DatabaseConfig holds PostgreSQL connection details.
type DatabaseConfig struct {
	URL             string `yaml:"url"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}

// LLMConfig holds API keys for LLM providers (BYOK).
type LLMConfig struct {
	OpenAIKey    string `yaml:"openai_key"`
	ClaudeKey    string `yaml:"claude_key"`
	OllamaURL    string `yaml:"ollama_url"`
	DefaultModel string `yaml:"default_model"`
	EmbedModel   string `yaml:"embed_model"`
}

// FacebookConfig holds Facebook automation settings.
type FacebookConfig struct {
	SessionPath string   `yaml:"session_path"`
	BusinessURL string   `yaml:"business_url"`
	PageIDs     []string `yaml:"page_ids"`
	Email       string   `yaml:"email"`
	Password    string   `yaml:"password"`
}

// ServerConfig holds API server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	CORS bool   `yaml:"cors"`
}

// AppConfig holds general application settings.
type AppConfig struct {
	ImagePath string `yaml:"image_path"`
}

// DefaultConfigPath returns the default path to the config file.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".super-agent.yaml"
	}
	return filepath.Join(home, ".super-agent.yaml")
}

// Load reads and parses the configuration file.
func Load() (*Config, error) {
	path := DefaultConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	applyDefaults(cfg)
	return cfg, nil
}

// Save writes the configuration to the default config file.
func Save(cfg *Config) error {
	applyDefaults(cfg)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not serialize config: %w", err)
	}

	path := DefaultConfigPath()
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	return nil
}

// applyDefaults fills in sensible default values for unset fields.
func applyDefaults(cfg *Config) {
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Database.ConnMaxLifetime == "" {
		cfg.Database.ConnMaxLifetime = "5m"
	}
	if cfg.LLM.OllamaURL == "" {
		cfg.LLM.OllamaURL = "http://localhost:11434"
	}
	if cfg.LLM.DefaultModel == "" {
		cfg.LLM.DefaultModel = "gpt-4o"
	}
	if cfg.LLM.EmbedModel == "" {
		cfg.LLM.EmbedModel = "text-embedding-3-small"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.App.ImagePath == "" {
		cfg.App.ImagePath = "./downloads"
	}
	if cfg.Facebook.SessionPath == "" {
		cfg.Facebook.SessionPath = "~/.super-agent/fb-session"
	}
}
