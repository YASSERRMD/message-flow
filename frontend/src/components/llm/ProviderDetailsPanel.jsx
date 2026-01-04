import { useEffect, useState } from "react";

export default function ProviderDetailsPanel({ provider, onClose, onUpdate, headers }) {
  const [form, setForm] = useState(null);
  const [history, setHistory] = useState([]);
  const [activeTab, setActiveTab] = useState("config");

  useEffect(() => {
    if (provider) {
      setForm({
        display_name: provider.display_name || "",
        model_name: provider.model_name,
        api_key: "", // Empty by default, only filled if user wants to update
        base_url: provider.base_url || "",
        azure_endpoint: provider.azure_endpoint || "",
        azure_deployment: provider.azure_deployment || "",
        azure_api_version: provider.azure_api_version || "",
        temperature: provider.temperature,
        max_tokens: provider.max_tokens,
        cost_per_1k_input: provider.cost_per_1k_input,
        cost_per_1k_output: provider.cost_per_1k_output,
        max_requests_per_minute: provider.max_requests_per_minute,
        max_requests_per_day: provider.max_requests_per_day,
        monthly_budget: provider.monthly_budget || "",
        is_active: provider.is_active,
        is_default: provider.is_default,
        is_fallback: provider.is_fallback
      });
    }
  }, [provider]);

  useEffect(() => {
    if (!provider || !headers) return;
    fetch(`${import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1"}/llm/providers/${provider.id}/history`, { headers })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => setHistory(data?.data || []))
      .catch(() => setHistory([]));
  }, [provider, headers]);

  if (!provider || !form) {
    return (
      <aside className="panel llm-panel llm-details empty-panel">
        <h3>Provider Details</h3>
        <p className="empty">Select a provider to edit configuration.</p>
      </aside>
    );
  }

  const updateField = (key, value) => setForm((prev) => ({ ...prev, [key]: value }));

  const handleSave = () => {
    const updates = {
      ...form,
      monthly_budget: form.monthly_budget === "" ? null : Number(form.monthly_budget)
    };
    // Only include api_key if user entered a new one
    if (!form.api_key) {
      delete updates.api_key;
    }
    onUpdate(provider.id, updates);
  };

  return (
    <aside className="panel llm-panel llm-details">
      <header className="panel-header" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: '12px' }}>
        <div style={{ width: '100%', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <h3>{provider.provider_name}</h3>
            <p className="panel-sub">{provider.model_name}</p>
          </div>
          <button className="action-btn ghost" onClick={onClose}><i className="fas fa-times"></i></button>
        </div>
        <div className="filters-bar" style={{ margin: 0, borderBottom: '1px solid #e5e7eb', width: '100%' }}>
          <button className={`filter-dropdown ${activeTab === 'config' ? 'active' : ''}`} style={activeTab === 'config' ? { background: '#f3f4f6', fontWeight: 600 } : {}} onClick={() => setActiveTab('config')}>Configuration</button>
          <button className={`filter-dropdown ${activeTab === 'history' ? 'active' : ''}`} style={activeTab === 'history' ? { background: '#f3f4f6', fontWeight: 600 } : {}} onClick={() => setActiveTab('history')}>History</button>
        </div>
      </header>

      <div className="panel-body" style={{ overflowY: 'auto', maxHeight: 'calc(100vh - 200px)' }}>
        {activeTab === 'config' && (
          <div className="form-grid" style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div className="form-group">
              <label className="form-label">Display Name</label>
              <input className="form-control" value={form.display_name} onChange={(e) => updateField("display_name", e.target.value)} />
            </div>

            <div className="form-group">
              <label className="form-label">Model</label>
              <input className="form-control" value={form.model_name} onChange={(e) => updateField("model_name", e.target.value)} />
            </div>

            <div className="form-group">
              <label className="form-label">API Key (leave blank to keep current)</label>
              <input
                type="password"
                className="form-control"
                placeholder="Enter new API key to update"
                value={form.api_key || ""}
                onChange={(e) => updateField("api_key", e.target.value)}
              />
              <p style={{ fontSize: '12px', color: '#6b7280', marginTop: '4px' }}>Only fill this if you want to update the key</p>
            </div>

            <div className="form-group">
              <label className="form-label">Base URL (Optional)</label>
              <input className="form-control" placeholder="https://api.openai.com/v1" value={form.base_url} onChange={(e) => updateField("base_url", e.target.value)} />
            </div>

            {(provider.provider_name?.toLowerCase() === 'azure_openai' || provider.provider_name?.toLowerCase() === 'azure openai') && (
              <>
                <div className="form-group">
                  <label className="form-label">Azure Endpoint</label>
                  <input className="form-control" placeholder="https://your-resource.openai.azure.com" value={form.azure_endpoint} onChange={(e) => updateField("azure_endpoint", e.target.value)} />
                </div>
                <div className="form-group">
                  <label className="form-label">Azure Deployment Name</label>
                  <input className="form-control" placeholder="gpt-4-deployment" value={form.azure_deployment} onChange={(e) => updateField("azure_deployment", e.target.value)} />
                </div>
                <div className="form-group">
                  <label className="form-label">Azure API Version</label>
                  <input className="form-control" placeholder="2024-02-15-preview" value={form.azure_api_version} onChange={(e) => updateField("azure_api_version", e.target.value)} />
                </div>
              </>
            )}

            <div className="two-column" style={{ gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
              <div className="form-group">
                <label className="form-label">Temperature</label>
                <input className="form-control" type="number" step="0.1" value={form.temperature} onChange={(e) => updateField("temperature", Number(e.target.value))} />
              </div>
              <div className="form-group">
                <label className="form-label">Max Tokens</label>
                <input className="form-control" type="number" value={form.max_tokens} onChange={(e) => updateField("max_tokens", Number(e.target.value))} />
              </div>
            </div>

            <div className="form-group">
              <label className="checkbox-group">
                <input type="checkbox" checked={form.is_active} onChange={(e) => updateField("is_active", e.target.checked)} />
                <label>Active</label>
              </label>
            </div>

            <div className="form-group">
              <label className="checkbox-group">
                <input type="checkbox" checked={form.is_default} onChange={(e) => updateField("is_default", e.target.checked)} />
                <label>Set as Default Provider</label>
              </label>
              <p style={{ fontSize: '12px', color: '#6b7280', marginTop: '4px' }}>Default provider is used for summarization and AI features</p>
            </div>

            <button className="action-btn action-btn-primary" style={{ width: '100%', justifyContent: 'center' }} onClick={handleSave}>Save Changes</button>
          </div>
        )}

        {activeTab === 'history' && (
          <ul className="info-list">
            {history.length === 0 && <p className="empty-text">No history available.</p>}
            {history.map((h, i) => (
              <li key={i} className="info-item" style={{ flexDirection: 'column', alignItems: 'flex-start' }}>
                <div style={{ fontSize: '12px', color: '#9ca3af' }}>{new Date(h.timestamp).toLocaleString()}</div>
                <div style={{ fontSize: '14px' }}>{h.action_summary || "Updated configuration"}</div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </aside>
  );
}
