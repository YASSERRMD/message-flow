const formatDate = (value) => {
  if (!value) return "--";
  return new Date(value).toLocaleString();
};

export default function ProviderCard({ provider, onSelect, onTest, onRemove }) {
  const spent = provider.monthly_spent || 0;
  const budget = provider.monthly_budget || 0;
  const percent = budget > 0 ? Math.min(100, (spent / budget) * 100) : 0;
  const status = provider.health_status || "unknown";

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
    >
      <div className="provider-card__header">
        <div>
          <h4>{provider.display_name || provider.provider_name}</h4>
          <p>{provider.model_name}</p>
        </div>
        <span className={`status-dot status-dot--${status}`} />
      </div>
      <div className="provider-metrics">
        <div>
          <span>Temp</span>
          <strong>{provider.temperature.toFixed(2)}</strong>
        </div>
        <div>
          <span>Max tokens</span>
          <strong>{provider.max_tokens}</strong>
        </div>
      </div>
      <div className="provider-cost">
        <div>
          <span>Input</span>
          <strong>${provider.cost_per_1k_input.toFixed(3)}</strong>
        </div>
        <div>
          <span>Output</span>
          <strong>${provider.cost_per_1k_output.toFixed(3)}</strong>
        </div>
      </div>
      <div className="progress">
        <div className="progress-bar" style={{ width: `${percent}%` }} />
      </div>
      <p className="provider-meta">Last check: {formatDate(provider.last_health_check)}</p>
      <div className="provider-actions" onClick={(event) => event.stopPropagation()}>
        <button type="button" className="ghost" onClick={onTest}>Test</button>
        <button type="button" className="ghost" onClick={onSelect}>Configure</button>
        <button type="button" className="danger" onClick={onRemove}>Remove</button>
      </div>
    </article>
  );
}
