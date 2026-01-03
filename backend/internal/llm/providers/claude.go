package providers

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"message-flow/backend/internal/llm/contract"
)

type ClaudeProvider struct {
	client     anthropic.Client
	config     *contract.ProviderConfig
	retrier    Retrier
	lastUsage  contract.UsageStats
	lastRecord contract.UsageRecord
}

func NewClaudeProvider(config *contract.ProviderConfig) *ClaudeProvider {
	client := anthropic.NewClient(option.WithAPIKey(config.APIKey))
	return &ClaudeProvider{
		client:  client,
		config:  config,
		retrier: Retrier{Attempts: 3, Delay: 500 * time.Millisecond},
	}
}

func (c *ClaudeProvider) Name() string { return "claude" }

func (c *ClaudeProvider) GetConfig() *contract.ProviderConfig { return c.config }

func (c *ClaudeProvider) GetUsage(ctx context.Context) (*contract.UsageStats, error) {
	return &c.lastUsage, nil
}

func (c *ClaudeProvider) Analyze(ctx context.Context, message string) (*contract.AnalysisResult, error) {
	prompt := "Analyze this WhatsApp message JSON-only response with: is_important(bool),\npriority(high|medium|low), reason, has_action(bool), action_required,\nsentiment(positive|neutral|negative), sentiment_score(-1 to 1),\ntopics[], confidence(0-1)\n\nMessage: " + message
	var response *anthropic.Message
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	err := c.retrier.Do(ctx, func() error {
		start := time.Now()
		result, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:       anthropic.Model(c.config.ModelName),
			MaxTokens:   int64(c.config.MaxTokens),
			Temperature: anthropic.Float(c.config.Temperature),
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return err
		}
		response = result
		c.captureUsage("analyze", start, result.Usage)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.Content) == 0 {
		return nil, errors.New("empty response")
	}
	content := extractJSON(response.Content[0].Text)
	var parsed contract.AnalysisResult
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (c *ClaudeProvider) Summarize(ctx context.Context, messages []string) (*contract.SummaryResult, error) {
	prompt := "Summarize conversation with: summary, key_points[], action_items[], sentiment, topics[]\n\nMessages: " + joinLines(messages)
	var response *anthropic.Message
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	err := c.retrier.Do(ctx, func() error {
		start := time.Now()
		result, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:       anthropic.Model(c.config.ModelName),
			MaxTokens:   int64(c.config.MaxTokens),
			Temperature: anthropic.Float(c.config.Temperature),
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return err
		}
		response = result
		c.captureUsage("summarize", start, result.Usage)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.Content) == 0 {
		return nil, errors.New("empty response")
	}
	content := extractJSON(response.Content[0].Text)
	var parsed contract.SummaryResult
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (c *ClaudeProvider) ExtractActions(ctx context.Context, text string) ([]string, error) {
	prompt := "Extract action items as JSON array of strings\n\nText: " + text
	var response *anthropic.Message
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	err := c.retrier.Do(ctx, func() error {
		start := time.Now()
		result, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:       anthropic.Model(c.config.ModelName),
			MaxTokens:   int64(c.config.MaxTokens),
			Temperature: anthropic.Float(c.config.Temperature),
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
			},
		})
		if err != nil {
			return err
		}
		response = result
		c.captureUsage("extract_actions", start, result.Usage)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if response == nil || len(response.Content) == 0 {
		return nil, errors.New("empty response")
	}
	content := extractJSON(response.Content[0].Text)
	var parsed []string
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (c *ClaudeProvider) HealthCheck(ctx context.Context) (*contract.HealthCheckResult, error) {
	prompt := "Respond with: OK"
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	result, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       anthropic.Model(c.config.ModelName),
		MaxTokens:   int64(32),
		Temperature: anthropic.Float(0),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	_ = result
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

func (c *ClaudeProvider) captureUsage(feature string, start time.Time, usage anthropic.Usage) {
	latency := time.Since(start)
	record := contract.UsageRecord{
		InputTokens:  int(usage.InputTokens),
		OutputTokens: int(usage.OutputTokens),
		TotalTokens:  int(usage.InputTokens + usage.OutputTokens),
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

func (c *ClaudeProvider) lastUsageRecord() contract.UsageRecord {
	return c.lastRecord
}
