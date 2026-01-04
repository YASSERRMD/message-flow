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
    <section className="section" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <header className="section-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h2 className="section-title-text">Providers</h2>
          <p className="section-subtitle">Manage active models and budgets</p>
        </div>
        <div className="panel-actions">
          {/* Sort layout if needed, but filters bar below handles it well */}
        </div>
      </header>

      <div className="section-body" style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <div className="filters-bar">
          <select className="filter-dropdown" value={sortKey} onChange={(event) => setSortKey(event.target.value)}>
            <option value="health">Sort by health</option>
            <option value="cost">Sort by cost</option>
            <option value="created">Sort by creation</option>
          </select>
          <select className="filter-dropdown" value={filters.type} onChange={(event) => setFilters({ ...filters, type: event.target.value })}>
            <option value="all">All providers</option>
            <option value="claude">Claude</option>
            <option value="openai">OpenAI</option>
            <option value="cohere">Cohere</option>
            <option value="azure">Azure</option>
            <option value="gemini">Gemini</option>
          </select>
          <select className="filter-dropdown" value={filters.active} onChange={(event) => setFilters({ ...filters, active: event.target.value })}>
            <option value="all">Active + Inactive</option>
            <option value="active">Active Only</option>
            <option value="inactive">Inactive Only</option>
          </select>
          <select className="filter-dropdown" value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}>
            <option value="all">All Health</option>
            <option value="healthy">Healthy</option>
            <option value="degraded">Degraded</option>
            <option value="unhealthy">Unhealthy</option>
          </select>
        </div>

        {loading ? (
          <div className="empty-state">
            <i className="fas fa-spinner fa-spin empty-icon"></i>
            <p className="empty-text">Loading providers...</p>
          </div>
        ) : (
          <div className="provider-grid" style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
            gap: '20px',
            overflowY: 'auto',
            paddingRight: '4px'
          }}>
            {providers.length === 0 && (
              <div className="empty-state" style={{ gridColumn: '1 / -1' }}>
                <div className="empty-icon"><i className="fas fa-server"></i></div>
                <p className="empty-text">No providers found matching your filters.</p>
              </div>
            )}
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
      </div>
    </section>
  );
}
