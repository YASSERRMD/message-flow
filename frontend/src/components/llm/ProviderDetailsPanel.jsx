import { useEffect, useState } from "react";

export default function ProviderDetailsPanel({ provider, onClose, onUpdate, headers }) {
  const [form, setForm] = useState(null);
  const [history, setHistory] = useState([]);

  useEffect(() => {
    if (provider) {
      setForm({
        display_name: provider.display_name || "",
        model_name: provider.model_name,
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
    fetch(`${import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1"}/llm/providers/${provider.id}/history`, { headers })
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
    onUpdate(provider.id, {
      ...form,
      monthly_budget: form.monthly_budget === "" ? null : Number(form.monthly_budget)
    });
  };

  return (
    <aside className="panel llm-panel llm-details">
      <header className="panel-header">
        <div>
          <h3>{provider.provider_name} settings</h3>
          <p className="panel-sub">{provider.model_name}</p>
        </div>
        <button className="ghost" onClick={onClose}>Close</button>
      </header>
      <div className="panel-body form-grid">
        <label>
          Display name
          <input value={form.display_name} onChange={(e) => updateField("display_name", e.target.value)} />
        </label>
        <label>
          Model
          <input value={form.model_name} onChange={(e) => updateField("model_name", e.target.value)} />
        </label>
        <label>
          Base URL
          <input value={form.base_url} onChange={(e) => updateField("base_url", e.target.value)} />
        </label>
        <label>
          Azure endpoint
          <input value={form.azure_endpoint} onChange={(e) => updateField("azure_endpoint", e.target.value)} />
        </label>
        <label>
          Azure deployment
          <input value={form.azure_deployment} onChange={(e) => updateField("azure_deployment", e.target.value)} />
        </label>
        <label>
          Azure API version
          <input value={form.azure_api_version} onChange={(e) => updateField("azure_api_version", e.target.value)} />
        </label>
        <label>
          Temperature
          <input
            type="number"
            min="0"
            max="2"
            step="0.1"
            value={form.temperature}
            onChange={(e) => updateField("temperature", Number(e.target.value))}
          />
        </label>
        <label>
          Max tokens
          <input
            type="number"
            min="100"
            max="4000"
            value={form.max_tokens}
            onChange={(e) => updateField("max_tokens", Number(e.target.value))}
          />
        </label>
        <label>
          Cost per 1k input
          <input
            type="number"
            min="0"
            step="0.001"
            value={form.cost_per_1k_input}
            onChange={(e) => updateField("cost_per_1k_input", Number(e.target.value))}
          />
        </label>
        <label>
          Cost per 1k output
          <input
            type="number"
            min="0"
            step="0.001"
            value={form.cost_per_1k_output}
            onChange={(e) => updateField("cost_per_1k_output", Number(e.target.value))}
          />
        </label>
        <label>
          Max requests per minute
          <input
            type="number"
            value={form.max_requests_per_minute}
            onChange={(e) => updateField("max_requests_per_minute", Number(e.target.value))}
          />
        </label>
        <label>
          Max requests per day
          <input
            type="number"
            value={form.max_requests_per_day}
            onChange={(e) => updateField("max_requests_per_day", Number(e.target.value))}
          />
        </label>
        <label>
          Monthly budget
          <input
            type="number"
            value={form.monthly_budget}
            onChange={(e) => updateField("monthly_budget", e.target.value)}
          />
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={form.is_active} onChange={(e) => updateField("is_active", e.target.checked)} />
          Active
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={form.is_default} onChange={(e) => updateField("is_default", e.target.checked)} />
          Default provider
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={form.is_fallback} onChange={(e) => updateField("is_fallback", e.target.checked)} />
          Fallback provider
        </label>
      </div>
      <div className="history-block">
        <h4>Version history</h4>
        <ul>
          {history.map((item) => {
            let payload = {};
            try {
              payload = JSON.parse(item.change_json);
            } catch (error) {
              payload = { event: "update" };
            }
            return (
              <li key={item.id}>
                <span>{new Date(item.created_at).toLocaleString()}</span>
                <span className="muted">{payload.event || "update"}</span>
              </li>
            );
          })}
          {history.length === 0 && <li className="muted">No recent changes.</li>}
        </ul>
      </div>
      <button className="primary" onClick={handleSave}>Save changes</button>
    </aside>
  );
}
