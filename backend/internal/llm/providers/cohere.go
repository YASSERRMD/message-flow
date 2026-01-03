package providers

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	cohere "github.com/cohere-ai/cohere-go"

	"message-flow/backend/internal/llm/contract"
)

type CohereProvider struct {
	client     *cohere.Client
	config     *contract.ProviderConfig
	retrier    Retrier
	lastUsage  contract.UsageStats
	lastRecord contract.UsageRecord
}

func NewCohereProvider(config *contract.ProviderConfig) *CohereProvider {
	client, _ := cohere.CreateClient(config.APIKey)
	return &CohereProvider{
		client:  client,
		config:  config,
		retrier: Retrier{Attempts: 3, Delay: 400 * time.Millisecond},
	}
}

func (c *CohereProvider) Name() string { return "cohere" }

func (c *CohereProvider) GetConfig() *contract.ProviderConfig { return c.config }

func (c *CohereProvider) GetUsage(ctx context.Context) (*contract.UsageStats, error) {
	return &c.lastUsage, nil
}

func (c *CohereProvider) Analyze(ctx context.Context, message string) (*contract.AnalysisResult, error) {
	if c.client == nil {
		return nil, errors.New("cohere client not initialized")
	}
	prompt := "Analyze this WhatsApp message JSON-only response with: is_important(bool), priority(high|medium|low), reason, has_action(bool), action_required, sentiment(positive|neutral|negative), sentiment_score(-1 to 1), topics[], confidence(0-1).\nMessage: " + message
	var response *cohere.GenerateResponse
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	err := c.retrier.Do(ctx, func() error {
		start := time.Now()
		maxTokens := uint(c.config.MaxTokens)
		temperature := c.config.Temperature
		result, err := c.client.Generate(cohere.GenerateOptions{
			Model:       c.config.ModelName,
			Prompt:      prompt,
			MaxTokens:   &maxTokens,
			Temperature: &temperature,
		})
		if err != nil {
			return err
		}
		response = result
		c.captureUsage("analyze", start)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.Generations) == 0 {
		return nil, errors.New("empty response")
	}
	content := extractJSON(response.Generations[0].Text)
	var parsed contract.AnalysisResult
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (c *CohereProvider) Summarize(ctx context.Context, messages []string) (*contract.SummaryResult, error) {
	if c.client == nil {
		return nil, errors.New("cohere client not initialized")
	}
	prompt := "Summarize conversation with: summary, key_points[], action_items[], sentiment, topics[]\nMessages: " + joinLines(messages)
	var response *cohere.GenerateResponse
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	err := c.retrier.Do(ctx, func() error {
		start := time.Now()
		maxTokens := uint(c.config.MaxTokens)
		temperature := c.config.Temperature
		result, err := c.client.Generate(cohere.GenerateOptions{
			Model:       c.config.ModelName,
			Prompt:      prompt,
			MaxTokens:   &maxTokens,
			Temperature: &temperature,
		})
		if err != nil {
			return err
		}
		response = result
		c.captureUsage("summarize", start)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.Generations) == 0 {
		return nil, errors.New("empty response")
	}
	content := extractJSON(response.Generations[0].Text)
	var parsed contract.SummaryResult
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (c *CohereProvider) ExtractActions(ctx context.Context, text string) ([]string, error) {
	if c.client == nil {
		return nil, errors.New("cohere client not initialized")
	}
	prompt := "Extract action items as JSON array of strings\nText: " + text
	var response *cohere.GenerateResponse
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	err := c.retrier.Do(ctx, func() error {
		start := time.Now()
		maxTokens := uint(c.config.MaxTokens)
		temperature := c.config.Temperature
		result, err := c.client.Generate(cohere.GenerateOptions{
			Model:       c.config.ModelName,
			Prompt:      prompt,
			MaxTokens:   &maxTokens,
			Temperature: &temperature,
		})
		if err != nil {
			return err
		}
		response = result
		c.captureUsage("extract_actions", start)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.Generations) == 0 {
		return nil, errors.New("empty response")
	}
	content := extractJSON(response.Generations[0].Text)
	var parsed []string
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (c *CohereProvider) HealthCheck(ctx context.Context) (*contract.HealthCheckResult, error) {
	if c.client == nil {
		return &contract.HealthCheckResult{
			Status:        "error",
			Latency:       0,
			EstimatedCost: 0,
			ErrorMessage:  "cohere client not initialized",
			Timestamp:     time.Now().UTC(),
		}, errors.New("cohere client not initialized")
	}
	prompt := "Respond with: OK"
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	start := time.Now()
	maxTokens := uint(10)
	temperature := 0.0
	_, err := c.client.Generate(cohere.GenerateOptions{
		Model:       c.config.ModelName,
		Prompt:      prompt,
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	})
	latency := time.Since(start)
	status := "ok"
	msg := ""
	if err != nil {
		status = "error"
		msg = err.Error()
	}
	return &contract.HealthCheckResult{
		Status:        status,
		Latency:       latency,
		EstimatedCost: 0,
		ErrorMessage:  msg,
		Timestamp:     time.Now().UTC(),
	}, err
}

func (c *CohereProvider) captureUsage(feature string, start time.Time) {
	latency := time.Since(start)
	record := contract.UsageRecord{
		InputTokens:  0,
		OutputTokens: 0,
		TotalTokens:  0,
		Latency:      latency,
		Success:      true,
		Feature:      feature,
	}
	c.lastRecord = record
	c.lastUsage.TotalRequests++
	c.lastUsage.SuccessfulRequests++
	c.lastUsage.TotalCost += record.TotalCost(c.config.CostPer1KInput, c.config.CostPer1KOutput)
	c.lastUsage.AverageLatency = averageLatency(c.lastUsage.AverageLatency, latency, c.lastUsage.SuccessfulRequests)
}

func (c *CohereProvider) lastUsageRecord() contract.UsageRecord {
	return c.lastRecord
}
