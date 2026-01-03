package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/crypto"
	"message-flow/backend/internal/llm"
	"message-flow/backend/internal/models"
)

var errNotFound = errors.New("not found")

type providerRequest struct {
	ProviderName         string   `json:"provider_name"`
	APIKey               string   `json:"api_key"`
	ModelName            string   `json:"model_name"`
	Temperature          *float64 `json:"temperature"`
	MaxTokens            *int     `json:"max_tokens"`
	CostPer1KInput       *float64 `json:"cost_per_1k_input"`
	CostPer1KOutput      *float64 `json:"cost_per_1k_output"`
	MaxRequestsPerMinute *int     `json:"max_requests_per_minute"`
	IsActive             *bool    `json:"is_active"`
	IsDefault            *bool    `json:"is_default"`
}

type analyzeRequest struct {
	ProviderID int64  `json:"provider_id"`
	MessageID  *int64 `json:"message_id"`
	Message    string `json:"message"`
}

type batchAnalyzeRequest struct {
	Messages []struct {
		MessageID int64  `json:"message_id"`
		Content   string `json:"content"`
	} `json:"messages"`
}

type summarizeRequest struct {
	ProviderID     *int64   `json:"provider_id"`
	ConversationID *int64   `json:"conversation_id"`
	Messages       []string `json:"messages"`
}

func (a *API) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req providerRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ProviderName == "" || req.APIKey == "" {
		writeError(w, http.StatusBadRequest, "provider_name and api_key are required")
		return
	}
	if a.LLMStore == nil || a.LLMStore.MasterKey == "" {
		writeError(w, http.StatusInternalServerError, "master key not configured")
		return
	}
	config := defaultProviderConfig(req.ProviderName)
	if config == nil {
		writeError(w, http.StatusBadRequest, "unsupported provider")
		return
	}
	if req.ModelName != "" {
		config.ModelName = req.ModelName
	}
	if req.Temperature != nil {
		config.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		config.MaxTokens = *req.MaxTokens
	}
	if req.CostPer1KInput != nil {
		config.CostPer1KInput = *req.CostPer1KInput
	}
	if req.CostPer1KOutput != nil {
		config.CostPer1KOutput = *req.CostPer1KOutput
	}
	if req.MaxRequestsPerMinute != nil {
		config.MaxRequestsPerMinute = *req.MaxRequestsPerMinute
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	encrypted, err := crypto.Encrypt(a.LLMStore.MasterKey, req.APIKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt api key")
		return
	}

	var provider models.LLMProvider
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		if isDefault {
			_, _ = conn.Exec(ctx, `UPDATE llm_providers SET is_default=FALSE WHERE tenant_id=$1`, tenantID)
		}
		query := `
			INSERT INTO llm_providers (tenant_id, provider_name, api_key, model_name, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, is_active, is_default, health_status, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'unknown',$12)
			RETURNING id, tenant_id, provider_name, model_name, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, is_active, is_default, health_status, last_health_check, created_at`
		return conn.QueryRow(ctx, query, tenantID, config.ProviderName, encrypted, config.ModelName, config.Temperature, config.MaxTokens, config.CostPer1KInput, config.CostPer1KOutput, config.MaxRequestsPerMinute, isActive, isDefault, time.Now().UTC()).Scan(
			&provider.ID, &provider.TenantID, &provider.ProviderName, &provider.ModelName, &provider.Temperature, &provider.MaxTokens, &provider.CostPer1KInput, &provider.CostPer1KOutput, &provider.MaxRequestsPerMinute, &provider.IsActive, &provider.IsDefault, &provider.HealthStatus, &provider.LastHealthCheck, &provider.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create provider")
		return
	}

	provider.APIKey = "****"
	writeJSON(w, http.StatusCreated, provider)
	if a.HealthScheduler != nil {
		a.HealthScheduler.EnsureTenant(context.Background(), tenantID)
	}
}

func (a *API) ListProviders(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	providers := []models.LLMProvider{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, provider_name, model_name, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, is_active, is_default, health_status, last_health_check, created_at
			FROM llm_providers
			WHERE tenant_id=$1
			ORDER BY id DESC`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.LLMProvider
			if err := rows.Scan(&item.ID, &item.TenantID, &item.ProviderName, &item.ModelName, &item.Temperature, &item.MaxTokens, &item.CostPer1KInput, &item.CostPer1KOutput, &item.MaxRequestsPerMinute, &item.IsActive, &item.IsDefault, &item.HealthStatus, &item.LastHealthCheck, &item.CreatedAt); err != nil {
				return err
			}
			item.APIKey = "****"
			providers = append(providers, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list providers")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": providers})
}

func (a *API) GetProvider(w http.ResponseWriter, r *http.Request, providerID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var provider models.LLMProvider
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			SELECT id, tenant_id, provider_name, model_name, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, is_active, is_default, health_status, last_health_check, created_at
			FROM llm_providers WHERE tenant_id=$1 AND id=$2`
		return conn.QueryRow(ctx, query, tenantID, providerID).Scan(
			&provider.ID, &provider.TenantID, &provider.ProviderName, &provider.ModelName, &provider.Temperature, &provider.MaxTokens, &provider.CostPer1KInput, &provider.CostPer1KOutput, &provider.MaxRequestsPerMinute, &provider.IsActive, &provider.IsDefault, &provider.HealthStatus, &provider.LastHealthCheck, &provider.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}
	provider.APIKey = "****"
	writeJSON(w, http.StatusOK, provider)
}

func (a *API) UpdateProvider(w http.ResponseWriter, r *http.Request, providerID int64) {
	var req providerRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ProviderName != "" {
		if defaultProviderConfig(req.ProviderName) == nil {
			writeError(w, http.StatusBadRequest, "unsupported provider")
			return
		}
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var encrypted *string
	if req.APIKey != "" {
		if a.LLMStore == nil || a.LLMStore.MasterKey == "" {
			writeError(w, http.StatusInternalServerError, "master key not configured")
			return
		}
		value, err := crypto.Encrypt(a.LLMStore.MasterKey, req.APIKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to encrypt api key")
			return
		}
		encrypted = &value
	}

	var provider models.LLMProvider
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		if req.IsDefault != nil && *req.IsDefault {
			_, _ = conn.Exec(ctx, `UPDATE llm_providers SET is_default=FALSE WHERE tenant_id=$1`, tenantID)
		}
		query := `
			UPDATE llm_providers
			SET provider_name=COALESCE($1, provider_name),
			    api_key=COALESCE($2, api_key),
			    model_name=COALESCE($3, model_name),
			    temperature=COALESCE($4, temperature),
			    max_tokens=COALESCE($5, max_tokens),
			    cost_per_1k_input=COALESCE($6, cost_per_1k_input),
			    cost_per_1k_output=COALESCE($7, cost_per_1k_output),
			    max_requests_per_minute=COALESCE($8, max_requests_per_minute),
			    is_active=COALESCE($9, is_active),
			    is_default=COALESCE($10, is_default)
			WHERE tenant_id=$11 AND id=$12
			RETURNING id, tenant_id, provider_name, model_name, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, is_active, is_default, health_status, last_health_check, created_at`
		return conn.QueryRow(ctx, query, emptyString(req.ProviderName), encrypted, emptyString(req.ModelName), req.Temperature, req.MaxTokens, req.CostPer1KInput, req.CostPer1KOutput, req.MaxRequestsPerMinute, req.IsActive, req.IsDefault, tenantID, providerID).Scan(
			&provider.ID, &provider.TenantID, &provider.ProviderName, &provider.ModelName, &provider.Temperature, &provider.MaxTokens, &provider.CostPer1KInput, &provider.CostPer1KOutput, &provider.MaxRequestsPerMinute, &provider.IsActive, &provider.IsDefault, &provider.HealthStatus, &provider.LastHealthCheck, &provider.CreatedAt,
		)
	}); err != nil {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}
	provider.APIKey = "****"
	writeJSON(w, http.StatusOK, provider)
}

func (a *API) DeleteProvider(w http.ResponseWriter, r *http.Request, providerID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		command, err := conn.Exec(ctx, `DELETE FROM llm_providers WHERE tenant_id=$1 AND id=$2`, tenantID, providerID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return errNotFound
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) TestProvider(w http.ResponseWriter, r *http.Request, providerID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	result, err := a.LLM.HealthCheck(ctx, tenantID, providerID)
	status := "ok"
	if err != nil {
		status = "error"
	}
	var errMsg *string
	if err != nil {
		msg := err.Error()
		errMsg = &msg
	}
	latency := time.Duration(0)
	if result != nil {
		latency = result.Latency
	}
	_ = a.LLMStore.InsertHealth(ctx, tenantID, providerID, status, latency, errMsg, nil)
	if result == nil {
		writeError(w, http.StatusInternalServerError, "health check failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) AnalyzeMessage(w http.ResponseWriter, r *http.Request) {
	var req analyzeRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ProviderID == 0 || req.Message == "" {
		writeError(w, http.StatusBadRequest, "provider_id and message are required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	tenantID := a.tenantID(r)

	result, err := a.LLM.Analyze(ctx, tenantID, req.ProviderID, req.Message, req.MessageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "analysis failed")
		return
	}
	if req.MessageID != nil {
		_ = llm.StoreAnalysis(ctx, a.Store, tenantID, *req.MessageID, result)
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) BatchAnalyze(w http.ResponseWriter, r *http.Request) {
	var req batchAnalyzeRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages are required")
		return
	}
	if len(req.Messages) > 100 {
		writeError(w, http.StatusBadRequest, "batch size limit 100")
		return
	}
	ctx := r.Context()
	tenantID := a.tenantID(r)

	if a.Queue != nil {
		if a.WorkerScheduler != nil {
			a.WorkerScheduler.EnsureTenant(context.Background(), tenantID)
		}
		for _, msg := range req.Messages {
			_ = a.Queue.Enqueue(ctx, llm.QueueMessage{TenantID: tenantID, MessageID: msg.MessageID, Content: msg.Content, Feature: "analyze", CreatedAt: time.Now().UTC()})
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "queued", "count": len(req.Messages)})
		return
	}

	results := []map[string]any{}
	for _, msg := range req.Messages {
		result, err := a.LLM.AnalyzeWithFallback(ctx, tenantID, msg.Content, &msg.MessageID)
		if err != nil {
			results = append(results, map[string]any{"message_id": msg.MessageID, "error": err.Error()})
			continue
		}
		_ = llm.StoreAnalysis(ctx, a.Store, tenantID, msg.MessageID, result)
		results = append(results, map[string]any{"message_id": msg.MessageID, "analysis": result})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": results})
}

func (a *API) SummarizeConversation(w http.ResponseWriter, r *http.Request) {
	var req summarizeRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	messages := req.Messages
	conversationID := req.ConversationID
	if len(messages) == 0 && conversationID != nil {
		messages = []string{}
		if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
			rows, err := conn.Query(ctx, `
				SELECT content FROM messages WHERE tenant_id=$1 AND conversation_id=$2 ORDER BY timestamp ASC`, tenantID, *conversationID)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var content string
				if err := rows.Scan(&content); err != nil {
					return err
				}
				messages = append(messages, content)
			}
			return rows.Err()
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load messages")
			return
		}
	}
	if len(messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages or conversation_id required")
		return
	}

	providerID := int64(0)
	if req.ProviderID != nil {
		providerID = *req.ProviderID
	}
	if providerID == 0 {
		provider, err := a.LLM.Router.GetDefaultProvider(ctx, tenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "no default provider")
			return
		}
		providerID = provider.GetConfig().ID
	}

	result, err := a.LLM.Summarize(ctx, tenantID, providerID, messages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "summarize failed")
		return
	}

	if conversationID != nil {
		keyPoints, _ := json.Marshal(map[string]any{
			"key_points":   result.KeyPoints,
			"action_items": result.ActionItems,
			"sentiment":    result.Sentiment,
			"topics":       result.Topics,
		})
		_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
			_, err := conn.Exec(ctx, `
				INSERT INTO daily_summaries (tenant_id, conversation_id, summary_text, key_points_json, created_at)
				VALUES ($1,$2,$3,$4,$5)`, tenantID, *conversationID, result.Summary, string(keyPoints), time.Now().UTC())
			return err
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (a *API) GetUsageStats(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var stats llm.UsageStats
	var avgLatencyMs float64
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			SELECT COUNT(*) AS total,
			       SUM(CASE WHEN success THEN 1 ELSE 0 END) AS success,
			       SUM(CASE WHEN success THEN 0 ELSE 1 END) AS failed,
			       COALESCE(SUM(total_cost), 0),
			       COALESCE(AVG(response_time_ms), 0)
			FROM llm_usage_logs
			WHERE tenant_id=$1`
		return conn.QueryRow(ctx, query, tenantID).Scan(&stats.TotalRequests, &stats.SuccessfulRequests, &stats.FailedRequests, &stats.TotalCost, &avgLatencyMs)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load usage stats")
		return
	}
	stats.AverageLatency = time.Duration(avgLatencyMs) * time.Millisecond
	writeJSON(w, http.StatusOK, stats)
}

func (a *API) GetCosts(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	budget, _ := strconv.ParseFloat(r.URL.Query().Get("budget"), 64)

	providerCosts := []map[string]any{}
	featureCosts := []map[string]any{}
	dailyCosts := []map[string]any{}
	conversationCosts := []map[string]any{}
	var totalCost float64

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT p.provider_name, COALESCE(SUM(l.total_cost),0)
			FROM llm_usage_logs l
			JOIN llm_providers p ON p.id = l.provider_id
			WHERE l.tenant_id=$1
			GROUP BY p.provider_name
			ORDER BY 2 DESC`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var name string
			var cost float64
			if err := rows.Scan(&name, &cost); err != nil {
				rows.Close()
				return err
			}
			providerCosts = append(providerCosts, map[string]any{"provider": name, "total_cost": cost})
			totalCost += cost
		}
		rows.Close()

		rows, err = conn.Query(ctx, `
			SELECT feature_used, COALESCE(SUM(total_cost),0)
			FROM llm_usage_logs
			WHERE tenant_id=$1
			GROUP BY feature_used
			ORDER BY 2 DESC`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var feature string
			var cost float64
			if err := rows.Scan(&feature, &cost); err != nil {
				rows.Close()
				return err
			}
			featureCosts = append(featureCosts, map[string]any{"feature": feature, "total_cost": cost})
		}
		rows.Close()

		rows, err = conn.Query(ctx, `
			SELECT DATE_TRUNC('day', created_at) AS day, COALESCE(SUM(total_cost),0)
			FROM llm_usage_logs
			WHERE tenant_id=$1
			GROUP BY day
			ORDER BY day DESC
			LIMIT 30`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var day time.Time
			var cost float64
			if err := rows.Scan(&day, &cost); err != nil {
				rows.Close()
				return err
			}
			dailyCosts = append(dailyCosts, map[string]any{"day": day, "total_cost": cost})
		}
		rows.Close()

		rows, err = conn.Query(ctx, `
			SELECT m.conversation_id, COALESCE(SUM(l.total_cost),0)
			FROM llm_usage_logs l
			JOIN messages m ON m.id = l.message_id
			WHERE l.tenant_id=$1 AND l.message_id IS NOT NULL
			GROUP BY m.conversation_id
			ORDER BY 2 DESC
			LIMIT 20`, tenantID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var convoID int64
			var cost float64
			if err := rows.Scan(&convoID, &cost); err != nil {
				rows.Close()
				return err
			}
			conversationCosts = append(conversationCosts, map[string]any{"conversation_id": convoID, "total_cost": cost})
		}
		rows.Close()

		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load costs")
		return
	}

	alert := false
	if budget > 0 && totalCost >= budget*0.8 {
		alert = true
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_cost":         totalCost,
		"provider_costs":     providerCosts,
		"feature_costs":      featureCosts,
		"daily_costs":        dailyCosts,
		"conversation_costs": conversationCosts,
		"budget_alert":       alert,
	})
}

func (a *API) GetHealthStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	items := []map[string]any{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT p.id, p.provider_name, p.health_status, p.last_health_check,
			       COALESCE(AVG(h.latency_ms),0)
			FROM llm_providers p
			LEFT JOIN llm_provider_health h ON h.provider_id = p.id
			WHERE p.tenant_id=$1
			GROUP BY p.id, p.provider_name, p.health_status, p.last_health_check
			ORDER BY p.id DESC`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var name string
			var status string
			var lastCheck *time.Time
			var avgLatency float64
			if err := rows.Scan(&id, &name, &status, &lastCheck, &avgLatency); err != nil {
				return err
			}
			items = append(items, map[string]any{
				"provider_id":    id,
				"provider":       name,
				"status":         status,
				"last_check":     lastCheck,
				"avg_latency_ms": avgLatency,
			})
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load health")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func defaultProviderConfig(provider string) *llm.ProviderConfig {
	switch provider {
	case "claude":
		return &llm.ProviderConfig{
			ProviderName:         "claude",
			ModelName:            "claude-3-opus-20240229",
			Temperature:          0.2,
			MaxTokens:            1024,
			CostPer1KInput:       0.003,
			CostPer1KOutput:      0.015,
			MaxRequestsPerMinute: 60,
		}
	case "openai":
		return &llm.ProviderConfig{
			ProviderName:         "openai",
			ModelName:            "gpt-4-turbo",
			Temperature:          0.2,
			MaxTokens:            1024,
			CostPer1KInput:       0.01,
			CostPer1KOutput:      0.03,
			MaxRequestsPerMinute: 60,
		}
	case "cohere":
		return &llm.ProviderConfig{
			ProviderName:         "cohere",
			ModelName:            "command-r-plus",
			Temperature:          0.2,
			MaxTokens:            1024,
			CostPer1KInput:       0.0003,
			CostPer1KOutput:      0.0003,
			MaxRequestsPerMinute: 60,
		}
	default:
		return nil
	}
}

func emptyString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
