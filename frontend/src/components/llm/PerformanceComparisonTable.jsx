export default function PerformanceComparisonTable({ comparison = [], recommendations = [] }) {
  const safeComparison = Array.isArray(comparison) ? comparison : [];
  const safeRecommendations = Array.isArray(recommendations) ? recommendations : [];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <table className="data-table">
        <thead>
          <tr>
            <th>Provider</th>
            <th>Latency (avg)</th>
            <th>Success rate</th>
            <th>Cost / 1k</th>
            <th>Est. Monthly</th>
          </tr>
        </thead>
        <tbody>
          {safeComparison.length === 0 && (
            <tr><td colSpan="5" className="empty-text" style={{ textAlign: 'center', padding: '40px' }}>No performance data available</td></tr>
          )}
          {safeComparison.map((row, i) => (
            <tr key={i}>
              <td style={{ fontWeight: 600, color: '#1a1d1f' }}>{row.provider_name}</td>
              <td>{Math.round(row.avg_latency)}ms</td>
              <td>
                <span style={{
                  display: 'inline-block',
                  padding: '2px 8px',
                  borderRadius: '4px',
                  background: row.success_rate > 0.95 ? '#ecfdf5' : '#fef2f2',
                  color: row.success_rate > 0.95 ? '#059669' : '#b91c1c',
                  fontSize: '12px',
                  fontWeight: 600
                }}>
                  {(row.success_rate * 100).toFixed(1)}%
                </span>
              </td>
              <td>{row.cost_per_1k}</td>
              <td>$0.00</td> {/* TODO: Wire up actual monthly cost if available in comparison object */}
            </tr>
          ))}
        </tbody>
      </table>

      {safeRecommendations.length > 0 && (
        <div className="section" style={{ border: '1px solid #dbeafe', background: '#eff6ff' }}>
          <div className="section-header" style={{ borderBottom: '1px solid #bfdbfe' }}>
            <h3 className="section-title-text" style={{ color: '#1e40af' }}>AI Recommendations</h3>
          </div>
          <div className="section-body">
            <ul className="info-list">
              {safeRecommendations.map((rec, index) => (
                <li key={index} className="info-item" style={{ background: 'white', border: '1px solid #dbeafe' }}>
                  <i className="fas fa-lightbulb" style={{ color: '#3b82f6' }}></i>
                  <span>{rec}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  );
}
