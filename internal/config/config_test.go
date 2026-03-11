package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"MaxOpenConns", cfg.Database.MaxOpenConns, 25},
		{"MaxIdleConns", cfg.Database.MaxIdleConns, 5},
		{"ConnMaxLifetime", cfg.Database.ConnMaxLifetime, "5m"},
		{"OllamaURL", cfg.LLM.OllamaURL, "http://localhost:11434"},
		{"DefaultModel", cfg.LLM.DefaultModel, "gpt-4o"},
		{"EmbedModel", cfg.LLM.EmbedModel, "text-embedding-3-small"},
		{"ServerPort", cfg.Server.Port, 8080},
		{"ServerHost", cfg.Server.Host, "0.0.0.0"},
		{"AppImagePath", cfg.App.ImagePath, "./downloads"},
		{"FacebookSessionPath", cfg.Facebook.SessionPath, "~/.super-agent/fb-session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("got %v, want %v", tt.got, tt.expected)
			}
		})
	}
}

func TestApplyDefaults_DoesNotOverwrite(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			MaxOpenConns: 50,
			MaxIdleConns: 10,
		},
		LLM: LLMConfig{
			OllamaURL:    "http://custom:11434",
			DefaultModel: "gpt-4-turbo",
		},
		Server: ServerConfig{
			Port: 9090,
			Host: "127.0.0.1",
		},
	}
	applyDefaults(cfg)

	if cfg.Database.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns overwritten: got %d, want 50", cfg.Database.MaxOpenConns)
	}
	if cfg.LLM.OllamaURL != "http://custom:11434" {
		t.Errorf("OllamaURL overwritten: got %s", cfg.LLM.OllamaURL)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Port overwritten: got %d, want 9090", cfg.Server.Port)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp file for this test
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".super-agent.yaml")

	cfg := &Config{
		Database: DatabaseConfig{
			URL:          "postgres://test:test@localhost:5432/test",
			MaxOpenConns: 10,
		},
		LLM: LLMConfig{
			OpenAIKey: "sk-test-key-12345",
		},
	}
	applyDefaults(cfg)

	// Save to temp file
	data, err := yamlMarshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Read back
	readData, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	loaded := &Config{}
	if err := yamlUnmarshal(readData, loaded); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if loaded.Database.URL != cfg.Database.URL {
		t.Errorf("DB URL mismatch: got %s, want %s", loaded.Database.URL, cfg.Database.URL)
	}
	if loaded.LLM.OpenAIKey != cfg.LLM.OpenAIKey {
		t.Errorf("OpenAI key mismatch: got %s, want %s", loaded.LLM.OpenAIKey, cfg.LLM.OpenAIKey)
	}
	if loaded.Database.MaxOpenConns != 10 {
		t.Errorf("MaxOpenConns mismatch: got %d, want 10", loaded.Database.MaxOpenConns)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if path == "" {
		t.Error("DefaultConfigPath returned empty string")
	}
	if filepath.Base(path) != ".super-agent.yaml" {
		t.Errorf("unexpected config filename: %s", filepath.Base(path))
	}
}

// Wrappers to avoid import cycle with yaml.v3 being used directly
var yamlMarshal = func(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

var yamlUnmarshal = func(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
