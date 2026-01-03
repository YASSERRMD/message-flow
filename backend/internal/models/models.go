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

type DashboardSummary struct {
	TotalConversations int64 `json:"total_conversations"`
	TotalMessages      int64 `json:"total_messages"`
	ImportantMessages  int64 `json:"important_messages"`
	OpenActionItems    int64 `json:"open_action_items"`
}
