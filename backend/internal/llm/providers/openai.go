package providers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"

	"message-flow/backend/internal/llm/contract"
)

type OpenAIProvider struct {
	client     openai.Client
	config     *contract.ProviderConfig
	retrier    Retrier
	lastUsage  contract.UsageStats
	lastRecord contract.UsageRecord
}

func NewOpenAIProvider(config *contract.ProviderConfig) *OpenAIProvider {
	client := openai.NewClient(option.WithAPIKey(config.APIKey))
	return &OpenAIProvider{
		client:  client,
		config:  config,
		retrier: Retrier{Attempts: 3, Delay: 400 * time.Millisecond},
	}
}

func (o *OpenAIProvider) Name() string { return "openai" }

func (o *OpenAIProvider) GetConfig() *contract.ProviderConfig { return o.config }

func (o *OpenAIProvider) GetUsage(ctx context.Context) (*contract.UsageStats, error) {
	return &o.lastUsage, nil
}

func (o *OpenAIProvider) Analyze(ctx context.Context, message string) (*contract.AnalysisResult, error) {
	prompt := "Analyze this WhatsApp message JSON-only response with: is_important(bool),\npriority(high|medium|low), reason, has_action(bool), action_required,\nsentiment(positive|neutral|negative), sentiment_score(-1 to 1),\ntopics[], confidence(0-1)\n\nMessage: " + message
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	var resp *openai.ChatCompletion
	err := o.retrier.Do(ctx, func() error {
		format := shared.NewResponseFormatJSONObjectParam()
		result, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:       shared.ChatModel(o.config.ModelName),
			Temperature: openai.Float(o.config.Temperature),
			MaxTokens:   openai.Int(int64(o.config.MaxTokens)),
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONObject: &format,
			},
			Messages: []openai.ChatCompletionMessageParamUnion{
				userMessage(prompt),
			},
		})
		if err != nil {
			if isRateLimitError(err) {
				return err
			}
			return err
		}
		resp = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	o.captureUsage("analyze", start, resp.Usage)
	if len(resp.Choices) == 0 {
		return nil, errors.New("empty response")
	}
	content := resp.Choices[0].Message.Content
	var parsed contract.AnalysisResult
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (o *OpenAIProvider) Summarize(ctx context.Context, messages []string) (*contract.SummaryResult, error) {
	prompt := "Summarize conversation with: summary, key_points[], action_items[], sentiment, topics[]\n\nMessages: " + joinLines(messages)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	var resp *openai.ChatCompletion
	err := o.retrier.Do(ctx, func() error {
		format := shared.NewResponseFormatJSONObjectParam()
		result, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:       shared.ChatModel(o.config.ModelName),
			Temperature: openai.Float(o.config.Temperature),
			MaxTokens:   openai.Int(int64(o.config.MaxTokens)),
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONObject: &format,
			},
			Messages: []openai.ChatCompletionMessageParamUnion{
				userMessage(prompt),
			},
		})
		if err != nil {
			if isRateLimitError(err) {
				return err
			}
			return err
		}
		resp = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	o.captureUsage("summarize", start, resp.Usage)
	if len(resp.Choices) == 0 {
		return nil, errors.New("empty response")
	}
	content := resp.Choices[0].Message.Content
	var parsed contract.SummaryResult
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (o *OpenAIProvider) ExtractActions(ctx context.Context, text string) ([]string, error) {
	prompt := "Extract action items as JSON object with actions array of strings\n\nText: " + text
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	var resp *openai.ChatCompletion
	err := o.retrier.Do(ctx, func() error {
		format := shared.NewResponseFormatJSONObjectParam()
		result, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:       shared.ChatModel(o.config.ModelName),
			Temperature: openai.Float(o.config.Temperature),
			MaxTokens:   openai.Int(int64(o.config.MaxTokens)),
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONObject: &format,
			},
			Messages: []openai.ChatCompletionMessageParamUnion{
				userMessage(prompt),
			},
		})
		if err != nil {
			if isRateLimitError(err) {
				return err
			}
			return err
		}
		resp = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	o.captureUsage("extract_actions", start, resp.Usage)
	if len(resp.Choices) == 0 {
		return nil, errors.New("empty response")
	}
	content := resp.Choices[0].Message.Content
	var payload struct {
		Actions []string `json:"actions"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, err
	}
	return payload.Actions, nil
}

func (o *OpenAIProvider) HealthCheck(ctx context.Context) (*contract.HealthCheckResult, error) {
	prompt := "Respond with: OK"
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	_, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       shared.ChatModel(o.config.ModelName),
		Temperature: openai.Float(0),
		Messages: []openai.ChatCompletionMessageParamUnion{
			userMessage(prompt),
		},
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

func (o *OpenAIProvider) captureUsage(feature string, start time.Time, usage openai.CompletionUsage) {
	latency := time.Since(start)
	record := contract.UsageRecord{
		InputTokens:  int(usage.PromptTokens),
		OutputTokens: int(usage.CompletionTokens),
		TotalTokens:  int(usage.TotalTokens),
		Latency:      latency,
		Success:      true,
		Feature:      feature,
	}
	o.lastRecord = record
	o.lastUsage.TotalRequests++
	o.lastUsage.SuccessfulRequests++
	o.lastUsage.TotalCost += record.TotalCost(o.config.CostPer1KInput, o.config.CostPer1KOutput)
	o.lastUsage.AverageLatency = averageLatency(o.lastUsage.AverageLatency, latency, o.lastUsage.SuccessfulRequests)
}

func (o *OpenAIProvider) lastUsageRecord() contract.UsageRecord {
	return o.lastRecord
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	type statusCoder interface {
		StatusCode() int
	}
	if sc, ok := err.(statusCoder); ok {
		return sc.StatusCode() == 429
	}
	if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") {
		return true
	}
	return false
}

func userMessage(content string) openai.ChatCompletionMessageParamUnion {
	return openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: openai.String(content),
			},
		},
	}
}
