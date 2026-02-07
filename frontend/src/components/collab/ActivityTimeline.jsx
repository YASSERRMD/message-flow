import { useEffect, useMemo, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

export default function ActivityTimeline({ token, csrf }) {
  const [activity, setActivity] = useState([]);

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadActivity = async () => {
    const response = await fetch(`${API_BASE}/team/activity`, { headers });
    if (response.ok) {
      const data = await response.json();
      setActivity(data.data || []);
    }
  };

  useEffect(() => {
    if (!token) return;
    loadActivity();
  }, [token]);

  return (
    <section className="collab-card">
      <header className="collab-header">
        <div>
          <h2>Activity Timeline</h2>
          <p>Real-time log of team operations and message events.</p>
        </div>
        <button className="ghost" onClick={loadActivity}>Refresh</button>
      </header>
      <div className="timeline">
        {activity.map((item, index) => (
          <div className="timeline-row" key={`${item.user_id}-${index}`}>
            <div className="timeline-dot" />
            <div>
              <strong>{item.action}</strong>
              <p>User {item.user_id} · {new Date(item.created_at).toLocaleString()}</p>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
