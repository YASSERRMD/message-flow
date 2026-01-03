import { useEffect, useMemo, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";

export default function AnalyticsPage({ token, csrf }) {
  const [activity, setActivity] = useState([]);
  const [usage, setUsage] = useState(null);

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadAnalytics = async () => {
    const [activityRes, usageRes] = await Promise.all([
      fetch(`${API_BASE}/team/activity`, { headers }),
      fetch(`${API_BASE}/llm/usage`, { headers })
    ]);
    if (activityRes.ok) {
      const data = await activityRes.json();
      setActivity(data.data || []);
    }
    if (usageRes.ok) {
      setUsage(await usageRes.json());
    }
  };

  useEffect(() => {
    if (!token) return;
    loadAnalytics();
  }, [token]);

  const activityCount = activity.length;

  return (
    <section className="collab-card">
      <header className="collab-header">
        <div>
          <h2>Analytics Overview</h2>
          <p>Quick signals on team activity and model usage.</p>
        </div>
        <button className="ghost" onClick={loadAnalytics}>Refresh</button>
      </header>

      <div className="stat-grid">
        <div className="stat-card">
          <h4>Team Actions</h4>
          <strong>{activityCount}</strong>
          <span>events in last pull</span>
        </div>
        <div className="stat-card">
          <h4>Total LLM Requests</h4>
          <strong>{usage?.total_requests || 0}</strong>
          <span>logged requests</span>
        </div>
        <div className="stat-card">
          <h4>Failure Rate</h4>
          <strong>
            {usage ? `${Math.round((usage.failed_requests / Math.max(usage.total_requests, 1)) * 100)}%` : "0%"}
          </strong>
          <span>across providers</span>
        </div>
        <div className="stat-card">
          <h4>Total LLM Cost</h4>
          <strong>${usage?.total_cost?.toFixed(2) || "0.00"}</strong>
          <span>this period</span>
        </div>
      </div>
    </section>
  );
}
