const formatDate = (value) => {
  if (!value) return "--";
  return new Date(value).toLocaleString();
};

export default function ProviderCard({ provider, onSelect, onTest, onRemove }) {
  const status = provider.health_status || "unknown";

  // Calculate simplified budget visualization if needed, currently not storing historical spend in provider obj directly (relies on costs API).
  // We'll use a placeholder or basic visual for now.

  const handleKey = (event) => {
    if (event.key === "Enter" || event.key === " ") {
      onSelect();
    }
  };

  return (
    <article
      className="provider-card"
      onClick={onSelect}
      onKeyDown={handleKey}
      role="button"
      tabIndex={0}
      style={{
        background: 'white',
        border: '1px solid #e5e7eb',
        borderRadius: '12px',
        padding: '20px',
        cursor: 'pointer',
        transition: 'all 0.2s',
        display: 'flex',
        flexDirection: 'column',
        gap: '16px'
      }}
      onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-2px)'; e.currentTarget.style.boxShadow = '0 10px 15px -3px rgba(0, 0, 0, 0.1)'; }}
      onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = 'none'; }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <h4 style={{ fontSize: '16px', fontWeight: 700, margin: 0, color: '#1a1d1f' }}>{provider.display_name || provider.provider_name}</h4>
          <p style={{ fontSize: '13px', color: '#6b7280', margin: '4px 0 0' }}>{provider.model_name}</p>
        </div>
        <span className={`status-badge`} style={{
          background: status === 'healthy' ? '#ecfdf5' : (status === 'degraded' ? '#fffbeb' : '#fef2f2'),
          color: status === 'healthy' ? '#047857' : (status === 'degraded' ? '#b45309' : '#b91c1c'),
          padding: '4px 8px',
          borderRadius: '6px',
          fontSize: '11px',
          fontWeight: 600,
          textTransform: 'uppercase'
        }}>
          {status}
        </span>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px', background: '#f9fafb', padding: '12px', borderRadius: '8px' }}>
        <div>
          <span style={{ fontSize: '11px', color: '#6b7280', display: 'block', textTransform: 'uppercase', marginBottom: '4px' }}>Input Cost</span>
          <strong style={{ fontSize: '14px', color: '#1a1d1f' }}>${provider.cost_per_1k_input.toFixed(3)}</strong>
        </div>
        <div>
          <span style={{ fontSize: '11px', color: '#6b7280', display: 'block', textTransform: 'uppercase', marginBottom: '4px' }}>Output Cost</span>
          <strong style={{ fontSize: '14px', color: '#1a1d1f' }}>${provider.cost_per_1k_output.toFixed(3)}</strong>
        </div>
        <div>
          <span style={{ fontSize: '11px', color: '#6b7280', display: 'block', textTransform: 'uppercase', marginBottom: '4px' }}>Max Tokens</span>
          <strong style={{ fontSize: '14px', color: '#1a1d1f' }}>{provider.max_tokens}</strong>
        </div>
        <div>
          <span style={{ fontSize: '11px', color: '#6b7280', display: 'block', textTransform: 'uppercase', marginBottom: '4px' }}>Temp</span>
          <strong style={{ fontSize: '14px', color: '#1a1d1f' }}>{provider.temperature}</strong>
        </div>
      </div>

      <div className="provider-actions" onClick={(event) => event.stopPropagation()} style={{ display: 'flex', gap: '8px', paddingTop: '4px' }}>
        <button type="button" className="action-btn" style={{ flex: 1, justifyContent: 'center', padding: '8px' }} onClick={onSelect}>Configure</button>
        <button type="button" className="action-btn" style={{ padding: '8px 12px', color: '#b91c1c', borderColor: '#fee2e2', background: '#fef2f2' }} onClick={onRemove}>
          <i className="fas fa-trash"></i>
        </button>
      </div>
    </article>
  );
}
