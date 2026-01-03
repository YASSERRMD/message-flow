import { useMemo, useState } from "react";

export default function FeatureAssignmentPanel({ features, providers, onAssign, onRemove }) {
  const [selectedFeature, setSelectedFeature] = useState("");
  const [providerId, setProviderId] = useState("");
  const [priority, setPriority] = useState(1);

  const selectedProviders = useMemo(() => {
    if (!selectedFeature) return [];
    const feature = features.find((item) => item.feature === selectedFeature);
    return feature?.providers || [];
  }, [features, selectedFeature]);

  const providerMap = useMemo(() => {
    return providers.reduce((acc, provider) => {
      acc[provider.id] = provider;
      return acc;
    }, {});
  }, [providers]);

  const handleAssign = () => {
    if (!selectedFeature || !providerId) return;
    onAssign(selectedFeature, { provider_id: Number(providerId), priority: Number(priority) });
  };

  return (
    <section className="panel llm-panel">
      <header className="panel-header">
        <div>
          <h3>Feature Assignment</h3>
          <p className="panel-sub">Primary and fallback ordering per feature</p>
        </div>
      </header>
      <div className="feature-assign">
        <label>
          Feature
          <select value={selectedFeature} onChange={(e) => setSelectedFeature(e.target.value)}>
            <option value="">Select feature</option>
            {features.map((feature) => (
              <option key={feature.feature} value={feature.feature}>{feature.feature}</option>
            ))}
          </select>
        </label>
        <label>
          Provider
          <select value={providerId} onChange={(e) => setProviderId(e.target.value)}>
            <option value="">Select provider</option>
            {providers.map((provider) => (
              <option key={provider.id} value={provider.id}>{provider.provider_name} ({provider.model_name})</option>
            ))}
          </select>
        </label>
        <label>
          Priority
          <input type="number" min="1" value={priority} onChange={(e) => setPriority(e.target.value)} />
        </label>
        <button className="primary" onClick={handleAssign}>Assign</button>
      </div>
      <div className="feature-list">
        {selectedProviders.map((item) => {
          const provider = providerMap[item.provider_id];
          return (
            <div key={`${item.provider_id}-${item.priority}`} className="feature-row">
              <span>{provider ? provider.provider_name : `Provider ${item.provider_id}`}</span>
              <span>Priority {item.priority}</span>
              <button className="ghost" onClick={() => onRemove(selectedFeature, item.provider_id)}>Remove</button>
            </div>
          );
        })}
      </div>
    </section>
  );
}
