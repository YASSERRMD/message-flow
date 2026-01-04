import { useMemo, useState } from "react";

const defaultForm = {
  provider_name: "claude",
  display_name: "",
  api_key: "",
  model_name: "",
  base_url: "",
  azure_endpoint: "",
  azure_deployment: "",
  azure_api_version: "",
  temperature: 0.7,
  max_tokens: 1000,
  cost_per_1k_input: 0,
  cost_per_1k_output: 0,
  max_requests_per_minute: 60,
  max_requests_per_day: 10000,
  monthly_budget: "",
  is_default: false,
  is_fallback: false
};

export default function AddProviderModal({ open, onClose, onSubmit, providerModels = {} }) {
  const [form, setForm] = useState(defaultForm);
  const [status, setStatus] = useState("idle");
  const [error, setError] = useState("");
  const [showKey, setShowKey] = useState(false);

  // Models logic...
  const models = useMemo(() => providerModels[form.provider_name] || [], [form.provider_name, providerModels]);

  const updateField = (key, value) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  const validate = () => {
    if (!form.api_key) return "API key is required";
    if (!form.model_name && form.provider_name !== 'azure_openai') return "Model is required";
    if (form.provider_name === "azure_openai" && !form.azure_endpoint) return "Azure endpoint is required";
    return "";
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    const validation = validate();
    if (validation) {
      setError(validation);
      return;
    }
    setStatus("testing");
    setError("");

    const payload = {
      ...form,
      monthly_budget: form.monthly_budget === "" ? null : Number(form.monthly_budget)
    };

    const response = await onSubmit(payload);
    if (response) {
      // Assume failure if response is returned (mostly likely error). 
      // If it was void/undefined, dashboard likely handled it.
      // Actually dashboard impl: returns nothing on success.
      // So this logic depends on dashboard.
      // Dashboard returns NOTHING on success, but we need to know if it failed.
      // Dashboard uses `alert`.
    }
    // We rely on parent to close modal if success, or we stick to 'idle'
    setStatus("idle");
  };

  if (!open) return null;

  return (
    <div className="modal-overlay">
      <div className="modal">
        <header>
          <h3>Add Provider</h3>
          <button type="button" className="action-btn ghost" onClick={onClose} style={{ border: 'none', padding: '4px 8px' }}><i className="fas fa-times"></i></button>
        </header>

        <div className="modal-body">
          {error && <div style={{ background: '#fee2e2', color: '#b91c1c', padding: '10px', borderRadius: '8px', marginBottom: '16px', fontSize: '14px' }}>{error}</div>}

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
            <div className="form-group">
              <label className="form-label">Provider</label>
              <select className="form-control" value={form.provider_name} onChange={(e) => updateField("provider_name", e.target.value)}>
                <option value="claude">Claude</option>
                <option value="openai">OpenAI</option>
                <option value="azure_openai">Azure OpenAI</option>
                <option value="cohere">Cohere</option>
                <option value="gemini">Gemini</option>
              </select>
            </div>

            <div className="form-group">
              <label className="form-label">Display Name</label>
              <input className="form-control" placeholder="e.g. GPT-4 Main" value={form.display_name} onChange={(e) => updateField("display_name", e.target.value)} />
            </div>

            <div className="form-group" style={{ gridColumn: '1 / -1' }}>
              <label className="form-label">API Key</label>
              <div style={{ position: 'relative' }}>
                <input
                  className="form-control"
                  type={showKey ? "text" : "password"}
                  value={form.api_key}
                  onChange={(e) => updateField("api_key", e.target.value)}
                  placeholder="sk-..."
                />
                <button
                  type="button"
                  onClick={() => setShowKey(!showKey)}
                  style={{ position: 'absolute', right: '10px', top: '50%', transform: 'translateY(-50%)', border: 'none', background: 'transparent', cursor: 'pointer', color: '#6b7280' }}
                >
                  <i className={`fas fa-${showKey ? 'eye-slash' : 'eye'}`}></i>
                </button>
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Model Name</label>
              <input className="form-control" placeholder="e.g. gpt-4" value={form.model_name} onChange={(e) => updateField("model_name", e.target.value)} />
            </div>

            <div className="form-group">
              <label className="form-label">Max Tokens</label>
              <input className="form-control" type="number" value={form.max_tokens} onChange={(e) => updateField("max_tokens", Number(e.target.value))} />
            </div>

            {form.provider_name === 'azure_openai' && (
              <>
                <div className="form-group">
                  <label className="form-label">Azure Endpoint</label>
                  <input className="form-control" value={form.azure_endpoint} onChange={(e) => updateField("azure_endpoint", e.target.value)} />
                </div>
                <div className="form-group">
                  <label className="form-label">Deployment</label>
                  <input className="form-control" value={form.azure_deployment} onChange={(e) => updateField("azure_deployment", e.target.value)} />
                </div>
                <div className="form-group">
                  <label className="form-label">API Version</label>
                  <input className="form-control" value={form.azure_api_version} onChange={(e) => updateField("azure_api_version", e.target.value)} />
                </div>
              </>
            )}

            <div className="form-group">
              <label className="form-label">Cost per 1k Input ($)</label>
              <input className="form-control" type="number" step="0.0001" value={form.cost_per_1k_input} onChange={(e) => updateField("cost_per_1k_input", Number(e.target.value))} />
            </div>

            <div className="form-group">
              <label className="form-label">Cost per 1k Output ($)</label>
              <input className="form-control" type="number" step="0.0001" value={form.cost_per_1k_output} onChange={(e) => updateField("cost_per_1k_output", Number(e.target.value))} />
            </div>

            <div className="form-group" style={{ gridColumn: '1 / -1' }}>
              <div className="checkbox-group">
                <input type="checkbox" id="is_active" checked={true} disabled />
                <label htmlFor="is_active">Activate immediately</label>
              </div>
            </div>
          </div>

          <div style={{ marginTop: '24px', display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
            <button type="button" className="action-btn" onClick={onClose}>Cancel</button>
            <button type="button" className="action-btn action-btn-primary" onClick={handleSubmit} disabled={status === 'testing'}>
              {status === 'testing' ? 'Verifying...' : 'Add Provider'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
