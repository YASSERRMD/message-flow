INSERT INTO llm_providers (
    tenant_id, 
    provider_name, 
    model_name, 
    api_key, 
    is_active, 
    cost_per_1k_input, 
    cost_per_1k_output, 
    health_status, 
    created_at
) VALUES 
(1, 'OpenAI', 'gpt-4o', 'sk-placeholder-openai', true, 0.005, 0.015, 'healthy', NOW()),
(1, 'Anthropic', 'claude-3-5-sonnet', 'sk-placeholder-anthropic', true, 0.003, 0.015, 'healthy', NOW()),
(1, 'Google', 'gemini-1.5-pro', 'sk-placeholder-google', true, 0.001, 0.002, 'healthy', NOW()),
(1, 'Cohere', 'command-r-plus', 'sk-placeholder-cohere', false, 0.004, 0.008, 'degraded', NOW());

-- Seed some sample costs/analytics if table exists (optional, assuming schema)
-- INSERT INTO llm_usage_logs ... (skipping for now to avoid schema complex conflicts, providers should be enough to show list)
