import { useMemo, useState } from "react";

const defaultForm = {
  provider_name: "claude",
  display_name: "",
  api_key: "",
  model_name: "",
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

export default function AddProviderModal({ open, onClose, onSubmit, providerModels }) {
  const [form, setForm] = useState(defaultForm);
  const [status, setStatus] = useState("idle");
  const [error, setError] = useState("");
  const [showKey, setShowKey] = useState(false);

  const models = useMemo(() => providerModels[form.provider_name] || [], [form.provider_name, providerModels]);

  const updateField = (key, value) => setForm((prev) => ({ ...prev, [key]: value }));

  const validate = () => {
    if (!form.api_key) return "API key is required";
    if (!form.model_name) return "Model is required";
    if (form.temperature < 0 || form.temperature > 2) return "Temperature must be between 0 and 2";
    if (form.max_tokens <= 0) return "Max tokens must be greater than 0";
    if (form.cost_per_1k_input < 0 || form.cost_per_1k_output < 0) return "Cost values must be >= 0";
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
    if (response?.ok) {
      setStatus("success");
      setForm(defaultForm);
      onClose();
      return;
    }
    const data = response ? await response.json().catch(() => ({})) : {};
    setError(data.error || "Connection failed");
    setStatus("idle");
  };

  if (!open) return null;

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true">
      <div className="modal">
        <header>
          <h3>Add Provider</h3>
          <button type="button" className="ghost" onClick={onClose}>Close</button>
        </header>
        <form onSubmit={handleSubmit} className="modal-body">
          <label>
            Provider
            <select value={form.provider_name} onChange={(e) => updateField("provider_name", e.target.value)}>
              <option value="claude">Claude</option>
              <option value="openai">OpenAI</option>
              <option value="cohere">Cohere</option>
              <option value="gemini">Gemini</option>
              <option value="anthropic">Anthropic</option>
            </select>
          </label>
          <label>
            Display name
            <input value={form.display_name} onChange={(e) => updateField("display_name", e.target.value)} />
          </label>
          <label>
            API key
            <div className="inline-input">
              <input
                type={showKey ? "text" : "password"}
                value={form.api_key}
                onChange={(e) => updateField("api_key", e.target.value)}
              />
              <button type="button" className="ghost" onClick={() => setShowKey((prev) => !prev)}>
                {showKey ? "Hide" : "Show"}
              </button>
            </div>
          </label>
          <label>
            Model
            <select value={form.model_name} onChange={(e) => updateField("model_name", e.target.value)}>
              <option value="">Select model</option>
              {models.map((model) => (
                <option key={model} value={model}>{model}</option>
              ))}
            </select>
          </label>
          <label>
            Temperature
            <input
              type="range"
              min="0"
              max="2"
              step="0.1"
              value={form.temperature}
              onChange={(e) => updateField("temperature", Number(e.target.value))}
            />
            <span>{form.temperature.toFixed(1)}</span>
          </label>
          <label>
            Max tokens
            <input
              type="range"
              min="100"
              max="4000"
              step="50"
              value={form.max_tokens}
              onChange={(e) => updateField("max_tokens", Number(e.target.value))}
            />
            <span>{form.max_tokens}</span>
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
              min="1"
              value={form.max_requests_per_minute}
              onChange={(e) => updateField("max_requests_per_minute", Number(e.target.value))}
            />
          </label>
          <label>
            Max requests per day
            <input
              type="number"
              min="1"
              value={form.max_requests_per_day}
              onChange={(e) => updateField("max_requests_per_day", Number(e.target.value))}
            />
          </label>
          <label>
            Monthly budget (optional)
            <input
              type="number"
              min="0"
              value={form.monthly_budget}
              onChange={(e) => updateField("monthly_budget", e.target.value)}
            />
          </label>
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={form.is_default}
              onChange={(e) => updateField("is_default", e.target.checked)}
            />
            Set as default provider
          </label>
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={form.is_fallback}
              onChange={(e) => updateField("is_fallback", e.target.checked)}
            />
            Set as fallback provider
          </label>
          {error && <p className="error-text">{error}</p>}
          <button className="primary" type="submit" disabled={status === "testing"}>
            {status === "testing" ? "Testing…" : "Save provider"}
          </button>
        </form>
      </div>
    </div>
  );
}
