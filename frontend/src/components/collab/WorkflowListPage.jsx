import { useEffect, useMemo, useState } from "react";
import WorkflowBuilder from "./WorkflowBuilder.jsx";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

export default function WorkflowListPage({ token, csrf }) {
  const [workflows, setWorkflows] = useState([]);
  const [showBuilder, setShowBuilder] = useState(false);

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadWorkflows = async () => {
    const response = await fetch(`${API_BASE}/workflows`, { headers });
    if (response.ok) {
      const data = await response.json();
      setWorkflows(data.data || []);
    }
  };

  useEffect(() => {
    if (!token) return;
    loadWorkflows();
  }, [token]);

  const handleCreate = async (payload) => {
    let actions = payload.actions;
    try {
      JSON.parse(actions);
    } catch (err) {
      actions = JSON.stringify({ actions: [] });
    }
    await fetch(`${API_BASE}/workflows`, {
      method: "POST",
      headers,
      body: JSON.stringify({
        name: payload.name,
        trigger: payload.trigger,
        actions: JSON.parse(actions),
        enabled: payload.enabled
      })
    });
    setShowBuilder(false);
    loadWorkflows();
  };

  const handleToggle = async (workflow) => {
    await fetch(`${API_BASE}/workflows/${workflow.id}`, {
      method: "PATCH",
      headers,
      body: JSON.stringify({ enabled: !workflow.enabled })
    });
    loadWorkflows();
  };

  const handleTest = async (id) => {
    await fetch(`${API_BASE}/workflows/${id}/test`, {
      method: "POST",
      headers
    });
  };

  const handleDelete = async (id) => {
    await fetch(`${API_BASE}/workflows/${id}`, {
      method: "DELETE",
      headers
    });
    loadWorkflows();
  };

  return (
    <section className="collab-card">
      <header className="collab-header">
        <div>
          <h2>Workflow Automation</h2>
          <p>Build conditional flows to handle urgent messages automatically.</p>
        </div>
        <button className="primary" onClick={() => setShowBuilder(true)}>New Workflow</button>
      </header>

      <div className="collab-list">
        {workflows.map((workflow) => (
          <div className="collab-row" key={workflow.id}>
            <div>
              <strong>{workflow.name}</strong>
              <p>{workflow.trigger} · {workflow.enabled ? "Enabled" : "Disabled"}</p>
            </div>
            <div className="collab-row-actions">
              <button className="ghost" onClick={() => handleTest(workflow.id)}>Test</button>
              <button className="ghost" onClick={() => handleToggle(workflow)}>
                {workflow.enabled ? "Disable" : "Enable"}
              </button>
              <button className="danger" onClick={() => handleDelete(workflow.id)}>Delete</button>
            </div>
          </div>
        ))}
      </div>

      {showBuilder && (
        <WorkflowBuilder
          onSubmit={handleCreate}
          onClose={() => setShowBuilder(false)}
        />
      )}
    </section>
  );
}
