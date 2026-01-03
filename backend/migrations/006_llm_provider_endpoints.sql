ALTER TABLE llm_providers
  ADD COLUMN IF NOT EXISTS base_url TEXT,
  ADD COLUMN IF NOT EXISTS azure_endpoint TEXT,
  ADD COLUMN IF NOT EXISTS azure_deployment TEXT,
  ADD COLUMN IF NOT EXISTS azure_api_version TEXT;

CREATE INDEX IF NOT EXISTS llm_providers_base_url_idx ON llm_providers (tenant_id, base_url);
