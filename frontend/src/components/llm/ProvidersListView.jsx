import ProviderCard from "./ProviderCard.jsx";

export default function ProvidersListView({
  providers,
  onSelect,
  onTest,
  onRemove,
  filters,
  setFilters,
  sortKey,
  setSortKey,
  loading
}) {
  return (
    <section className="panel llm-panel">
      <header className="panel-header">
        <div>
          <h3>Providers</h3>
          <p className="panel-sub">Manage active models and budgets</p>
        </div>
        <div className="panel-actions">
          <select value={sortKey} onChange={(event) => setSortKey(event.target.value)}>
            <option value="health">Sort by health</option>
            <option value="cost">Sort by cost</option>
            <option value="last_check">Sort by last check</option>
            <option value="created">Sort by creation</option>
          </select>
        </div>
      </header>
      <div className="filters">
        <select value={filters.type} onChange={(event) => setFilters({ ...filters, type: event.target.value })}>
          <option value="all">All providers</option>
          <option value="claude">Claude</option>
          <option value="openai">OpenAI</option>
          <option value="cohere">Cohere</option>
        </select>
        <select value={filters.active} onChange={(event) => setFilters({ ...filters, active: event.target.value })}>
          <option value="all">Active + inactive</option>
          <option value="active">Active only</option>
          <option value="inactive">Inactive only</option>
        </select>
        <select value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}>
          <option value="all">All health</option>
          <option value="ok">Healthy</option>
          <option value="slow">Slow</option>
          <option value="unhealthy">Unhealthy</option>
          <option value="error">Error</option>
        </select>
      </div>

      {loading ? (
        <p className="empty">Loading providers…</p>
      ) : (
        <div className="provider-grid">
          {providers.length === 0 && <p className="empty">No providers configured yet.</p>}
          {providers.map((provider) => (
            <ProviderCard
              key={provider.id}
              provider={provider}
              onSelect={() => onSelect(provider)}
              onTest={() => onTest(provider.id)}
              onRemove={() => onRemove(provider.id)}
            />
          ))}
        </div>
      )}
    </section>
  );
}
