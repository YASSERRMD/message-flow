import { useState, useEffect, useMemo, useCallback } from "react";
import useStoredState from "../../hooks/useStoredState";
import AddProviderModal from "./AddProviderModal";
import PerformanceComparisonTable from "./PerformanceComparisonTable";
import ProvidersListView from "./ProvidersListView";
import ProviderDetailsPanel from "./ProviderDetailsPanel";
import FeatureAssignmentPanel from "./FeatureAssignmentPanel";
import SettingsPanel from "./SettingsPanel";
import "./llm-dashboard.css";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";

export default function LLMProviderDashboard({ onNavigate, token, csrf }) {
  // --- View State ---
  const [activeView, setActiveView] = useState("dashboard"); // dashboard, providers, performance, configuration
  const [showAdd, setShowAdd] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState(null);

  // --- Data State ---
  const [providers, setProviders] = useState([]);
  const [health, setHealth] = useState([]);
  const [costs, setCosts] = useState({ by_provider: [], by_feature: [], daily: [], total: 0 });
  const [usageStats, setUsageStats] = useState(null);
  const [features, setFeatures] = useState([]);

  // --- Filter/Sort State ---
  const [filters, setFilters] = useState({ type: "all", active: "all", status: "all" });
  const [sortKey, setSortKey] = useState("health");
  const [loading, setLoading] = useState(false);

  // --- Routing/Settings State ---
  const [settings, setSettings] = useStoredState("llm-global-settings", {
    budgetAlert: 80,
    healthInterval: 5,
    fallback: true
  });

  // --- Fetch Data ---
  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const headers = { "Authorization": `Bearer ${token}` };

      const [pRes, hRes, cRes, uRes, fRes] = await Promise.all([
        fetch(`${API_BASE}/llm/providers`, { headers }),
        fetch(`${API_BASE}/llm/health`, { headers }).catch(() => ({ ok: false })),
        fetch(`${API_BASE}/llm/costs`, { headers }),
        fetch(`${API_BASE}/llm/usage`, { headers }),
        fetch(`${API_BASE}/llm/features`, { headers }).catch(() => ({ ok: false }))
      ]);

      if (pRes.ok) {
        const data = await pRes.json();
        setProviders(data.data || []);
      }
      if (hRes.ok) {
        const data = await hRes.json();
        setHealth(data.data || []);
      }
      if (cRes.ok) {
        setCosts(await cRes.json());
      }
      if (uRes.ok) {
        setUsageStats(await uRes.json());
      }
      if (fRes.ok) {
        const featData = await fRes.json();
        setFeatures(featData.data || featData || []);
      }
    } catch (err) {
      console.error("Failed to load dashboard data", err);
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    if (token) loadAll();
  }, [loadAll, token]);

  // --- Computed Data ---
  const filteredProviders = useMemo(() => {
    let list = [...providers];
    if (filters.type !== "all") list = list.filter(p => p.provider_name.toLowerCase().includes(filters.type));
    if (filters.active !== "all") list = list.filter(p => filters.active === "active" ? p.is_active : !p.is_active);
    if (filters.status !== "all") list = list.filter(p => p.health_status === filters.status);

    return list.sort((a, b) => {
      if (sortKey === "health") return (a.health_status === "healthy" ? -1 : 1) - (b.health_status === "healthy" ? -1 : 1);
      if (sortKey === "created") return new Date(b.created_at) - new Date(a.created_at);
      if (sortKey === "cost") return (b.cost_per_1k_input + b.cost_per_1k_output) - (a.cost_per_1k_input + a.cost_per_1k_output);
      return 0;
    });
  }, [providers, filters, sortKey]);

  const comparisonData = useMemo(() => {
    return providers.map(p => {
      const h = health.find(x => x.provider_id === p.id);
      return {
        provider_name: p.provider_name,
        avg_latency: h?.latency_ms || 0,
        success_rate: h?.status === 'healthy' ? 1.0 : (h?.status === 'degraded' ? 0.9 : 0.5),
        cost_per_1k: `$${(p.cost_per_1k_input + p.cost_per_1k_output).toFixed(3)}`
      };
    });
  }, [providers, health]);

  // --- Actions ---
  const handleAdd = async (newProvider) => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/llm/providers`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
          "X-CSRF-Token": csrf
        },
        body: JSON.stringify(newProvider)
      });
      if (res.ok) {
        setShowAdd(false);
        loadAll();
        setActiveView("providers"); // Switch to list
        return { ok: true };
      } else {
        const errorData = await res.json().catch(() => ({}));
        alert(`Failed to create provider: ${errorData.error || res.statusText}`);
        return { ok: false, error: errorData.error };
      }
    } catch (e) {
      console.error(e);
      alert("Error adding provider");
      return { ok: false, error: e.message };
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateProvider = async (id, updates) => {
    try {
      const res = await fetch(`${API_BASE}/llm/providers/${id}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
          "X-CSRF-Token": csrf
        },
        body: JSON.stringify(updates)
      });
      if (res.ok) {
        loadAll();
        setSelectedProvider(null);
      } else {
        alert("Failed to update provider");
      }
    } catch (e) {
      console.error(e);
      alert("Error updating provider");
    }
  };

  const handleRemoveProvider = async (id) => {
    if (!confirm("Are you sure you want to remove this provider?")) return;
    try {
      const res = await fetch(`${API_BASE}/llm/providers/${id}`, {
        method: "DELETE",
        headers: {
          "Authorization": `Bearer ${token}`,
          "X-CSRF-Token": csrf
        }
      });
      if (res.ok) {
        loadAll();
        if (selectedProvider?.id === id) setSelectedProvider(null);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleAssignRoute = async (feature, assignment) => {
    try {
      const res = await fetch(`${API_BASE}/llm/features/${feature}/assign-provider`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": `Bearer ${token}`,
          "X-CSRF-Token": csrf
        },
        body: JSON.stringify(assignment)
      });
      if (res.ok) {
        alert(`Route assigned for ${feature}`);
        loadAll(); // Refresh features after assignment
      } else {
        alert("Failed to assign route");
      }
    } catch (e) {
      console.error(e);
      alert("Error assigning route");
    }
  };

  const handleRemoveRoute = async (feature, providerId) => {
    try {
      const res = await fetch(`${API_BASE}/llm/features/${feature}/providers/${providerId}`, {
        method: "DELETE",
        headers: {
          "Authorization": `Bearer ${token}`,
          "X-CSRF-Token": csrf
        }
      });
      if (res.ok) {
        loadAll(); // Refresh features after removal
      } else {
        alert("Failed to remove provider from feature");
      }
    } catch (e) {
      console.error(e);
      alert("Error removing provider");
    }
  };

  // --- Content Renderers ---

  const renderDashboard = () => (
    <div className="content-body">
      <div className="stats-overview">
        <div className="stat-box">
          <div className="stat-value">{providers.filter(p => p.is_active).length}</div>
          <div className="stat-label">Active Providers</div>
        </div>
        <div className="stat-box">
          <div className="stat-value">${(costs.total || 0).toFixed(2)}</div>
          <div className="stat-label">Total Spend</div>
        </div>
        <div className="stat-box">
          <div className="stat-value">{usageStats?.TotalRequests || 0}</div>
          <div className="stat-label">Total Requests</div>
        </div>
        <div className="stat-box">
          <div className="stat-value">{providers.filter(p => p.health_status !== 'healthy').length}</div>
          <div className="stat-label">Issues</div>
        </div>
      </div>

      <div className="two-column">
        <div className="section">
          <div className="section-header">
            <h2 className="section-title-text">Active Providers</h2>
          </div>
          <div className="section-body">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Provider</th>
                  <th>Model</th>
                  <th>Status</th>
                  <th>Cost/1k</th>
                </tr>
              </thead>
              <tbody>
                {filteredProviders.slice(0, 5).map(p => (
                  <tr key={p.id} onClick={() => { setActiveView("providers"); setSelectedProvider(p); }} style={{ cursor: 'pointer' }}>
                    <td>{p.provider_name}</td>
                    <td>{p.model_name}</td>
                    <td><span style={{ color: p.health_status === 'healthy' ? '#059669' : '#DC2626' }}>●</span> {p.health_status}</td>
                    <td>${(p.cost_per_1k_input + p.cost_per_1k_output).toFixed(3)}</td>
                  </tr>
                ))}
                {filteredProviders.length === 0 && <tr><td colSpan="4" className="empty-text">No providers</td></tr>}
              </tbody>
            </table>
            <button className="action-btn ghost" style={{ marginTop: '12px' }} onClick={() => setActiveView("providers")}>View all providers &rarr;</button>
          </div>
        </div>

        <div className="section">
          <div className="section-header">
            <h2 className="section-title-text">System Health</h2>
          </div>
          <div className="section-body">
            {health.length > 0 ? (
              <ul className="info-list">
                {health.map(h => {
                  const pName = providers.find(p => p.id === h.provider_id)?.provider_name || h.provider_id;
                  return (
                    <li className="info-item" key={h.id}>
                      <span style={{ color: h.status === 'healthy' ? '#059669' : '#DC2626' }}>●</span>
                      <strong>{pName}</strong>: {h.status} ({h.latency_ms}ms)
                    </li>
                  );
                })}
              </ul>
            ) : (
              <p className="empty-text">No health alerts.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );

  return (
    <div className="llm-dashboard-container">
      <div className="main-layout">
        <aside className="sidebar">
          <div className="sidebar-section">
            <h3 className="sidebar-title">Overview</h3>
            <ul className="sidebar-menu">
              <li className={`menu-item ${activeView === "dashboard" ? "active" : ""}`} onClick={() => setActiveView("dashboard")}>
                <i className="fas fa-chart-pie"></i>
                <span>Dashboard</span>
              </li>
              <li className={`menu-item ${activeView === "providers" ? "active" : ""}`} onClick={() => setActiveView("providers")}>
                <i className="fas fa-server"></i>
                <span>Providers</span>
              </li>
              <li className={`menu-item ${activeView === "performance" ? "active" : ""}`} onClick={() => setActiveView("performance")}>
                <i className="fas fa-chart-line"></i>
                <span>Performance</span>
              </li>
              <li className={`menu-item ${activeView === "configuration" ? "active" : ""}`} onClick={() => setActiveView("configuration")}>
                <i className="fas fa-cog"></i>
                <span>Configuration</span>
              </li>
            </ul>
          </div>
        </aside>

        <main className="content-area">
          <div className="content-header">
            <div className="content-title">
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <button className="icon-btn" onClick={() => onNavigate && onNavigate("operations")} title="Back to Operations">
                  <i className="fas fa-arrow-left"></i>
                </button>
                <h1>{activeView.charAt(0).toUpperCase() + activeView.slice(1)}</h1>
              </div>
              <p>Manage and monitor your AI infrastructure</p>
            </div>
            <div className="header-actions">
              <button className="action-btn" onClick={loadAll}>
                <i className="fas fa-sync-alt"></i>
                Refresh
              </button>
              <button className="action-btn-primary action-btn" onClick={() => setShowAdd(true)}>
                <i className="fas fa-plus"></i>
                Add Provider
              </button>
            </div>
          </div>

          {activeView === "dashboard" && renderDashboard()}

          {activeView === "providers" && (
            <div className="content-body" style={{ display: 'flex', gap: '20px', alignItems: 'flex-start' }}>
              <div style={{ flex: 1, minWidth: 0 }}>
                <ProvidersListView
                  providers={filteredProviders}
                  onSelect={(p) => setSelectedProvider(p)}
                  onTest={(id) => console.log("Test", id)}
                  onRemove={handleRemoveProvider}
                  filters={filters}
                  setFilters={setFilters}
                  sortKey={sortKey}
                  setSortKey={setSortKey}
                  loading={loading}
                />
              </div>
              {selectedProvider && (
                <div style={{ width: '400px', flexShrink: 0 }}>
                  <ProviderDetailsPanel
                    provider={selectedProvider}
                    onClose={() => setSelectedProvider(null)}
                    onUpdate={handleUpdateProvider}
                    headers={{ "Authorization": `Bearer ${token}`, "X-CSRF-Token": csrf }}
                  />
                </div>
              )}
            </div>
          )}

          {activeView === "performance" && (
            <div className="content-body">
              <div className="section">
                <div className="section-header">
                  <h2 className="section-title-text">Performance Comparison</h2>
                </div>
                <div className="section-body">
                  <PerformanceComparisonTable comparison={comparisonData} recommendations={[]} />
                </div>
              </div>
            </div>
          )}

          {activeView === "configuration" && (
            <div className="content-body">
              <div className="two-column">
                <FeatureAssignmentPanel
                  features={features.length > 0 ? features : [{ feature: "analyze", providers: [] }, { feature: "summarize", providers: [] }, { feature: "action_items", providers: [] }]}
                  providers={providers}
                  onAssign={handleAssignRoute}
                  onRemove={handleRemoveRoute}
                />
                <SettingsPanel
                  providers={providers}
                  onUpdate={(id) => handleAssignRoute("global_default", { provider_id: id, priority: 1 })}
                />
              </div>
            </div>
          )}
        </main>
      </div>

      {showAdd && <AddProviderModal open={showAdd} onClose={() => setShowAdd(false)} onSubmit={handleAdd} providerModels={{}} />}
    </div>
  );
}
