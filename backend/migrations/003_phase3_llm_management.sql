CREATE TABLE IF NOT EXISTS llm_provider_history (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  provider_id BIGINT NOT NULL REFERENCES llm_providers(id) ON DELETE CASCADE,
  change_json JSONB NOT NULL,
  changed_by BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS llm_provider_history_provider_idx ON llm_provider_history (provider_id, created_at DESC);

CREATE TABLE IF NOT EXISTS llm_feature_assignments (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  feature_name TEXT NOT NULL,
  provider_id BIGINT NOT NULL REFERENCES llm_providers(id) ON DELETE CASCADE,
  priority INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS llm_feature_assignments_unique ON llm_feature_assignments (tenant_id, feature_name, provider_id);
CREATE INDEX IF NOT EXISTS llm_feature_assignments_feature_idx ON llm_feature_assignments (tenant_id, feature_name, priority);

ALTER TABLE llm_provider_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE llm_feature_assignments ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_llm_provider_history ON llm_provider_history
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_llm_feature_assignments ON llm_feature_assignments
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);
