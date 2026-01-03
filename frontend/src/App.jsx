import { useEffect, useMemo, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

const initialSummary = {
  total_conversations: 0,
  total_messages: 0,
  important_messages: 0,
  open_action_items: 0
};

const formatDate = (value) => {
  if (!value) return "--";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "--";
  return date.toLocaleString();
};

export default function App() {
  const [summary, setSummary] = useState(initialSummary);
  const [conversations, setConversations] = useState([]);
  const [importantMessages, setImportantMessages] = useState([]);
  const [actionItems, setActionItems] = useState([]);
  const [dailySummary, setDailySummary] = useState(null);
  const [streamStatus, setStreamStatus] = useState("connecting");

  const stats = useMemo(
    () => [
      { label: "Conversations", value: summary.total_conversations },
      { label: "Messages", value: summary.total_messages },
      { label: "Important", value: summary.important_messages },
      { label: "Open Actions", value: summary.open_action_items }
    ],
    [summary]
  );

  const loadAll = async () => {
    try {
      const [summaryRes, convoRes, importantRes, actionsRes, dailyRes] = await Promise.all([
        fetch(`${API_BASE}/dashboard`),
        fetch(`${API_BASE}/conversations?limit=6`),
        fetch(`${API_BASE}/important-messages?limit=5`),
        fetch(`${API_BASE}/action-items?limit=6`),
        fetch(`${API_BASE}/daily-summary`)
      ]);

      if (summaryRes.ok) {
        setSummary(await summaryRes.json());
      }
      if (convoRes.ok) {
        const data = await convoRes.json();
        setConversations(data.data || []);
      }
      if (importantRes.ok) {
        const data = await importantRes.json();
        setImportantMessages(data.data || []);
      }
      if (actionsRes.ok) {
        const data = await actionsRes.json();
        setActionItems(data.data || []);
      }
      if (dailyRes.ok) {
        setDailySummary(await dailyRes.json());
      }
    } catch (error) {
      setStreamStatus("offline");
    }
  };

  useEffect(() => {
    loadAll();
  }, []);

  useEffect(() => {
    const streamUrl = `${API_BASE}/dashboard/stream`;
    const source = new EventSource(streamUrl);

    source.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setSummary(data);
        setStreamStatus("live");
      } catch (error) {
        setStreamStatus("degraded");
      }
    };

    source.onerror = () => {
      setStreamStatus((prev) => (prev === "live" ? "reconnecting" : prev));
    };

    return () => source.close();
  }, []);

  return (
    <div className="app">
      <div className="halo" aria-hidden="true" />
      <header className="hero">
        <div>
          <p className="eyebrow">MessageFlow</p>
          <h1>WhatsApp ops, in real-time.</h1>
          <p className="subtitle">
            A multi-tenant command center for fast replies, high-priority signals, and
            actionable follow-ups.
          </p>
        </div>
        <div className="status-card">
          <p className="status-label">Live Sync</p>
          <div className={`status-pill status-pill--${streamStatus}`}>
            {streamStatus}
          </div>
          <p className="status-note">Stream from /api/v1/dashboard/stream</p>
        </div>
      </header>

      <section className="stats-grid">
        {stats.map((stat) => (
          <article key={stat.label} className="stat-card">
            <p>{stat.label}</p>
            <h2>{stat.value}</h2>
          </article>
        ))}
      </section>

      <section className="grid">
        <article className="panel panel--wide">
          <header>
            <h3>Active Conversations</h3>
            <span>{conversations.length} showing</span>
          </header>
          <div className="panel-body list">
            {conversations.length === 0 && <p className="empty">No conversations yet.</p>}
            {conversations.map((convo) => (
              <div key={convo.id} className="list-row">
                <div>
                  <h4>{convo.contact_name || "Unknown"}</h4>
                  <p>{convo.contact_number}</p>
                </div>
                <span>{formatDate(convo.last_message_at)}</span>
              </div>
            ))}
          </div>
        </article>

        <article className="panel">
          <header>
            <h3>Daily Summary</h3>
            <span>{dailySummary ? formatDate(dailySummary.created_at) : "--"}</span>
          </header>
          <div className="panel-body">
            {dailySummary ? (
              <>
                <p className="summary-text">{dailySummary.summary_text}</p>
                <p className="summary-sub">Conversation ID: {dailySummary.conversation_id}</p>
              </>
            ) : (
              <p className="empty">No summary generated yet.</p>
            )}
          </div>
        </article>

        <article className="panel">
          <header>
            <h3>Important Messages</h3>
            <span>{importantMessages.length} flagged</span>
          </header>
          <div className="panel-body list">
            {importantMessages.length === 0 && <p className="empty">No important messages.</p>}
            {importantMessages.map((item) => (
              <div key={item.id} className="list-row">
                <div>
                  <h4>{item.priority}</h4>
                  <p>{item.content}</p>
                </div>
                <span>{formatDate(item.timestamp)}</span>
              </div>
            ))}
          </div>
        </article>

        <article className="panel">
          <header>
            <h3>Action Items</h3>
            <span>{actionItems.length} tracked</span>
          </header>
          <div className="panel-body list">
            {actionItems.length === 0 && <p className="empty">No action items.</p>}
            {actionItems.map((item) => (
              <div key={item.id} className="list-row">
                <div>
                  <h4>{item.description}</h4>
                  <p>Status: {item.status}</p>
                </div>
                <span>{item.due_date || "--"}</span>
              </div>
            ))}
          </div>
        </article>
      </section>
    </div>
  );
}
