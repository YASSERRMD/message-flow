export default function PerformanceComparisonTable({ comparison = [], recommendations = [] }) {
  const safeComparison = Array.isArray(comparison) ? comparison : [];
  const safeRecommendations = Array.isArray(recommendations) ? recommendations : [];
  return (
    <section className="panel llm-panel">
      <header className="panel-header">
        <div>
          <h3>Performance Comparison</h3>
          <p className="panel-sub">Latency, success rate, and spend overview</p>
        </div>
      </header>
      <table className="comparison-table">
        <thead>
          <tr>
            <th>Provider</th>
            <th>Latency</th>
            <th>Success rate</th>
            <th>Cost / 1k</th>
            <th>Monthly spent</th>
            <th>Requests</th>
          </tr>
        </thead>
        <tbody>
          {safeComparison.length === 0 && (
            <tr><td colSpan="6" style={{ textAlign: 'center', padding: '20px', color: '#888' }}>No performance data available</td></tr>
          )}
          {safeComparison.map((row) => (
            <tr key={row.provider_id}>
              <td>{row.provider}</td>
              <td>{Math.round(row.avg_latency_ms)}ms</td>
              <td>{row.success_rate.toFixed(1)}%</td>
              <td>{row.cost_per_1k ? `$${row.cost_per_1k}` : "--"}</td>
              <td>${(row.monthly_spent || 0).toFixed(2)}</td>
              <td>{row.requests}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="recommendations">
        <h4>Recommendations</h4>
        <ul>
          {safeRecommendations.length === 0 && <li>No recommendations yet.</li>}
          {safeRecommendations.map((rec, index) => (
            <li key={index}>{rec}</li>
          ))}
        </ul>
      </div>
    </section>
  );
}
