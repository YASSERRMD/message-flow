import { useMemo, useState } from "react";

export default function FeatureAssignmentPanel({ features = [], providers = [], onAssign, onRemove }) {
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
      <div className="panel-body">
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 100px auto', gap: '16px', alignItems: 'end', marginBottom: '24px' }}>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">Feature</label>
            <select
              className="form-control"
              value={selectedFeature}
              onChange={(e) => setSelectedFeature(e.target.value)}
            >
              <option value="">Select feature</option>
              {features?.map((feature) => (
                <option key={feature.feature} value={feature.feature}>{feature.feature}</option>
              ))}
            </select>
          </div>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">Provider</label>
            <select
              className="form-control"
              value={providerId}
              onChange={(e) => setProviderId(e.target.value)}
            >
              <option value="">Select provider</option>
              {providers?.map((provider) => (
                <option key={provider.id} value={provider.id}>{provider.provider_name} ({provider.model_name})</option>
              ))}
            </select>
          </div>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">Priority</label>
            <input
              type="number"
              className="form-control"
              min="1"
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
            />
          </div>
          <button className="action-btn action-btn-primary" onClick={handleAssign} style={{ height: '42px' }}>Assign</button>
        </div>

        <h4 style={{ margin: '24px 0 12px', fontSize: '14px', fontWeight: 600 }}>Current Assignments</h4>
        <div className="feature-list">
          {features.length === 0 && <p className="empty-text">No features configured.</p>}
          {features.map((feature) => {
            const assignedProviders = feature.providers || [];
            const primaryProvider = assignedProviders.length > 0 ? providerMap[assignedProviders[0]?.provider_id] : null;
            return (
              <div key={feature.feature} className="info-item" style={{ flexDirection: 'column', alignItems: 'flex-start', gap: '8px', padding: '12px', background: 'rgba(0,0,0,0.02)', borderRadius: '8px', marginBottom: '8px' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                  <span style={{ fontWeight: 600, textTransform: 'capitalize' }}>{feature.feature.replace(/_/g, ' ')}</span>
                  {primaryProvider ? (
                    <span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '4px', background: '#10b981', color: 'white' }}>
                      {primaryProvider.provider_name} ({primaryProvider.model_name})
                    </span>
                  ) : (
                    <span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '4px', background: '#ef4444', color: 'white' }}>Not Assigned</span>
                  )}
                </div>
                {assignedProviders.length > 1 && (
                  <div style={{ fontSize: '12px', color: '#6b7280' }}>
                    Fallbacks: {assignedProviders.slice(1).map((item) => providerMap[item.provider_id]?.provider_name || `ID ${item.provider_id}`).join(', ')}
                  </div>
                )}
              </div>
            );
          })}
        </div>

        {selectedFeature && selectedProviders.length > 0 && (
          <>
            <h4 style={{ margin: '24px 0 12px', fontSize: '14px', fontWeight: 600 }}>Providers for "{selectedFeature}"</h4>
            <div className="feature-list">
              {selectedProviders.map((item) => {
                const provider = providerMap[item.provider_id];
                return (
                  <div key={`${item.provider_id}-${item.priority}`} className="info-item" style={{ justifyContent: 'space-between' }}>
                    <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                      <span style={{ fontWeight: 600 }}>{provider ? provider.provider_name : `Provider ${item.provider_id}`}</span>
                      <span style={{ fontSize: '12px', color: '#6b7280' }}>Priority {item.priority}</span>
                    </div>
                    <button className="action-btn ghost" onClick={() => onRemove(selectedFeature, item.provider_id)} style={{ padding: '4px 8px', fontSize: '12px' }}>Remove</button>
                  </div>
                );
              })}
            </div>
          </>
        )}
      </div>
    </section>
  );
}
