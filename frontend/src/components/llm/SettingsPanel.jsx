import { useState } from "react";

export default function SettingsPanel({ providers, onUpdate }) {
  const [defaults, setDefaults] = useState({
    analysis: "",
    summarization: "",
    extraction: "",
    fallback: true,
    budgetAlert: 80,
    healthInterval: 5
  });

  const handleSave = () => {
    if (defaults.analysis) {
      onUpdate(Number(defaults.analysis), { is_default: true });
    }
  };

  return (
    <section className="panel llm-panel">
      <header className="panel-header">
        <div>
          <h3>Global Settings</h3>
          <p className="panel-sub">Defaults and automation preferences</p>
        </div>
      </header>
      <div className="settings-grid">
        <label>
          Default provider for analysis
          <select value={defaults.analysis} onChange={(e) => setDefaults({ ...defaults, analysis: e.target.value })}>
            <option value="">Select provider</option>
            {providers.map((provider) => (
              <option key={provider.id} value={provider.id}>{provider.provider_name}</option>
            ))}
          </select>
        </label>
        <label>
          Default provider for summarization
          <select value={defaults.summarization} onChange={(e) => setDefaults({ ...defaults, summarization: e.target.value })}>
            <option value="">Select provider</option>
            {providers.map((provider) => (
              <option key={provider.id} value={provider.id}>{provider.provider_name}</option>
            ))}
          </select>
        </label>
        <label>
          Default provider for action extraction
          <select value={defaults.extraction} onChange={(e) => setDefaults({ ...defaults, extraction: e.target.value })}>
            <option value="">Select provider</option>
            {providers.map((provider) => (
              <option key={provider.id} value={provider.id}>{provider.provider_name}</option>
            ))}
          </select>
        </label>
        <label>
          Budget alert threshold (%)
          <input type="number" min="0" max="100" value={defaults.budgetAlert} onChange={(e) => setDefaults({ ...defaults, budgetAlert: e.target.value })} />
        </label>
        <label>
          Health check interval (minutes)
          <input type="number" min="1" value={defaults.healthInterval} onChange={(e) => setDefaults({ ...defaults, healthInterval: e.target.value })} />
        </label>
        <label className="checkbox-row">
          <input type="checkbox" checked={defaults.fallback} onChange={(e) => setDefaults({ ...defaults, fallback: e.target.checked })} />
          Automatic fallback strategy
        </label>
      </div>
      <button className="primary" onClick={handleSave}>Save settings</button>
    </section>
  );
}
