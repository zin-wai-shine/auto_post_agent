package embedding

import (
	"context"
	"fmt"
)

// Provider defines the interface for vector embedding generation.
type Provider interface {
	// Embed generates a vector embedding for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embeddings for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// ModelName returns the name of the embedding model being used.
	ModelName() string

	// Dimensions returns the embedding vector dimensions.
	Dimensions() int
}

// NewProvider creates an embedding provider based on the provider name.
func NewProvider(providerName, apiKey, modelName string) (Provider, error) {
	switch providerName {
	case "openai":
		return NewOpenAIProvider(apiKey, modelName), nil
	case "ollama":
		return NewOllamaProvider(apiKey, modelName), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", providerName)
	}
}

// OpenAIProvider implements the Provider interface using OpenAI's API.
type OpenAIProvider struct {
	apiKey    string
	modelName string
}

// NewOpenAIProvider creates a new OpenAI embedding provider.
func NewOpenAIProvider(apiKey, modelName string) *OpenAIProvider {
	if modelName == "" {
		modelName = "text-embedding-3-small"
	}
	return &OpenAIProvider{apiKey: apiKey, modelName: modelName}
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// TODO: Implement OpenAI embedding API call
	return nil, fmt.Errorf("OpenAI embedding not yet implemented")
}

func (p *OpenAIProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// TODO: Implement batch embedding
	return nil, fmt.Errorf("OpenAI batch embedding not yet implemented")
}

func (p *OpenAIProvider) ModelName() string { return p.modelName }
func (p *OpenAIProvider) Dimensions() int   { return 1536 }

// OllamaProvider implements the Provider interface using a local Ollama server.
type OllamaProvider struct {
	endpoint  string
	modelName string
}

// NewOllamaProvider creates a new Ollama embedding provider.
func NewOllamaProvider(endpoint, modelName string) *OllamaProvider {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if modelName == "" {
		modelName = "nomic-embed-text"
	}
	return &OllamaProvider{endpoint: endpoint, modelName: modelName}
}

func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// TODO: Implement Ollama embedding API call
	return nil, fmt.Errorf("Ollama embedding not yet implemented")
}

func (p *OllamaProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// TODO: Implement batch embedding
	return nil, fmt.Errorf("Ollama batch embedding not yet implemented")
}

func (p *OllamaProvider) ModelName() string { return p.modelName }
func (p *OllamaProvider) Dimensions() int   { return 768 }
