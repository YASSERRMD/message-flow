import { useState } from "react";

export default function SettingsPanel({ providers = [], onUpdate }) {
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
      <div className="panel-body">
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '20px', marginBottom: '24px' }}>
          <div className="form-group">
            <label className="form-label">Default provider for analysis</label>
            <select className="form-control" value={defaults.analysis} onChange={(e) => setDefaults({ ...defaults, analysis: e.target.value })}>
              <option value="">Select provider</option>
              {providers?.map((provider) => (
                <option key={provider.id} value={provider.id}>{provider.provider_name}</option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label className="form-label">Default provider for summarization</label>
            <select className="form-control" value={defaults.summarization} onChange={(e) => setDefaults({ ...defaults, summarization: e.target.value })}>
              <option value="">Select provider</option>
              {providers?.map((provider) => (
                <option key={provider.id} value={provider.id}>{provider.provider_name}</option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label className="form-label">Default provider for action extraction</label>
            <select className="form-control" value={defaults.extraction} onChange={(e) => setDefaults({ ...defaults, extraction: e.target.value })}>
              <option value="">Select provider</option>
              {providers?.map((provider) => (
                <option key={provider.id} value={provider.id}>{provider.provider_name}</option>
              ))}
            </select>
          </div>
          <div className="form-group">
            <label className="form-label">Budget alert threshold (%)</label>
            <input className="form-control" type="number" min="0" max="100" value={defaults.budgetAlert} onChange={(e) => setDefaults({ ...defaults, budgetAlert: e.target.value })} />
          </div>
          <div className="form-group">
            <label className="form-label">Health check interval (minutes)</label>
            <input className="form-control" type="number" min="1" value={defaults.healthInterval} onChange={(e) => setDefaults({ ...defaults, healthInterval: e.target.value })} />
          </div>
          <div className="form-group" style={{ display: 'flex', alignItems: 'center', paddingTop: '24px' }}>
            <div className="checkbox-group">
              <input type="checkbox" checked={defaults.fallback} onChange={(e) => setDefaults({ ...defaults, fallback: e.target.checked })} />
              <label>Automatic fallback strategy</label>
            </div>
          </div>
        </div>
        <button className="action-btn action-btn-primary" onClick={handleSave}>Save settings</button>
      </div>
    </section>
  );
}
