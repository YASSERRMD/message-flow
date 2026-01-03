CREATE TABLE IF NOT EXISTS llm_providers (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  provider_name TEXT NOT NULL,
  api_key TEXT NOT NULL,
  model_name TEXT NOT NULL,
  temperature DOUBLE PRECISION NOT NULL DEFAULT 0.2,
  max_tokens INTEGER NOT NULL DEFAULT 1024,
  cost_per_1k_input DOUBLE PRECISION NOT NULL DEFAULT 0,
  cost_per_1k_output DOUBLE PRECISION NOT NULL DEFAULT 0,
  max_requests_per_minute INTEGER NOT NULL DEFAULT 60,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  health_status TEXT NOT NULL DEFAULT 'unknown',
  last_health_check TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS llm_providers_tenant_idx ON llm_providers (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS llm_providers_tenant_default_idx ON llm_providers (tenant_id) WHERE is_default;

CREATE TABLE IF NOT EXISTS llm_usage_logs (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  provider_id BIGINT NOT NULL REFERENCES llm_providers(id) ON DELETE CASCADE,
  message_id BIGINT,
  input_tokens INTEGER NOT NULL,
  output_tokens INTEGER NOT NULL,
  total_tokens INTEGER NOT NULL,
  input_cost DOUBLE PRECISION NOT NULL,
  output_cost DOUBLE PRECISION NOT NULL,
  total_cost DOUBLE PRECISION NOT NULL,
  response_time_ms BIGINT NOT NULL,
  success BOOLEAN NOT NULL,
  error_message TEXT,
  feature_used TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS llm_usage_logs_tenant_created_idx ON llm_usage_logs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS llm_usage_logs_provider_idx ON llm_usage_logs (provider_id, created_at DESC);

CREATE TABLE IF NOT EXISTS llm_provider_health (
  id BIGSERIAL PRIMARY KEY,
  provider_id BIGINT NOT NULL REFERENCES llm_providers(id) ON DELETE CASCADE,
  tenant_id BIGINT NOT NULL,
  check_time TIMESTAMPTZ NOT NULL,
  status TEXT NOT NULL,
  latency_ms BIGINT NOT NULL,
  error_message TEXT,
  http_status_code INTEGER,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS llm_provider_health_tenant_idx ON llm_provider_health (tenant_id, check_time DESC);

CREATE UNIQUE INDEX IF NOT EXISTS important_messages_message_id_idx ON important_messages (message_id);

ALTER TABLE llm_providers ENABLE ROW LEVEL SECURITY;
ALTER TABLE llm_usage_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE llm_provider_health ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_llm_providers ON llm_providers
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_llm_usage_logs ON llm_usage_logs
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_llm_provider_health ON llm_provider_health
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);
