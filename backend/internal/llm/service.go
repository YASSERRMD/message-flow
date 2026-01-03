package llm

import (
	"context"
	"errors"
	"time"
)

type Service struct {
	Router *Router
	Store  *Store
}

type usageAware interface {
	lastUsageRecord() UsageRecord
}

func NewService(router *Router, store *Store) *Service {
	return &Service{Router: router, Store: store}
}

func (s *Service) Analyze(ctx context.Context, tenantID, providerID int64, message string, messageID *int64) (*AnalysisResult, error) {
	provider, err := s.Router.GetProvider(ctx, tenantID, providerID)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	result, err := provider.Analyze(ctx, message)
	record := usageFromProvider(provider, start, err, "analyze")
	_ = s.Store.InsertUsage(ctx, tenantID, providerID, messageID, record, provider.GetConfig().CostPer1KInput, provider.GetConfig().CostPer1KOutput)
	return result, err
}

func (s *Service) AnalyzeWithFallback(ctx context.Context, tenantID int64, message string, messageID *int64) (*AnalysisResult, error) {
	result, provider, providerID, err := s.Router.AnalyzeWithFallback(ctx, tenantID, message)
	if provider != nil {
		record := usageFromProvider(provider, time.Now(), err, "analyze")
		_ = s.Store.InsertUsage(ctx, tenantID, providerID, messageID, record, provider.GetConfig().CostPer1KInput, provider.GetConfig().CostPer1KOutput)
	}
	if err != nil && result != nil {
		return result, nil
	}
	return result, err
}

func (s *Service) Summarize(ctx context.Context, tenantID, providerID int64, messages []string) (*SummaryResult, error) {
	provider, err := s.Router.GetProvider(ctx, tenantID, providerID)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	result, err := provider.Summarize(ctx, messages)
	record := usageFromProvider(provider, start, err, "summarize")
	_ = s.Store.InsertUsage(ctx, tenantID, providerID, nil, record, provider.GetConfig().CostPer1KInput, provider.GetConfig().CostPer1KOutput)
	return result, err
}

func (s *Service) ExtractActions(ctx context.Context, tenantID, providerID int64, text string) ([]string, error) {
	provider, err := s.Router.GetProvider(ctx, tenantID, providerID)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	result, err := provider.ExtractActions(ctx, text)
	record := usageFromProvider(provider, start, err, "extract_actions")
	_ = s.Store.InsertUsage(ctx, tenantID, providerID, nil, record, provider.GetConfig().CostPer1KInput, provider.GetConfig().CostPer1KOutput)
	return result, err
}

func usageFromProvider(provider Provider, start time.Time, err error, feature string) UsageRecord {
	if aware, ok := provider.(usageAware); ok {
		record := aware.lastUsageRecord()
		record.Feature = feature
		if err != nil {
			record.Success = false
			record.ErrorMessage = err.Error()
		}
		if record.InputTokens == 0 && record.OutputTokens == 0 {
			record.Latency = time.Since(start)
		}
		return record
	}
	latency := time.Since(start)
	return UsageRecord{Latency: latency, Success: err == nil, ErrorMessage: errorString(err), Feature: feature}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Service) HealthCheck(ctx context.Context, tenantID, providerID int64) (*HealthCheckResult, error) {
	provider, err := s.Router.GetProvider(ctx, tenantID, providerID)
	if err != nil {
		return nil, err
	}
	result, err := provider.HealthCheck(ctx)
	if err != nil {
		return result, err
	}
	if result == nil {
		return nil, errors.New("no health result")
	}
	return result, nil
}
