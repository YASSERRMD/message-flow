const formatDate = (value) => (value ? new Date(value).toLocaleString() : "--");

export default function HealthStatusDashboard({ health, comparison }) {
  const overallScore = health.length
    ? Math.round((health.filter((item) => item.status === "ok").length / health.length) * 100)
    : 0;

  const latestCheck = health.reduce((latest, item) => {
    const time = item.last_check ? new Date(item.last_check).getTime() : 0;
    return time > latest ? time : latest;
  }, 0);

  return (
    <section className="panel llm-panel">
      <header className="panel-header">
        <div>
          <h3>Health Monitoring</h3>
          <p className="panel-sub">{overallScore}% healthy</p>
        </div>
        <span className="panel-chip">Last update {latestCheck ? formatDate(latestCheck) : "--"}</span>
      </header>
      <div className="health-grid">
        {health.map((item) => (
          <div key={item.provider_id} className="health-row">
            <span className={`status-dot status-dot--${item.status}`} />
            <div>
              <strong>{item.provider}</strong>
              <p>{item.status === "ok" ? "Healthy" : item.status === "slow" ? "Slow" : "Unhealthy"}</p>
            </div>
            <span>{Math.round(item.avg_latency_ms || 0)}ms</span>
            <span>{formatDate(item.last_check)}</span>
          </div>
        ))}
      </div>
      <div className="line-chart">
        {comparison.map((item) => (
          <div key={item.provider_id} className="line-item">
            <span>{item.provider}</span>
            <div className="line">
              <div className="line-fill" style={{ width: `${Math.min(100, item.avg_latency_ms / 50)}%` }} />
            </div>
            <span>{Math.round(item.avg_latency_ms)}ms</span>
          </div>
        ))}
      </div>
    </section>
  );
}
