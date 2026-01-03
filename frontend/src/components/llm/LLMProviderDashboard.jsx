import { useCallback, useEffect, useMemo, useState } from "react";
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
        setHealth(data.data || []);
      }
      if (costsRes.ok) {
        setCosts(await costsRes.json());
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
    costs.by_provider?.forEach((item) => {
      const provider = providers.find((p) => p.provider_name === item.provider);
      if (provider?.monthly_budget && item.total_cost >= provider.monthly_budget * 0.8) {
        alertsList.push({
          id: `budget-${provider.id}`,
          type: "warning",
          message: `${provider.provider_name} budget above 80%`
        });
      }
    });
    setAlerts(alertsList);
  }, [health, costs, providers]);

  useEffect(() => {
    if (!costs.by_provider?.length) return;
    setProviders((prev) =>
      prev.map((provider) => {
        const match = costs.by_provider.find((item) => item.provider === provider.provider_name);
        return { ...provider, monthly_spent: match ? match.total_cost : 0 };
      })
    );
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
    <div className="llm-dashboard">
      <header className="llm-header">
        <div>
          <p className="eyebrow">LLM Control Room</p>
          <h1>Provider Management</h1>
          <p className="subtitle">Track health, cost, and performance for every model in your stack.</p>
        </div>
        <div className="llm-header-actions">
          <button className="primary" onClick={() => setShowAdd(true)}>Add provider</button>
          <button className="ghost" onClick={handleBulkTest}>Test all</button>
        </div>
      </header>

      <AlertPanel alerts={alerts} />

      <div className="llm-grid">
        <ProvidersListView
          providers={filteredProviders}
          onSelect={setSelectedProvider}
          onTest={handleTest}
          onRemove={handleRemove}
          filters={filters}
          setFilters={setFilters}
          sortKey={sortKey}
          setSortKey={setSortKey}
          loading={loading}
        />
        <ProviderDetailsPanel
          provider={selectedProvider}
          onClose={() => setSelectedProvider(null)}
          onUpdate={handleUpdate}
          headers={headers}
        />
      </div>

      <div className="llm-row">
        <HealthStatusDashboard health={health} comparison={comparison} />
        <CostAnalyticsDashboard costs={costs} usageByFeature={usageByFeature} />
      </div>

      <div className="llm-row">
        <PerformanceComparisonTable comparison={comparison} recommendations={recommendations} />
        <FeatureAssignmentPanel
          features={features}
          providers={providers}
          onAssign={handleAssignProvider}
          onRemove={handleRemoveFeatureProvider}
        />
      </div>

      <SettingsPanel providers={providers} onUpdate={handleUpdate} />

      <AddProviderModal
        open={showAdd}
        onClose={() => setShowAdd(false)}
        onSubmit={handleAdd}
        providerModels={providerModels}
      />
    </div>
  );
}
