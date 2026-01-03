package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	TenantID     int64     `json:"tenant_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Conversation struct {
	ID            int64      `json:"id"`
	TenantID      int64      `json:"tenant_id"`
	ContactNumber string     `json:"contact_number"`
	ContactName   *string    `json:"contact_name"`
	LastMessageAt *time.Time `json:"last_message_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type Message struct {
	ID             int64     `json:"id"`
	TenantID       int64     `json:"tenant_id"`
	ConversationID int64     `json:"conversation_id"`
	Sender         string    `json:"sender"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
	MetadataJSON   *string   `json:"metadata_json"`
	CreatedAt      time.Time `json:"created_at"`
}

type DailySummary struct {
	ID             int64     `json:"id"`
	TenantID       int64     `json:"tenant_id"`
	ConversationID int64     `json:"conversation_id"`
	SummaryText    string    `json:"summary_text"`
	KeyPointsJSON  *string   `json:"key_points_json"`
	CreatedAt      time.Time `json:"created_at"`
}

type ImportantMessage struct {
	ID        int64     `json:"id"`
	TenantID  int64     `json:"tenant_id"`
	MessageID int64     `json:"message_id"`
	Priority  string    `json:"priority"`
	Reason    *string   `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type ActionItem struct {
	ID             int64      `json:"id"`
	TenantID       int64      `json:"tenant_id"`
	ConversationID int64      `json:"conversation_id"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	AssignedTo     *int64     `json:"assigned_to"`
	DueDate        *time.Time `json:"due_date"`
	CreatedAt      time.Time  `json:"created_at"`
}

type UserActivityLog struct {
	ID          int64     `json:"id"`
	TenantID    int64     `json:"tenant_id"`
	UserID      int64     `json:"user_id"`
	Action      string    `json:"action"`
	DetailsJSON *string   `json:"details_json"`
	CreatedAt   time.Time `json:"created_at"`
}

type LLMProvider struct {
	ID                   int64      `json:"id"`
	TenantID             int64      `json:"tenant_id"`
	ProviderName         string     `json:"provider_name"`
	APIKey               string     `json:"api_key"`
	ModelName            string     `json:"model_name"`
	DisplayName          *string    `json:"display_name"`
	Temperature          float64    `json:"temperature"`
	MaxTokens            int        `json:"max_tokens"`
	CostPer1KInput       float64    `json:"cost_per_1k_input"`
	CostPer1KOutput      float64    `json:"cost_per_1k_output"`
	MaxRequestsPerMinute int        `json:"max_requests_per_minute"`
	MaxRequestsPerDay    int        `json:"max_requests_per_day"`
	MonthlyBudget        *float64   `json:"monthly_budget"`
	IsActive             bool       `json:"is_active"`
	IsDefault            bool       `json:"is_default"`
	IsFallback           bool       `json:"is_fallback"`
	HealthStatus         string     `json:"health_status"`
	LastHealthCheck      *time.Time `json:"last_health_check"`
	CreatedAt            time.Time  `json:"created_at"`
}

type LLMUsageLog struct {
	ID             int64     `json:"id"`
	TenantID       int64     `json:"tenant_id"`
	ProviderID     int64     `json:"provider_id"`
	MessageID      *int64    `json:"message_id"`
	InputTokens    int       `json:"input_tokens"`
	OutputTokens   int       `json:"output_tokens"`
	TotalTokens    int       `json:"total_tokens"`
	InputCost      float64   `json:"input_cost"`
	OutputCost     float64   `json:"output_cost"`
	TotalCost      float64   `json:"total_cost"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	Success        bool      `json:"success"`
	ErrorMessage   *string   `json:"error_message"`
	FeatureUsed    string    `json:"feature_used"`
	CreatedAt      time.Time `json:"created_at"`
}

type LLMProviderHealth struct {
	ID             int64     `json:"id"`
	ProviderID     int64     `json:"provider_id"`
	TenantID       int64     `json:"tenant_id"`
	CheckTime      time.Time `json:"check_time"`
	Status         string    `json:"status"`
	LatencyMs      int64     `json:"latency_ms"`
	ErrorMessage   *string   `json:"error_message"`
	HTTPStatusCode *int      `json:"http_status_code"`
	CreatedAt      time.Time `json:"created_at"`
}

type LLMProviderHistory struct {
	ID         int64     `json:"id"`
	TenantID   int64     `json:"tenant_id"`
	ProviderID int64     `json:"provider_id"`
	ChangeJSON string    `json:"change_json"`
	ChangedBy  *int64    `json:"changed_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type LLMFeatureAssignment struct {
	ID          int64     `json:"id"`
	TenantID    int64     `json:"tenant_id"`
	FeatureName string    `json:"feature_name"`
	ProviderID  int64     `json:"provider_id"`
	Priority    int       `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
}

type DashboardSummary struct {
	TotalConversations int64 `json:"total_conversations"`
	TotalMessages      int64 `json:"total_messages"`
	ImportantMessages  int64 `json:"important_messages"`
	OpenActionItems    int64 `json:"open_action_items"`
}
