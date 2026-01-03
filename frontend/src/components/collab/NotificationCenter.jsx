import { useEffect, useMemo, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";

export default function NotificationCenter({ token, csrf }) {
  const [notifications, setNotifications] = useState([]);

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadNotifications = async () => {
    const response = await fetch(`${API_BASE}/notifications`, {
      method: "POST",
      headers,
      body: JSON.stringify({})
    });
    if (response.ok) {
      const data = await response.json();
      setNotifications(data.data || []);
    }
  };

  useEffect(() => {
    if (!token) return;
    loadNotifications();
  }, [token]);

  const markRead = async (id) => {
    await fetch(`${API_BASE}/notifications/${id}`, {
      method: "PATCH",
      headers
    });
    loadNotifications();
  };

  return (
    <section className="collab-card">
      <header className="collab-header">
        <div>
          <h2>Notifications</h2>
          <p>Assignment alerts, workflow updates, and mentions.</p>
        </div>
        <button className="ghost" onClick={loadNotifications}>Refresh</button>
      </header>
      <div className="collab-list">
        {notifications.map((note) => (
          <div className="collab-row" key={note.id}>
            <div>
              <strong>{note.type}</strong>
              <p>{note.content}</p>
            </div>
            <div className="collab-row-actions">
              {!note.read && (
                <button className="ghost" onClick={() => markRead(note.id)}>Mark read</button>
              )}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
