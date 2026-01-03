ALTER TABLE llm_providers
  ADD COLUMN IF NOT EXISTS display_name TEXT,
  ADD COLUMN IF NOT EXISTS max_requests_per_day INTEGER NOT NULL DEFAULT 10000,
  ADD COLUMN IF NOT EXISTS monthly_budget DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS is_fallback BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS llm_providers_fallback_idx ON llm_providers (tenant_id, is_fallback);
