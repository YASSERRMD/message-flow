package llm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Router struct {
	factory *Factory
	cache   *cache
	db      ProviderStore
}

type ProviderStore interface {
	ListProviders(ctx context.Context, tenantID int64) ([]ProviderConfig, error)
	GetDefaultProvider(ctx context.Context, tenantID int64) (*ProviderConfig, error)
	GetProviderByID(ctx context.Context, tenantID int64, providerID int64) (*ProviderConfig, error)
}

type cachedProvider struct {
	provider Provider
	expires  time.Time
}

type cache struct {
	mu    sync.Mutex
	items map[string]cachedProvider
	ttl   time.Duration
}

func newCache(ttl time.Duration) *cache {
	return &cache{items: map[string]cachedProvider{}, ttl: ttl}
}

func (c *cache) get(key string) (Provider, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok || time.Now().After(item.expires) {
		delete(c.items, key)
		return nil, false
	}
	return item.provider, true
}

func (c *cache) set(key string, provider Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cachedProvider{provider: provider, expires: time.Now().Add(c.ttl)}
}

func NewRouter(factory *Factory, store ProviderStore) *Router {
	return &Router{factory: factory, cache: newCache(5 * time.Minute), db: store}
}

func (r *Router) GetProvider(ctx context.Context, tenantID, providerID int64) (Provider, error) {
	key := cacheKey(tenantID, providerID)
	if provider, ok := r.cache.get(key); ok {
		return provider, nil
	}
	config, err := r.db.GetProviderByID(ctx, tenantID, providerID)
	if err != nil || config == nil {
		return nil, errors.New("provider not found")
	}
	provider := r.factory.CreateProvider(config)
	if provider == nil {
		return nil, errors.New("provider not supported")
	}
	r.cache.set(key, provider)
	return provider, nil
}

func (r *Router) GetDefaultProvider(ctx context.Context, tenantID int64) (Provider, error) {
	key := cacheKey(tenantID, 0)
	if provider, ok := r.cache.get(key); ok {
		return provider, nil
	}
	config, err := r.db.GetDefaultProvider(ctx, tenantID)
	if err != nil || config == nil {
		return nil, errors.New("default provider not found")
	}
	provider := r.factory.CreateProvider(config)
	if provider == nil {
		return nil, errors.New("provider not supported")
	}
	r.cache.set(key, provider)
	return provider, nil
}

func (r *Router) GetProviderForFeature(ctx context.Context, tenantID int64, feature string) (Provider, error) {
	provider, err := r.GetDefaultProvider(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (r *Router) AnalyzeWithFallback(ctx context.Context, tenantID int64, message string) (*AnalysisResult, Provider, int64, error) {
	configs, err := r.db.ListProviders(ctx, tenantID)
	if err != nil {
		return fallbackAnalysis(message), nil, 0, err
	}
	for _, cfg := range configs {
		provider := r.factory.CreateProvider(&cfg)
		if provider == nil {
			continue
		}
		result, err := provider.Analyze(ctx, message)
		if err == nil {
			return result, provider, cfg.ID, nil
		}
	}
	return fallbackAnalysis(message), nil, 0, errors.New("all providers failed")
}

func cacheKey(tenantID, providerID int64) string {
	return fmtKey(tenantID, providerID)
}

func fmtKey(tenantID, providerID int64) string {
	return fmt.Sprintf("%d:%d", tenantID, providerID)
}
