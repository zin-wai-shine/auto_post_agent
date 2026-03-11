package embedding

import (
	"testing"
)

func TestNewProvider_OpenAI(t *testing.T) {
	p, err := NewProvider("openai", "sk-test", "text-embedding-3-small")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ModelName() != "text-embedding-3-small" {
		t.Errorf("model name mismatch: got %s", p.ModelName())
	}
	if p.Dimensions() != 1536 {
		t.Errorf("dimensions mismatch: got %d", p.Dimensions())
	}
}

func TestNewProvider_Ollama(t *testing.T) {
	p, err := NewProvider("ollama", "http://localhost:11434", "nomic-embed-text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ModelName() != "nomic-embed-text" {
		t.Errorf("model name mismatch: got %s", p.ModelName())
	}
	if p.Dimensions() != 768 {
		t.Errorf("dimensions mismatch: got %d", p.Dimensions())
	}
}

func TestNewProvider_DefaultModels(t *testing.T) {
	openai := NewOpenAIProvider("key", "")
	if openai.ModelName() != "text-embedding-3-small" {
		t.Errorf("default OpenAI model: got %s", openai.ModelName())
	}

	ollama := NewOllamaProvider("", "")
	if ollama.ModelName() != "nomic-embed-text" {
		t.Errorf("default Ollama model: got %s", ollama.ModelName())
	}
	if ollama.endpoint != "http://localhost:11434" {
		t.Errorf("default Ollama endpoint: got %s", ollama.endpoint)
	}
}

func TestNewProvider_Unsupported(t *testing.T) {
	_, err := NewProvider("gemini", "key", "model")
	if err == nil {
		t.Error("expected error for unsupported provider")
	}
}
