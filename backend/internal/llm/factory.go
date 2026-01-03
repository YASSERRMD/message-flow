package llm

import (
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

	key := config.ProviderName + ":" + config.ModelName
	if provider, ok := f.instances[key]; ok {
		return provider
	}

	var provider Provider
	switch config.ProviderName {
	case "claude":
		provider = providers.NewClaudeProvider(config)
	case "openai":
		provider = providers.NewOpenAIProvider(config)
	case "cohere":
		provider = providers.NewCohereProvider(config)
	default:
		return nil
	}
	f.instances[key] = provider
	return provider
}
