import { useCallback, useEffect, useMemo, useState } from "react";
import useStoredState from "../../hooks/useStoredState.js";
import ProvidersListView from "./ProvidersListView.jsx";
import ProviderDetailsPanel from "./ProviderDetailsPanel.jsx";
import AddProviderModal from "./AddProviderModal.jsx";
import HealthStatusDashboard from "./HealthStatusDashboard.jsx";
import CostAnalyticsDashboard from "./CostAnalyticsDashboard.jsx";
import PerformanceComparisonTable from "./PerformanceComparisonTable.jsx";
import FeatureAssignmentPanel from "./FeatureAssignmentPanel.jsx";
import SettingsPanel from "./SettingsPanel.jsx";
import AlertPanel from "./AlertPanel.jsx";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";

const providerModels = {
  claude: ["claude-3-opus-20240229", "claude-3-sonnet-20240229"],
  openai: ["gpt-4-turbo", "gpt-4o", "gpt-4.1"],
  azure_openai: ["gpt-4o", "gpt-4.1", "gpt-35-turbo"],
  cohere: ["command-r-plus", "command-r"],
  gemini: ["gemini-1.5-pro", "gemini-1.5-flash"],
  anthropic: ["claude-3-opus-20240229"]
};

export default function LLMProviderDashboard({ token, csrf }) {
  const [theme] = useStoredState("mf-theme", "light");
  const [providers, setProviders] = useState([]);
  const [comparison, setComparison] = useState([]);
  const [health, setHealth] = useState([]);
  const [costs, setCosts] = useState({ by_provider: [], by_feature: [], by_day: [] });
  const [usageByFeature, setUsageByFeature] = useState([]);
  const [features, setFeatures] = useState([]);
  const [recommendations, setRecommendations] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [selectedProvider, setSelectedProvider] = useState(null);
  const [showAdd, setShowAdd] = useState(false);
  const [loading, setLoading] = useState(true);
  const [filters, setFilters] = useState({ status: "all", type: "all", active: "all" });
  const [sortKey, setSortKey] = useState("health");

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadAll = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const [providersRes, comparisonRes, healthRes, costsRes, usageRes, featuresRes, recRes] = await Promise.all([
        fetch(`${API_BASE}/llm/providers`, { headers }),
        fetch(`${API_BASE}/llm/providers/comparison`, { headers }),
        fetch(`${API_BASE}/llm/health`, { headers }),
        fetch(`${API_BASE}/llm/analytics/cost-breakdown`, { headers }),
        fetch(`${API_BASE}/llm/analytics/usage-by-feature`, { headers }),
        fetch(`${API_BASE}/llm/features`, { headers }),
        fetch(`${API_BASE}/llm/recommendations`, { headers })
      ]);

      if (providersRes.ok) {
        const data = await providersRes.json();
        setProviders(data.data || []);
      }
      if (comparisonRes.ok) {
        const data = await comparisonRes.json();
        setComparison(data.data || []);
      }
      if (healthRes.ok) {
        const data = await healthRes.json();
        setHealth(Array.isArray(data.data) ? data.data : []);
      }
      if (costsRes.ok) {
        const costData = await costsRes.json();
        setCosts(costData || { by_provider: [], by_feature: [], by_day: [] });
      }
      if (usageRes.ok) {
        const data = await usageRes.json();
        setUsageByFeature(data.data || []);
      }
      if (featuresRes.ok) {
        const data = await featuresRes.json();
        setFeatures(data.data || []);
      }
      if (recRes.ok) {
        const data = await recRes.json();
        setRecommendations(data.data || []);
      }
    } catch (e) {
      console.error("Failed to load dashboard data", e);
    } finally {
      setLoading(false);
    }
  }, [headers, token]);

  useEffect(() => {
    loadAll();
    const timer = setInterval(loadAll, 15000);
    return () => clearInterval(timer);
  }, [loadAll]);

  useEffect(() => {
    const alertsList = [];
    if (Array.isArray(health)) {
      health.forEach((item) => {
        if (item.status === "unhealthy" || item.status === "error") {
          alertsList.push({
            id: `health-${item.provider_id}`,
            type: "error",
            message: `${item.provider} is unhealthy`
          });
        }
        if (item.status === "slow") {
          alertsList.push({
            id: `slow-${item.provider_id}`,
            type: "warning",
            message: `${item.provider} is slow (>${Math.round(item.avg_latency_ms)}ms)`
          });
        }
      });
    }

    if (costs?.by_provider && Array.isArray(providers)) {
      costs.by_provider.forEach((item) => {
        const provider = providers.find((p) => p.provider_name === item.provider);
        if (provider?.monthly_budget && item.total_cost >= provider.monthly_budget * 0.8) {
          alertsList.push({
            id: `budget-${provider.id}`,
            type: "warning",
            message: `${provider.provider_name} budget above 80%`
          });
        }
      });
    }
    setAlerts(alertsList);
  }, [health, costs, providers]);

  useEffect(() => {
    if (!costs?.by_provider?.length) return;
    setProviders((prev) => {
      if (!Array.isArray(prev)) return [];
      return prev.map((provider) => {
        const match = costs.by_provider.find((item) => item.provider === provider.provider_name);
        return { ...provider, monthly_spent: match ? match.total_cost : 0 };
      });
    });
  }, [costs]);

  useEffect(() => {
    if (!providers.length) return;
    setComparison((prev) =>
      prev.map((row) => {
        const provider = providers.find((p) => p.id === row.provider_id);
        if (!provider) return row;
        return {
          ...row,
          cost_per_1k: (provider.cost_per_1k_input + provider.cost_per_1k_output).toFixed(3)
        };
      })
    );
  }, [providers]);

  const filteredProviders = useMemo(() => {
    return providers
      .filter((provider) => {
        if (filters.active === "active" && !provider.is_active) return false;
        if (filters.active === "inactive" && provider.is_active) return false;
        if (filters.type !== "all" && provider.provider_name !== filters.type) return false;
        if (filters.status !== "all" && provider.health_status !== filters.status) return false;
        return true;
      })
      .sort((a, b) => {
        if (sortKey === "health") {
          return (a.health_status || "").localeCompare(b.health_status || "");
        }
        if (sortKey === "cost") {
          return (b.cost_per_1k_input + b.cost_per_1k_output) - (a.cost_per_1k_input + a.cost_per_1k_output);
        }
        if (sortKey === "last_check") {
          return new Date(b.last_health_check || 0) - new Date(a.last_health_check || 0);
        }
        if (sortKey === "created") {
          return new Date(b.created_at || 0) - new Date(a.created_at || 0);
        }
        return 0;
      });
  }, [providers, filters, sortKey]);

  const handleTest = async (providerId) => {
    await fetch(`${API_BASE}/llm/providers/${providerId}/test`, {
      method: "POST",
      headers
    });
    loadAll();
  };

  const handleRemove = async (providerId) => {
    await fetch(`${API_BASE}/llm/providers/${providerId}`, {
      method: "DELETE",
      headers
    });
    loadAll();
  };

  const handleUpdate = async (providerId, payload) => {
    await fetch(`${API_BASE}/llm/providers/${providerId}`, {
      method: "PATCH",
      headers,
      body: JSON.stringify(payload)
    });
    loadAll();
  };

  const handleAdd = async (payload) => {
    const response = await fetch(`${API_BASE}/llm/providers`, {
      method: "POST",
      headers,
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      return response;
    }
    const created = await response.json();
    const test = await fetch(`${API_BASE}/llm/providers/${created.id}/test`, {
      method: "POST",
      headers
    });
    if (!test.ok) {
      await fetch(`${API_BASE}/llm/providers/${created.id}`, {
        method: "PATCH",
        headers,
        body: JSON.stringify({ is_active: false })
      });
      return test;
    }
    setShowAdd(false);
    loadAll();
    return test;
  };

  const handleBulkTest = async () => {
    await fetch(`${API_BASE}/llm/bulk-test`, { method: "POST", headers });
    loadAll();
  };

  const handleAssignProvider = async (feature, payload) => {
    await fetch(`${API_BASE}/llm/features/${feature}/assign-provider`, {
      method: "POST",
      headers,
      body: JSON.stringify(payload)
    });
    loadAll();
  };

  const handleRemoveFeatureProvider = async (feature, providerId) => {
    await fetch(`${API_BASE}/llm/features/${feature}/providers/${providerId}`, {
      method: "DELETE",
      headers
    });
    loadAll();
  };

  return (
    <div className="main-container">
      {/* Sidebar for navigation or filters could go here, but using full width for now */}
      <div className="chat-area" style={{ background: theme === 'dark' ? '#0f0f0f' : '#f8f9fa' }}>
        <div className="chat-header">
          <div className="chat-user-info">
            <div className="chat-avatar" style={{ background: 'linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%)', color: 'white' }}>
              <i className="fas fa-brain"></i>
            </div>
            <div className="chat-details">
              <h3>Provider Management</h3>
              <p>Track health, cost, and performance for every model</p>
            </div>
          </div>
          <div className="chat-actions">
            <button className="action-btn primary" onClick={() => setShowAdd(true)}>
              <i className="fas fa-plus"></i> Add Provider
            </button>
            <button className="action-btn" onClick={loadAll}>
              <i className={`fas fa-sync ${loading ? 'fa-spin' : ''}`}></i> Refresh
            </button>
          </div>
        </div>

        <div className="messages-container" style={{ padding: '24px' }}>

          {/* Stats Grid */}
          <div className="stats-grid" style={{ gridTemplateColumns: 'repeat(4, 1fr)', marginBottom: '24px' }}>
            <div className="stat-box">
              <div className="stat-value">{health.filter(h => h.status === 'healthy').length}/{providers.length}</div>
              <div className="stat-label">Healthy Providers</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">${(costs?.total_cost || 0).toFixed(2)}</div>
              <div className="stat-label">Monthly Spend</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">{comparison.length}</div>
              <div className="stat-label">Models Tracked</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">{alerts.length}</div>
              <div className="stat-label">Active Alerts</div>
            </div>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '24px' }}>

            {/* Main Content Column */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>

              {/* Providers List */}
              <div className="connect-card" style={{ maxWidth: '100%', padding: '24px', textAlign: 'left' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
                  <h2 style={{ fontSize: '18px', fontWeight: '600' }}>Active Providers</h2>

                </div>
                <ProvidersListView
                  providers={filteredProviders}
                  health={health}
                  onSelect={setSelectedProvider}
                  selectedId={selectedProvider?.id}
                  filters={filters}
                  setFilters={setFilters}
                  sortKey={sortKey}
                  setSortKey={setSortKey}
                  onTest={handleTest}
                  onRemove={handleRemove}
                  loading={loading}
                />
              </div>

              {/* Performance Comparison */}
              <div className="connect-card" style={{ maxWidth: '100%', padding: '24px', textAlign: 'left' }}>
                <h2 style={{ fontSize: '18px', fontWeight: '600', marginBottom: '16px' }}>Performance Benchmarks</h2>
                <PerformanceComparisonTable data={comparison} />
              </div>

              {/* Cost Analytics */}
              <div className="connect-card" style={{ maxWidth: '100%', padding: '24px', textAlign: 'left' }}>
                <h2 style={{ fontSize: '18px', fontWeight: '600', marginBottom: '16px' }}>Cost Analysis</h2>
                <CostAnalyticsDashboard data={costs} />
              </div>

            </div>

            {/* Right Sidebar Column */}
            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>

              {/* Recommendations */}
              {recommendations.length > 0 && (
                <div className="connect-card" style={{ maxWidth: '100%', padding: '24px', textAlign: 'left', background: '#f0fdf4', borderColor: '#bbf7d0' }}>
                  <h2 style={{ fontSize: '16px', fontWeight: '600', color: '#166534', marginBottom: '12px' }}>
                    <i className="fas fa-lightbulb"></i> Optimization Tips
                  </h2>
                  <ul style={{ listStyle: 'none', padding: 0 }}>
                    {recommendations.map((rec, i) => (
                      <li key={i} style={{ fontSize: '13px', marginBottom: '8px', color: '#166534' }}>• {rec.message}</li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Feature Assignment */}
              <div className="connect-card" style={{ maxWidth: '100%', padding: '24px', textAlign: 'left' }}>
                <h2 style={{ fontSize: '18px', fontWeight: '600', marginBottom: '16px' }}>Routing Rules</h2>
                <FeatureAssignmentPanel features={features} providers={providers} />
              </div>

              {/* Global Settings */}
              <div className="connect-card" style={{ maxWidth: '100%', padding: '24px', textAlign: 'left' }}>
                <h2 style={{ fontSize: '18px', fontWeight: '600', marginBottom: '16px' }}>Global Settings</h2>
                <SettingsPanel />
              </div>

            </div>
          </div>

        </div>
      </div>

      {showAdd && <AddProviderModal onClose={() => setShowAdd(false)} onSave={loadAll} token={token} />}

      {selectedProvider && (
        <div className="summary-modal-overlay">
          <div className="summary-modal" style={{ maxWidth: '800px' }}>
            <div className="summary-header">
              <h3>{selectedProvider.provider_name} Details</h3>
              <button className="close-btn" onClick={() => setSelectedProvider(null)}><i className="fas fa-times"></i></button>
            </div>
            <div className="summary-content">
              <ProviderDetailsPanel
                provider={selectedProvider}
                onClose={() => setSelectedProvider(null)}
                onUpdate={handleUpdate}
                headers={headers}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
