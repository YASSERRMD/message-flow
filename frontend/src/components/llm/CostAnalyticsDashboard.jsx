const formatCurrency = (value) => `$${value.toFixed(2)}`;

export default function CostAnalyticsDashboard({ costs, usageByFeature }) {
  const safeCosts = costs || {};
  const safeUsage = Array.isArray(usageByFeature) ? usageByFeature : [];
  const total = safeCosts.by_provider?.reduce((sum, item) => sum + (item.total_cost || 0), 0) || 0;
  const remaining = null;

  return (
    <section className="panel llm-panel">
      <header className="panel-header">
        <div>
          <h3>Cost Analytics</h3>
          <p className="panel-sub">{formatCurrency(total)} spent this month</p>
        </div>
        <span className="panel-chip">Remaining {remaining === null ? "--" : formatCurrency(remaining)}</span>
      </header>
      <div className="cost-grid">
        <div>
          <h4>Cost by provider</h4>
          <div className="mini-chart">
            {safeCosts.by_provider?.map((item) => (
              <div key={item.provider} className="mini-row">
                <span>{item.provider}</span>
                <div className="mini-bar">
                  <div className="mini-bar-fill" style={{ width: `${Math.min(100, ((item.total_cost || 0) / (total || 1)) * 100)}%` }} />
                </div>
                <span>{formatCurrency(item.total_cost || 0)}</span>
              </div>
            ))}
            {(!safeCosts.by_provider || safeCosts.by_provider.length === 0) && <p className="empty-text">No cost data available</p>}
          </div>
        </div>
        <div>
          <h4>Cost by feature</h4>
          <div className="mini-chart">
            {safeUsage.map((item) => (
              <div key={item.feature} className="mini-row">
                <span>{item.feature}</span>
                <div className="mini-bar">
                  <div className="mini-bar-fill" style={{ width: `${Math.min(100, ((item.total_cost || 0) / (total || 1)) * 100)}%` }} />
                </div>
                <span>{formatCurrency(item.total_cost || 0)}</span>
              </div>
            ))}
            {safeUsage.length === 0 && <p className="empty-text">No usage data available</p>}
          </div>
        </div>
      </div>
      <div className="bar-chart">
        {safeCosts.by_day?.slice().reverse().map((item) => (
          <div key={item.day} className="bar" style={{ height: `${Math.min(100, (item.total_cost || 0) * 6)}%` }} />
        ))}
      </div>
    </section>
  );
}
