const formatCurrency = (value) => `$${value.toFixed(2)}`;

export default function CostAnalyticsDashboard({ costs, usageByFeature }) {
  const total = costs.by_provider?.reduce((sum, item) => sum + item.total_cost, 0) || 0;
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
            {costs.by_provider?.map((item) => (
              <div key={item.provider} className="mini-row">
                <span>{item.provider}</span>
                <div className="mini-bar">
                  <div className="mini-bar-fill" style={{ width: `${Math.min(100, (item.total_cost / (total || 1)) * 100)}%` }} />
                </div>
                <span>{formatCurrency(item.total_cost)}</span>
              </div>
            ))}
          </div>
        </div>
        <div>
          <h4>Cost by feature</h4>
          <div className="mini-chart">
            {usageByFeature.map((item) => (
              <div key={item.feature} className="mini-row">
                <span>{item.feature}</span>
                <div className="mini-bar">
                  <div className="mini-bar-fill" style={{ width: `${Math.min(100, (item.total_cost / (total || 1)) * 100)}%` }} />
                </div>
                <span>{formatCurrency(item.total_cost)}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className="bar-chart">
        {costs.by_day?.slice().reverse().map((item) => (
          <div key={item.day} className="bar" style={{ height: `${Math.min(100, item.total_cost * 6)}%` }} />
        ))}
      </div>
    </section>
  );
}
