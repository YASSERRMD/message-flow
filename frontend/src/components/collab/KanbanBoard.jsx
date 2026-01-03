import { useEffect, useMemo, useState } from "react";
import CommentThread from "./CommentThread.jsx";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

const columns = [
  { id: "new", title: "New" },
  { id: "in_progress", title: "In Progress" },
  { id: "in_review", title: "In Review" },
  { id: "done", title: "Done" }
];

export default function KanbanBoard({ token, csrf }) {
  const [items, setItems] = useState([]);
  const [expanded, setExpanded] = useState(null);

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadItems = async () => {
    const response = await fetch(`${API_BASE}/action-items`, { headers });
    if (response.ok) {
      const data = await response.json();
      setItems(data.data || []);
    }
  };

  useEffect(() => {
    if (!token) return;
    loadItems();
  }, [token]);

  const updateStatus = async (item, status) => {
    await fetch(`${API_BASE}/action-items/${item.id}`, {
      method: "PATCH",
      headers,
      body: JSON.stringify({ status })
    });
    loadItems();
  };

  return (
    <section className="collab-card">
      <header className="collab-header">
        <div>
          <h2>Action Item Board</h2>
          <p>Track progress and collaborate on shared tasks.</p>
        </div>
        <button className="ghost" onClick={loadItems}>Refresh</button>
      </header>

      <div className="kanban">
        {columns.map((column) => (
          <div className="kanban-column" key={column.id}>
            <h3>{column.title}</h3>
            {items.filter((item) => {
              const status = item.status === "open" ? "new" : item.status;
              return status === column.id;
            }).map((item) => (
              <div className="kanban-card" key={item.id}>
                <strong>{item.description}</strong>
                <p>Assigned: {item.assigned_to || "Unassigned"}</p>
                <div className="kanban-actions">
                  <select
                    value={item.status === "open" ? "new" : item.status}
                    onChange={(event) => updateStatus(item, event.target.value)}
                  >
                    {columns.map((col) => (
                      <option key={col.id} value={col.id}>{col.title}</option>
                    ))}
                  </select>
                  <button className="ghost" onClick={() => setExpanded(expanded === item.id ? null : item.id)}>
                    {expanded === item.id ? "Hide" : "Comments"}
                  </button>
                </div>
                {expanded === item.id && (
                  <CommentThread actionItemId={item.id} token={token} csrf={csrf} />
                )}
              </div>
            ))}
          </div>
        ))}
      </div>
    </section>
  );
}
