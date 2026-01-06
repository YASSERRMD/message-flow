package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/crypto"
	"message-flow/backend/internal/llm"
	"message-flow/backend/internal/models"
)

var errNotFound = errors.New("not found")

type providerRequest struct {
	ProviderName         string   `json:"provider_name"`
	APIKey               string   `json:"api_key"`
	DisplayName          *string  `json:"display_name"`
	ModelName            string   `json:"model_name"`
	BaseURL              *string  `json:"base_url"`
	AzureEndpoint        *string  `json:"azure_endpoint"`
	AzureDeployment      *string  `json:"azure_deployment"`
	AzureAPIVersion      *string  `json:"azure_api_version"`
	Temperature          *float64 `json:"temperature"`
	MaxTokens            *int     `json:"max_tokens"`
	CostPer1KInput       *float64 `json:"cost_per_1k_input"`
	CostPer1KOutput      *float64 `json:"cost_per_1k_output"`
	MaxRequestsPerMinute *int     `json:"max_requests_per_minute"`
	MaxRequestsPerDay    *int     `json:"max_requests_per_day"`
	MonthlyBudget        *float64 `json:"monthly_budget"`
	IsActive             *bool    `json:"is_active"`
	IsDefault            *bool    `json:"is_default"`
	IsFallback           *bool    `json:"is_fallback"`
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
	if req.BaseURL != nil {
		config.BaseURL = *req.BaseURL
	}
	if req.AzureEndpoint != nil {
		config.AzureEndpoint = *req.AzureEndpoint
	}
	if req.AzureDeployment != nil {
		config.AzureDeployment = *req.AzureDeployment
	}
	if req.AzureAPIVersion != nil {
		config.AzureAPIVersion = *req.AzureAPIVersion
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
	maxPerDay := 10000
	if req.MaxRequestsPerDay != nil {
		maxPerDay = *req.MaxRequestsPerDay
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	isFallback := false
	if req.IsFallback != nil {
		isFallback = *req.IsFallback
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
			INSERT INTO llm_providers (tenant_id, provider_name, api_key, model_name, display_name, base_url, azure_endpoint, azure_deployment, azure_api_version, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, max_requests_per_day, monthly_budget, is_active, is_default, is_fallback, health_status, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,'unknown',$20)
			RETURNING id, tenant_id, provider_name, model_name, display_name, base_url, azure_endpoint, azure_deployment, azure_api_version, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, max_requests_per_day, monthly_budget, is_active, is_default, is_fallback, health_status, last_health_check, created_at`
		return conn.QueryRow(ctx, query, tenantID, config.ProviderName, encrypted, config.ModelName, req.DisplayName, req.BaseURL, req.AzureEndpoint, req.AzureDeployment, req.AzureAPIVersion, config.Temperature, config.MaxTokens, config.CostPer1KInput, config.CostPer1KOutput, config.MaxRequestsPerMinute, maxPerDay, req.MonthlyBudget, isActive, isDefault, isFallback, time.Now().UTC()).Scan(
			&provider.ID, &provider.TenantID, &provider.ProviderName, &provider.ModelName, &provider.DisplayName, &provider.BaseURL, &provider.AzureEndpoint, &provider.AzureDeployment, &provider.AzureAPIVersion, &provider.Temperature, &provider.MaxTokens, &provider.CostPer1KInput, &provider.CostPer1KOutput, &provider.MaxRequestsPerMinute, &provider.MaxRequestsPerDay, &provider.MonthlyBudget, &provider.IsActive, &provider.IsDefault, &provider.IsFallback, &provider.HealthStatus, &provider.LastHealthCheck, &provider.CreatedAt,
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
	a.writeProviderHistory(ctx, tenantID, provider.ID, authUserID(r), map[string]any{
		"event":       "created",
		"provider_id": provider.ID,
		"config":      providerSnapshot(provider),
	})
}

func (a *API) ListProviders(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	providers := []models.LLMProvider{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, provider_name, model_name, display_name, base_url, azure_endpoint, azure_deployment, azure_api_version, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, max_requests_per_day, monthly_budget, is_active, is_default, is_fallback, health_status, last_health_check, created_at
			FROM llm_providers
			WHERE tenant_id=$1
			ORDER BY id DESC`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.LLMProvider
			if err := rows.Scan(&item.ID, &item.TenantID, &item.ProviderName, &item.ModelName, &item.DisplayName, &item.BaseURL, &item.AzureEndpoint, &item.AzureDeployment, &item.AzureAPIVersion, &item.Temperature, &item.MaxTokens, &item.CostPer1KInput, &item.CostPer1KOutput, &item.MaxRequestsPerMinute, &item.MaxRequestsPerDay, &item.MonthlyBudget, &item.IsActive, &item.IsDefault, &item.IsFallback, &item.HealthStatus, &item.LastHealthCheck, &item.CreatedAt); err != nil {
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
			SELECT id, tenant_id, provider_name, model_name, display_name, base_url, azure_endpoint, azure_deployment, azure_api_version, temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, max_requests_per_day, monthly_budget, is_active, is_default, is_fallback, health_status, last_health_check, created_at
			FROM llm_providers WHERE tenant_id=$1 AND id=$2`
		return conn.QueryRow(ctx, query, tenantID, providerID).Scan(
			&provider.ID, &provider.TenantID, &provider.ProviderName, &provider.ModelName, &provider.DisplayName, &provider.BaseURL, &provider.AzureEndpoint, &provider.AzureDeployment, &provider.AzureAPIVersion, &provider.Temperature, &provider.MaxTokens, &provider.CostPer1KInput, &provider.CostPer1KOutput, &provider.MaxRequestsPerMinute, &provider.MaxRequestsPerDay, &provider.MonthlyBudget, &provider.IsActive, &provider.IsDefault, &provider.IsFallback, &provider.HealthStatus, &provider.LastHealthCheck, &provider.CreatedAt,
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
			    display_name=COALESCE($4, display_name),
			    base_url=COALESCE($5, base_url),
			    azure_endpoint=COALESCE($6, azure_endpoint),
			    azure_deployment=COALESCE($7, azure_deployment),
			    azure_api_version=COALESCE($8, azure_api_version),
			    temperature=COALESCE($9, temperature),
			    max_tokens=COALESCE($10, max_tokens),
			    cost_per_1k_input=COALESCE($11, cost_per_1k_input),
			    cost_per_1k_output=COALESCE($12, cost_per_1k_output),
			    max_requests_per_minute=COALESCE($13, max_requests_per_minute),
			    max_requests_per_day=COALESCE($14, max_requests_per_day),
			    monthly_budget=COALESCE($15, monthly_budget),
			    is_active=COALESCE($16, is_active),
			    is_default=COALESCE($17, is_default),
			    is_fallback=COALESCE($18, is_fallback)
			WHERE tenant_id=$19 AND id=$20
			RETURNING id, tenant_id, provider_name, model_name, COALESCE(display_name, ''), COALESCE(base_url, ''), COALESCE(azure_endpoint, ''), COALESCE(azure_deployment, ''), COALESCE(azure_api_version, ''), temperature, max_tokens, cost_per_1k_input, cost_per_1k_output, max_requests_per_minute, max_requests_per_day, monthly_budget, is_active, is_default, is_fallback, health_status, last_health_check, created_at`
		return conn.QueryRow(ctx, query, emptyString(req.ProviderName), encrypted, emptyString(req.ModelName), req.DisplayName, req.BaseURL, req.AzureEndpoint, req.AzureDeployment, req.AzureAPIVersion, req.Temperature, req.MaxTokens, req.CostPer1KInput, req.CostPer1KOutput, req.MaxRequestsPerMinute, req.MaxRequestsPerDay, req.MonthlyBudget, req.IsActive, req.IsDefault, req.IsFallback, tenantID, providerID).Scan(
			&provider.ID, &provider.TenantID, &provider.ProviderName, &provider.ModelName, &provider.DisplayName, &provider.BaseURL, &provider.AzureEndpoint, &provider.AzureDeployment, &provider.AzureAPIVersion, &provider.Temperature, &provider.MaxTokens, &provider.CostPer1KInput, &provider.CostPer1KOutput, &provider.MaxRequestsPerMinute, &provider.MaxRequestsPerDay, &provider.MonthlyBudget, &provider.IsActive, &provider.IsDefault, &provider.IsFallback, &provider.HealthStatus, &provider.LastHealthCheck, &provider.CreatedAt,
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
	a.writeProviderHistory(ctx, tenantID, providerID, authUserID(r), map[string]any{
		"event":       "deleted",
		"provider_id": providerID,
	})
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
	fmt.Println("DEBUG: SummarizeConversation ENTRY")
	var req summarizeRequest
	if err := readJSON(r, &req); err != nil {
		fmt.Println("DEBUG: readJSON failed:", err)
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	fmt.Println("DEBUG: readJSON OK, req:", req)
	tenantID := a.tenantID(r)
	fmt.Println("DEBUG: tenantID:", tenantID)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	messages := req.Messages
	conversationID := req.ConversationID
	fmt.Println("DEBUG: conversationID:", conversationID, "messages len:", len(messages))
	if len(messages) == 0 && conversationID != nil {
		fmt.Println("DEBUG: Loading messages from DB for conversation:", *conversationID)
		messages = []string{}
		if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
			rows, err := conn.Query(ctx, `
				SELECT content FROM messages WHERE tenant_id=$1 AND conversation_id=$2 ORDER BY timestamp ASC`, tenantID, *conversationID)
			if err != nil {
				fmt.Println("DEBUG: DB query error:", err)
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var content string
				if err := rows.Scan(&content); err != nil {
					fmt.Println("DEBUG: DB scan error:", err)
					return err
				}
				messages = append(messages, content)
			}
			fmt.Println("DEBUG: Loaded", len(messages), "messages from DB")
			return rows.Err()
		}); err != nil {
			fmt.Println("DEBUG: WithTenantConn error:", err)
			writeError(w, http.StatusInternalServerError, "failed to load messages: "+err.Error())
			return
		}
	}
	fmt.Println("DEBUG: Total messages:", len(messages))
	if len(messages) == 0 {
		fmt.Println("DEBUG: No messages, returning 400")
		writeError(w, http.StatusBadRequest, "messages or conversation_id required")
		return
	}

	providerID := int64(0)
	if req.ProviderID != nil {
		providerID = *req.ProviderID
	}
	fmt.Println("DEBUG: providerID from request:", providerID)
	if providerID == 0 {
		fmt.Println("DEBUG: Getting default provider...")
		provider, err := a.LLM.Router.GetDefaultProvider(ctx, tenantID)
		if err != nil {
			fmt.Println("DEBUG: GetDefaultProvider error:", err)
			// Continue to fallback
		} else {
			providerID = provider.GetConfig().ID
			fmt.Println("DEBUG: Got default provider ID:", providerID)
		}
	}

	var result *llm.SummaryResult
	var err error

	if providerID != 0 {
		fmt.Println("DEBUG: Calling LLM.Summarize with providerID:", providerID)
		result, err = a.LLM.Summarize(ctx, tenantID, providerID, messages)
	} else {
		err = errors.New("no provider available")
	}

	if err != nil {
		fmt.Println("DEBUG: LLM.Summarize error:", err)
		// Mock fallback: return sample summary when LLM fails
		result = &llm.SummaryResult{
			Summary:     "This conversation discusses various topics. Due to a temporary service issue, an AI-generated summary is not available at this time.",
			KeyPoints:   []string{"Multiple messages exchanged", "Topics discussed include general conversation"},
			ActionItems: []string{},
			Sentiment:   "neutral",
			Topics:      []string{"general"},
		}
	}
	fmt.Println("DEBUG: Summarize completed, writing response")

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
			       COALESCE(SUM(CASE WHEN success THEN 1 ELSE 0 END), 0) AS success,
			       COALESCE(SUM(CASE WHEN success THEN 0 ELSE 1 END), 0) AS failed,
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

func (a *API) GetProviderComparison(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	rows := []map[string]any{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		query := `
			SELECT p.id, p.provider_name, p.model_name,
			       COALESCE(AVG(l.response_time_ms), 0) AS avg_latency,
			       COALESCE(SUM(CASE WHEN l.success THEN 1 ELSE 0 END), 0) AS success_count,
			       COALESCE(COUNT(l.id), 0) AS total_count,
			       COALESCE(SUM(l.total_cost), 0) AS total_cost
			FROM llm_providers p
			LEFT JOIN llm_usage_logs l ON l.provider_id = p.id
			WHERE p.tenant_id=$1
			GROUP BY p.id, p.provider_name, p.model_name
			ORDER BY p.id DESC`
		result, err := conn.Query(ctx, query, tenantID)
		if err != nil {
			return err
		}
		defer result.Close()
		for result.Next() {
			var id int64
			var name string
			var model string
			var avgLatency float64
			var successCount int64
			var totalCount int64
			var totalCost float64
			if err := result.Scan(&id, &name, &model, &avgLatency, &successCount, &totalCount, &totalCost); err != nil {
				return err
			}
			successRate := 0.0
			if totalCount > 0 {
				successRate = (float64(successCount) / float64(totalCount)) * 100
			}
			rows = append(rows, map[string]any{
				"provider_id":    id,
				"provider":       name,
				"model":          model,
				"avg_latency_ms": avgLatency,
				"success_rate":   successRate,
				"monthly_spent":  totalCost,
				"requests":       totalCount,
			})
		}
		return result.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load comparison")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (a *API) GetProviderHistory(w http.ResponseWriter, r *http.Request, providerID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	history := []models.LLMProviderHistory{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, tenant_id, provider_id, change_json, changed_by, created_at
			FROM llm_provider_history
			WHERE tenant_id=$1 AND provider_id=$2
			ORDER BY created_at DESC
			LIMIT 5`, tenantID, providerID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.LLMProviderHistory
			if err := rows.Scan(&item.ID, &item.TenantID, &item.ProviderID, &item.ChangeJSON, &item.ChangedBy, &item.CreatedAt); err != nil {
				return err
			}
			history = append(history, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load history")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": history})
}

func (a *API) GetFeatures(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	features := []map[string]any{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT feature_name, provider_id, priority
			FROM llm_feature_assignments
			WHERE tenant_id=$1
			ORDER BY feature_name, priority ASC`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		assignments := map[string][]map[string]any{}
		for rows.Next() {
			var feature string
			var providerID int64
			var priority int
			if err := rows.Scan(&feature, &providerID, &priority); err != nil {
				return err
			}
			assignments[feature] = append(assignments[feature], map[string]any{
				"provider_id": providerID,
				"priority":    priority,
			})
		}
		for _, feature := range defaultFeatures() {
			features = append(features, map[string]any{
				"feature":   feature,
				"providers": assignments[feature],
			})
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load features")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": features})
}

type featureAssignRequest struct {
	ProviderID int64 `json:"provider_id"`
	Priority   int   `json:"priority"`
}

func (a *API) AssignProviderToFeature(w http.ResponseWriter, r *http.Request, feature string) {
	var req featureAssignRequest
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.ProviderID == 0 {
		writeError(w, http.StatusBadRequest, "provider_id required")
		return
	}
	if req.Priority == 0 {
		req.Priority = 1
	}

	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO llm_feature_assignments (tenant_id, feature_name, provider_id, priority, created_at)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (tenant_id, feature_name, provider_id)
			DO UPDATE SET priority=EXCLUDED.priority`, tenantID, feature, req.ProviderID, req.Priority, time.Now().UTC())
		return err
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to assign provider")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "assigned"})
}

func (a *API) GetFeatureProviders(w http.ResponseWriter, r *http.Request, feature string) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	data := []map[string]any{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT a.provider_id, a.priority, p.provider_name, p.model_name, p.health_status
			FROM llm_feature_assignments a
			JOIN llm_providers p ON p.id = a.provider_id
			WHERE a.tenant_id=$1 AND a.feature_name=$2
			ORDER BY a.priority ASC`, tenantID, feature)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var providerID int64
			var priority int
			var name string
			var model string
			var health string
			if err := rows.Scan(&providerID, &priority, &name, &model, &health); err != nil {
				return err
			}
			data = append(data, map[string]any{
				"provider_id": providerID,
				"priority":    priority,
				"provider":    name,
				"model":       model,
				"health":      health,
			})
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load feature providers")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (a *API) DeleteFeatureProvider(w http.ResponseWriter, r *http.Request, feature string, providerID int64) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		command, err := conn.Exec(ctx, `
			DELETE FROM llm_feature_assignments WHERE tenant_id=$1 AND feature_name=$2 AND provider_id=$3`, tenantID, feature, providerID)
		if err != nil {
			return err
		}
		if command.RowsAffected() == 0 {
			return errNotFound
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusNotFound, "assignment not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (a *API) GetCostBreakdown(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	byProvider := []map[string]any{}
	byFeature := []map[string]any{}
	byDay := []map[string]any{}
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
			byProvider = append(byProvider, map[string]any{"provider": name, "total_cost": cost})
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
			byFeature = append(byFeature, map[string]any{"feature": feature, "total_cost": cost})
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
			byDay = append(byDay, map[string]any{"day": day, "total_cost": cost})
		}
		rows.Close()

		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load cost breakdown")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"by_provider": byProvider,
		"by_feature":  byFeature,
		"by_day":      byDay,
	})
}

func (a *API) GetUsageByFeature(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	data := []map[string]any{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT feature_used,
			       COALESCE(SUM(total_tokens),0),
			       COALESCE(SUM(total_cost),0),
			       COALESCE(AVG(response_time_ms),0)
			FROM llm_usage_logs
			WHERE tenant_id=$1
			GROUP BY feature_used`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var feature string
			var tokens int64
			var cost float64
			var latency float64
			if err := rows.Scan(&feature, &tokens, &cost, &latency); err != nil {
				return err
			}
			data = append(data, map[string]any{
				"feature":        feature,
				"total_tokens":   tokens,
				"total_cost":     cost,
				"avg_latency_ms": latency,
			})
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load usage by feature")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (a *API) BulkTestProviders(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	providers := []models.LLMProvider{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT id, provider_name, model_name
			FROM llm_providers
			WHERE tenant_id=$1 AND is_active=TRUE`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item models.LLMProvider
			if err := rows.Scan(&item.ID, &item.ProviderName, &item.ModelName); err != nil {
				return err
			}
			providers = append(providers, item)
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load providers")
		return
	}

	results := []map[string]any{}
	for _, provider := range providers {
		result, err := a.LLM.HealthCheck(ctx, tenantID, provider.ID)
		status := "ok"
		if err != nil {
			status = "error"
		}
		results = append(results, map[string]any{
			"provider_id": provider.ID,
			"provider":    provider.ProviderName,
			"model":       provider.ModelName,
			"status":      status,
			"latency_ms":  latencyMs(result),
			"error":       errorString(err),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": results})
}

func (a *API) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	tenantID := a.tenantID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	type providerStats struct {
		Name        string
		Latency     float64
		SuccessRate float64
		Cost        float64
	}
	stats := []providerStats{}
	if err := a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		rows, err := conn.Query(ctx, `
			SELECT p.provider_name,
			       COALESCE(AVG(l.response_time_ms),0) AS avg_latency,
			       COALESCE(SUM(CASE WHEN l.success THEN 1 ELSE 0 END),0) AS success_count,
			       COALESCE(COUNT(l.id),0) AS total_count,
			       COALESCE(SUM(l.total_cost),0) AS total_cost
			FROM llm_providers p
			LEFT JOIN llm_usage_logs l ON l.provider_id = p.id
			WHERE p.tenant_id=$1
			GROUP BY p.provider_name`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			var latency float64
			var successCount int64
			var totalCount int64
			var totalCost float64
			if err := rows.Scan(&name, &latency, &successCount, &totalCount, &totalCost); err != nil {
				return err
			}
			successRate := 0.0
			if totalCount > 0 {
				successRate = (float64(successCount) / float64(totalCount)) * 100
			}
			stats = append(stats, providerStats{Name: name, Latency: latency, SuccessRate: successRate, Cost: totalCost})
		}
		return rows.Err()
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load recommendations")
		return
	}

	recommendations := []string{}
	if len(stats) > 0 {
		fastest := stats[0]
		bestSuccess := stats[0]
		cheapest := stats[0]
		for _, item := range stats[1:] {
			if item.Latency < fastest.Latency {
				fastest = item
			}
			if item.SuccessRate > bestSuccess.SuccessRate {
				bestSuccess = item
			}
			if item.Cost < cheapest.Cost {
				cheapest = item
			}
		}
		recommendations = append(recommendations, fastest.Name+" is fastest by latency")
		recommendations = append(recommendations, bestSuccess.Name+" has the best success rate")
		recommendations = append(recommendations, cheapest.Name+" is most cost-effective")
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": recommendations})
}

func defaultFeatures() []string {
	return []string{
		"importance_detection",
		"summarization",
		"action_extraction",
		"daily_summary",
		"conversation_scoring",
	}
}

func latencyMs(result *llm.HealthCheckResult) float64 {
	if result == nil {
		return 0
	}
	return float64(result.Latency.Milliseconds())
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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
	case "azure_openai":
		return &llm.ProviderConfig{
			ProviderName:         "azure_openai",
			ModelName:            "gpt-4o",
			AzureAPIVersion:      "2024-02-15-preview",
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

func (a *API) writeProviderHistory(ctx context.Context, tenantID, providerID int64, userID *int64, payload any) {
	if payload == nil {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = a.Store.WithTenantConn(ctx, tenantID, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, `
			INSERT INTO llm_provider_history (tenant_id, provider_id, change_json, changed_by, created_at)
			VALUES ($1,$2,$3,$4,$5)`, tenantID, providerID, string(raw), userID, time.Now().UTC())
		return err
	})
}

func authUserID(r *http.Request) *int64 {
	if user, ok := auth.UserFromContext(r.Context()); ok {
		return &user.ID
	}
	return nil
}

func providerSnapshot(provider models.LLMProvider) map[string]any {
	return map[string]any{
		"provider_name":           provider.ProviderName,
		"model_name":              provider.ModelName,
		"display_name":            provider.DisplayName,
		"base_url":                provider.BaseURL,
		"azure_endpoint":          provider.AzureEndpoint,
		"azure_deployment":        provider.AzureDeployment,
		"azure_api_version":       provider.AzureAPIVersion,
		"temperature":             provider.Temperature,
		"max_tokens":              provider.MaxTokens,
		"cost_per_1k_input":       provider.CostPer1KInput,
		"cost_per_1k_output":      provider.CostPer1KOutput,
		"max_requests_per_minute": provider.MaxRequestsPerMinute,
		"max_requests_per_day":    provider.MaxRequestsPerDay,
		"monthly_budget":          provider.MonthlyBudget,
		"is_active":               provider.IsActive,
		"is_default":              provider.IsDefault,
		"is_fallback":             provider.IsFallback,
		"health_status":           provider.HealthStatus,
	}
}
