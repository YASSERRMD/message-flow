import { useEffect, useMemo, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

const defaultConfigs = {
  slack: { slack_workspace_id: "", slack_token: "" },
  email: { provider: "sendgrid", api_key: "" },
  webhook: { url: "", secret: "" }
};

export default function IntegrationSettingsPage({ token, csrf }) {
  const [integrations, setIntegrations] = useState([]);
  const [activeType, setActiveType] = useState("slack");
  const [config, setConfig] = useState(defaultConfigs.slack);

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadIntegrations = async () => {
    const response = await fetch(`${API_BASE}/integrations`, { headers });
    if (response.ok) {
      const data = await response.json();
      setIntegrations(data.data || []);
    }
  };

  useEffect(() => {
    if (!token) return;
    loadIntegrations();
  }, [token]);

  useEffect(() => {
    setConfig(defaultConfigs[activeType]);
  }, [activeType]);

  const handleSave = async () => {
    await fetch(`${API_BASE}/integrations/${activeType}`, {
      method: "POST",
      headers,
      body: JSON.stringify({ config })
    });
    loadIntegrations();
  };

  const handleDisconnect = async (id) => {
    await fetch(`${API_BASE}/integrations/${id}`, {
      method: "DELETE",
      headers
    });
    loadIntegrations();
  };

  return (
    <section className="collab-card">
      <header className="collab-header">
        <div>
          <h2>Integrations</h2>
          <p>Connect Slack, Email, or Webhook endpoints to sync activity.</p>
        </div>
      </header>

      <div className="pill-tabs">
        {["slack", "email", "webhook"].map((item) => (
          <button
            key={item}
            className={activeType === item ? "active" : ""}
            onClick={() => setActiveType(item)}
          >
            {item}
          </button>
        ))}
      </div>

      <div className="collab-form vertical">
        {Object.keys(config).map((key) => (
          <label key={key}>
            {key}
            <input
              type={key.includes("token") || key.includes("key") || key.includes("secret") ? "password" : "text"}
              value={config[key]}
              onChange={(event) => setConfig({ ...config, [key]: event.target.value })}
            />
          </label>
        ))}
        <button className="primary" onClick={handleSave}>Save Integration</button>
      </div>

      <div className="collab-subsection">
        <h3>Connected</h3>
        {integrations.map((integration) => (
          <div className="collab-row" key={integration.id}>
            <div>
              <strong>{integration.type}</strong>
              <p>Status: {integration.status}</p>
            </div>
            <div className="collab-row-actions">
              <button className="danger" onClick={() => handleDisconnect(integration.id)}>Disconnect</button>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
