import { useEffect, useMemo, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

export default function AuditLogPage({ token, csrf }) {
  const [logs, setLogs] = useState([]);
  const [action, setAction] = useState("");
  const [userId, setUserId] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadLogs = async () => {
    const payload = {
      action: action || undefined,
      user_id: userId ? Number(userId) : undefined,
      start_date: startDate || undefined,
      end_date: endDate || undefined
    };
    const response = await fetch(`${API_BASE}/audit-logs`, {
      method: "POST",
      headers,
      body: JSON.stringify(payload)
    });
    if (response.ok) {
      const data = await response.json();
      setLogs(data.data || []);
    }
  };

  useEffect(() => {
    if (!token) return;
    loadLogs();
  }, [token]);

  return (
    <section className="collab-card">
      <header className="collab-header">
        <div>
          <h2>Audit Logs</h2>
          <p>Trace permission checks, configuration changes, and team actions.</p>
        </div>
        <button className="ghost" onClick={loadLogs}>Refresh</button>
      </header>

      <div className="collab-form">
        <input placeholder="Action" value={action} onChange={(event) => setAction(event.target.value)} />
        <input placeholder="User ID" value={userId} onChange={(event) => setUserId(event.target.value)} />
        <input type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} />
        <input type="date" value={endDate} onChange={(event) => setEndDate(event.target.value)} />
        <button className="primary" onClick={loadLogs}>Filter</button>
      </div>

      <div className="collab-list">
        {logs.map((log) => (
          <div className="collab-row" key={log.id}>
            <div>
              <strong>{log.action}</strong>
              <p>{log.resource_type || "system"} · {new Date(log.created_at).toLocaleString()}</p>
            </div>
            <div className="collab-row-meta">
              <span>User: {log.user_id || "system"}</span>
              <span>IP: {log.ip_address || "-"}</span>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
