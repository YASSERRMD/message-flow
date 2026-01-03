package contract

import (
	"context"
	"time"
)

type Provider interface {
	Name() string
	Analyze(ctx context.Context, message string) (*AnalysisResult, error)
	Summarize(ctx context.Context, messages []string) (*SummaryResult, error)
	ExtractActions(ctx context.Context, text string) ([]string, error)
	HealthCheck(ctx context.Context) (*HealthCheckResult, error)
	GetConfig() *ProviderConfig
	GetUsage(ctx context.Context) (*UsageStats, error)
}

type ProviderConfig struct {
	ID                   int64
	ProviderName         string
	APIKey               string
	ModelName            string
	BaseURL              string
	AzureEndpoint        string
	AzureDeployment      string
	AzureAPIVersion      string
	Temperature          float64
	MaxTokens            int
	CostPer1KInput       float64
	CostPer1KOutput      float64
	MaxRequestsPerMinute int
}

type AnalysisResult struct {
	IsImportant    bool     `json:"is_important"`
	Priority       string   `json:"priority"`
	Reason         string   `json:"reason"`
	HasAction      bool     `json:"has_action"`
	ActionRequired string   `json:"action_required"`
	Sentiment      string   `json:"sentiment"`
	SentimentScore float64  `json:"sentiment_score"`
	Topics         []string `json:"topics"`
	Confidence     float64  `json:"confidence"`
}

type SummaryResult struct {
	Summary     string   `json:"summary"`
	KeyPoints   []string `json:"key_points"`
	ActionItems []string `json:"action_items"`
	Sentiment   string   `json:"sentiment"`
	Topics      []string `json:"topics"`
}

type HealthCheckResult struct {
	Status        string        `json:"status"`
	Latency       time.Duration `json:"latency"`
	EstimatedCost float64       `json:"estimated_cost"`
	ErrorMessage  string        `json:"error_message"`
	Timestamp     time.Time     `json:"timestamp"`
}

type UsageStats struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	TotalCost          float64       `json:"total_cost"`
	AverageLatency     time.Duration `json:"average_latency"`
}

type UsageRecord struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Latency      time.Duration
	Success      bool
	ErrorMessage string
	Feature      string
}

func (u UsageRecord) InputCost(costPer1K float64) float64 {
	return (float64(u.InputTokens) / 1000.0) * costPer1K
}

func (u UsageRecord) OutputCost(costPer1K float64) float64 {
	return (float64(u.OutputTokens) / 1000.0) * costPer1K
}

func (u UsageRecord) TotalCost(costIn, costOut float64) float64 {
	return u.InputCost(costIn) + u.OutputCost(costOut)
}
