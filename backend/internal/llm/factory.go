package llm

import (
	"strings"
	"sync"

	"message-flow/backend/internal/llm/providers"
)

type Factory struct {
	mu        sync.Mutex
	instances map[string]Provider
}

func NewFactory() *Factory {
	return &Factory{instances: map[string]Provider{}}
}

func (f *Factory) CreateProvider(config *ProviderConfig) Provider {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := config.ProviderName + ":" + config.ModelName + ":" + config.BaseURL + ":" + config.AzureEndpoint + ":" + config.AzureDeployment
	if provider, ok := f.instances[key]; ok {
		return provider
	}

	var provider Provider
	// Normalize provider name to lowercase for matching
	name := strings.ToLower(config.ProviderName)
	switch name {
	case "claude", "anthropic":
		provider = providers.NewClaudeProvider(config)
	case "openai":
		provider = providers.NewOpenAIProvider(config)
	case "azure_openai", "azureopenai":
		provider = providers.NewOpenAIProvider(config)
	case "cohere":
		provider = providers.NewCohereProvider(config)
	case "google", "gemini":
		// Use OpenAI provider with Gemini-compatible base URL
		// User should configure base_url to point to Gemini API or use an OpenAI-compatible gateway
		provider = providers.NewOpenAIProvider(config)
	default:
		return nil
	}
	f.instances[key] = provider
	return provider
}
