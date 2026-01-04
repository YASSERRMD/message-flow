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
	ID                int64      `json:"id"`
	TenantID          int64      `json:"tenant_id"`
	ContactNumber     string     `json:"contact_number"`
	ContactName       *string    `json:"contact_name"`
	LastMessageAt     *time.Time `json:"last_message_at"`
	CreatedAt         time.Time  `json:"created_at"`
	ProfilePictureURL *string    `json:"profile_picture_url"`
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
	WatchersJSON   *string    `json:"watchers_json"`
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
	BaseURL              *string    `json:"base_url"`
	AzureEndpoint        *string    `json:"azure_endpoint"`
	AzureDeployment      *string    `json:"azure_deployment"`
	AzureAPIVersion      *string    `json:"azure_api_version"`
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

type UserExtended struct {
	UserID    int64     `json:"user_id"`
	TenantID  int64     `json:"tenant_id"`
	Role      string    `json:"role"`
	TeamID    int64     `json:"team_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRole struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	TenantID  int64     `json:"tenant_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamMember struct {
	ID       int64     `json:"id"`
	TeamID   int64     `json:"team_id"`
	UserID   int64     `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type TeamInvitation struct {
	ID        int64      `json:"id"`
	TenantID  int64      `json:"tenant_id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	Token     string     `json:"token"`
	InvitedBy *int64     `json:"invited_by"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type Workflow struct {
	ID        int64     `json:"id"`
	TenantID  int64     `json:"tenant_id"`
	Name      string    `json:"name"`
	Trigger   string    `json:"trigger"`
	Actions   string    `json:"actions_json"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WorkflowExecution struct {
	ID            int64     `json:"id"`
	WorkflowID    int64     `json:"workflow_id"`
	ExecutionTime time.Time `json:"execution_time"`
	Success       bool      `json:"success"`
	ErrorMessage  *string   `json:"error_message"`
	CreatedAt     time.Time `json:"created_at"`
}

type Integration struct {
	ID        int64     `json:"id"`
	TenantID  int64     `json:"tenant_id"`
	Type      string    `json:"type"`
	Config    string    `json:"config_json"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID                int64     `json:"id"`
	TenantID          int64     `json:"tenant_id"`
	UserID            *int64    `json:"user_id"`
	Action            string    `json:"action"`
	ResourceType      *string   `json:"resource_type"`
	ResourceID        *int64    `json:"resource_id"`
	ChangesBeforeJSON *string   `json:"changes_before_json"`
	ChangesAfterJSON  *string   `json:"changes_after_json"`
	IPAddress         *string   `json:"ip_address"`
	UserAgent         *string   `json:"user_agent"`
	CreatedAt         time.Time `json:"created_at"`
}

type Label struct {
	ID        int64     `json:"id"`
	TenantID  int64     `json:"tenant_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type MessageLabel struct {
	MessageID int64     `json:"message_id"`
	LabelID   int64     `json:"label_id"`
	TenantID  int64     `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID           int64     `json:"id"`
	TenantID     int64     `json:"tenant_id"`
	ActionItemID int64     `json:"action_item_id"`
	UserID       int64     `json:"user_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

type Notification struct {
	ID        int64     `json:"id"`
	TenantID  int64     `json:"tenant_id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}
