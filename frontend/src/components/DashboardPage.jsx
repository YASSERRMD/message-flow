import { useCallback, useEffect, useMemo, useState } from "react";
import ActionItemsTab from "./ActionItemsTab.jsx";
import ConversationsSidebar from "./ConversationsSidebar.jsx";
import DailySummaryCard from "./DailySummaryCard.jsx";
import ImportantMessagesTab from "./ImportantMessagesTab.jsx";
import MessagesList from "./MessagesList.jsx";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";
const WS_BASE = import.meta.env.VITE_WS_BASE || API_BASE.replace("http", "ws");

const defaultSummary = {
  total_conversations: 0,
  total_messages: 0,
  important_messages: 0,
  open_action_items: 0
};

const useStoredState = (key, initialValue) => {
  const [state, setState] = useState(() => {
    const stored = localStorage.getItem(key);
    return stored ? JSON.parse(stored) : initialValue;
  });

  useEffect(() => {
    localStorage.setItem(key, JSON.stringify(state));
  }, [key, state]);

  return [state, setState];
};

const formatDate = (value) => {
  if (!value) return "--";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "--";
  return date.toLocaleString();
};

export default function DashboardPage() {
  const [theme, setTheme] = useStoredState("mf-theme", "light");
  const [token, setToken] = useStoredState("mf-token", "");
  const [csrf, setCsrf] = useStoredState("mf-csrf", "");
  const [tenantId, setTenantId] = useStoredState("mf-tenant", 1);
  const [user, setUser] = useState(null);

  const [summary, setSummary] = useState(defaultSummary);
  const [dailySummary, setDailySummary] = useState(null);
  const [conversations, setConversations] = useState([]);
  const [importantMessages, setImportantMessages] = useState([]);
  const [actionItems, setActionItems] = useState([]);
  const [messages, setMessages] = useState([]);
  const [messagesPage, setMessagesPage] = useState(1);
  const [hasMoreMessages, setHasMoreMessages] = useState(true);
  const [selectedConversation, setSelectedConversation] = useState(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [status, setStatus] = useState("idle");
  const [authStatus, setAuthStatus] = useState("signed-out");

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  const headers = useMemo(() => {
    const base = {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : ""
    };
    return base;
  }, [token]);

  const authHeaders = useMemo(() => {
    if (!csrf) return headers;
    return {
      ...headers,
      "X-CSRF-Token": csrf
    };
  }, [headers, csrf]);

  const loadDashboard = useCallback(async () => {
    if (!token) return;
    setStatus("loading");
    try {
      const [summaryRes, convoRes, importantRes, actionsRes, dailyRes] = await Promise.all([
        fetch(`${API_BASE}/dashboard`, { headers }),
        fetch(`${API_BASE}/conversations?limit=12`, { headers }),
        fetch(`${API_BASE}/important-messages?limit=8`, { headers }),
        fetch(`${API_BASE}/action-items?limit=12`, { headers }),
        fetch(`${API_BASE}/daily-summary`, { headers })
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
      setStatus("ready");
    } catch (error) {
      setStatus("error");
    }
  }, [headers, token]);

  const loadMessages = useCallback(async (conversationId, page = 1) => {
    if (!token || !conversationId) return;
    const res = await fetch(
      `${API_BASE}/conversations/${conversationId}/messages?page=${page}&limit=20`,
      { headers }
    );
    if (!res.ok) return;
    const data = await res.json();
    const next = data.data || [];
    setMessages((prev) => (page === 1 ? next : [...prev, ...next]));
    setHasMoreMessages(next.length === 20);
    setMessagesPage(page);
  }, [headers, token]);

  const refreshMe = useCallback(async () => {
    if (!token) return;
    const res = await fetch(`${API_BASE}/auth/me`, { headers });
    if (res.ok) {
      const payload = await res.json();
      setUser(payload.user);
      setAuthStatus("signed-in");
    }
  }, [headers, token]);

  useEffect(() => {
    refreshMe();
  }, [refreshMe]);

  useEffect(() => {
    loadDashboard();
  }, [loadDashboard]);

  useEffect(() => {
    if (!token) return;
    const ws = new WebSocket(`${WS_BASE}/ws?token=${encodeURIComponent(token)}`);
    ws.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        if (payload.type?.startsWith("message") || payload.type?.startsWith("action_item")) {
          loadDashboard();
          if (selectedConversation) {
            loadMessages(selectedConversation.id, 1);
          }
        }
      } catch (error) {
        // ignore
      }
    };
    return () => ws.close();
  }, [token, loadDashboard, loadMessages, selectedConversation]);

  useEffect(() => {
    if (!selectedConversation) return;
    loadMessages(selectedConversation.id, 1);
  }, [selectedConversation, loadMessages]);

  const filteredConversations = useMemo(() => {
    if (!searchTerm) return conversations;
    const term = searchTerm.toLowerCase();
    return conversations.filter((convo) => {
      return (
        convo.contact_name?.toLowerCase().includes(term) ||
        convo.contact_number?.toLowerCase().includes(term)
      );
    });
  }, [conversations, searchTerm]);

  const handleLogin = async ({ email, password, tenant }) => {
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password, tenant_id: Number(tenant) })
    });
    if (!res.ok) return;
    const data = await res.json();
    setToken(data.token || "");
    setCsrf(data.csrf || "");
    setTenantId(Number(tenant));
  };

  const handleRegister = async ({ email, password, tenant }) => {
    const res = await fetch(`${API_BASE}/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password, tenant_id: Number(tenant) })
    });
    if (!res.ok) return;
    const data = await res.json();
    setToken(data.token || "");
    setCsrf(data.csrf || "");
    setTenantId(Number(tenant));
  };

  const handleReply = async ({ conversationId, content }) => {
    const res = await fetch(`${API_BASE}/messages/reply`, {
      method: "POST",
      headers: authHeaders,
      body: JSON.stringify({ conversation_id: conversationId, content })
    });
    if (res.ok) {
      await loadMessages(conversationId, 1);
    }
  };

  const handleForward = async ({ messageId, targetConversationId }) => {
    const res = await fetch(`${API_BASE}/messages/forward`, {
      method: "POST",
      headers: authHeaders,
      body: JSON.stringify({ message_id: messageId, target_conversation_id: targetConversationId })
    });
    if (res.ok && selectedConversation) {
      await loadMessages(selectedConversation.id, 1);
    }
  };

  const handleActionCreate = async (payload) => {
    const res = await fetch(`${API_BASE}/action-items`, {
      method: "POST",
      headers: authHeaders,
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      loadDashboard();
    }
  };

  const handleActionUpdate = async (id, payload) => {
    const res = await fetch(`${API_BASE}/action-items/${id}`, {
      method: "PATCH",
      headers: authHeaders,
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      loadDashboard();
    }
  };

  const handleActionDelete = async (id) => {
    const res = await fetch(`${API_BASE}/action-items/${id}`, {
      method: "DELETE",
      headers: authHeaders
    });
    if (res.ok) {
      loadDashboard();
    }
  };

  const stats = useMemo(
    () => [
      { label: "Conversations", value: summary.total_conversations },
      { label: "Messages", value: summary.total_messages },
      { label: "Important", value: summary.important_messages },
      { label: "Open Actions", value: summary.open_action_items }
    ],
    [summary]
  );

  return (
    <div className="app">
      <div className="halo" aria-hidden="true" />
      <header className="hero">
        <div>
          <p className="eyebrow">MessageFlow</p>
          <h1>WhatsApp ops, in real-time.</h1>
          <p className="subtitle">
            Multi-tenant command center for fast replies, high-priority signals, and
            actionable follow-ups.
          </p>
        </div>
        <div className="status-card">
          <p className="status-label">Workspace</p>
          <p className="status-meta">Tenant {tenantId}</p>
          <p className="status-meta">{user ? user.email : "Not signed in"}</p>
          <div className="status-actions">
            <button type="button" onClick={() => setTheme(theme === "light" ? "dark" : "light")}
              className="ghost-button">
              {theme === "light" ? "Dark" : "Light"} mode
            </button>
          </div>
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

      <section className="auth-panel">
        <div>
          <h3>Access</h3>
          <p>{authStatus === "signed-in" ? "Authenticated" : "Sign in or register"}</p>
        </div>
        <div className="auth-actions">
          <AuthForm label="Login" onSubmit={handleLogin} />
          <AuthForm label="Register" onSubmit={handleRegister} />
        </div>
      </section>

      <section className="dashboard">
        <ConversationsSidebar
          conversations={filteredConversations}
          selected={selectedConversation}
          onSelect={setSelectedConversation}
          searchTerm={searchTerm}
          onSearch={setSearchTerm}
        />

        <div className="dashboard-main">
          <DailySummaryCard summary={dailySummary} stats={summary} />
          <MessagesList
            conversation={selectedConversation}
            messages={messages}
            onLoadMore={() => loadMessages(selectedConversation?.id, messagesPage + 1)}
            hasMore={hasMoreMessages}
            onReply={handleReply}
            onForward={handleForward}
            formatDate={formatDate}
          />
          <div className="tabs">
            <ImportantMessagesTab items={importantMessages} formatDate={formatDate} />
            <ActionItemsTab
              items={actionItems}
              conversations={conversations}
              onCreate={handleActionCreate}
              onUpdate={handleActionUpdate}
              onDelete={handleActionDelete}
            />
          </div>
        </div>
      </section>

      <footer className="footer">
        <span>Status: {status}</span>
      </footer>
    </div>
  );
}

function AuthForm({ label, onSubmit }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [tenant, setTenant] = useState(1);

  const handleSubmit = (event) => {
    event.preventDefault();
    onSubmit({ email, password, tenant });
  };

  return (
    <form className="auth-form" onSubmit={handleSubmit}>
      <h4>{label}</h4>
      <input
        type="email"
        placeholder="Email"
        value={email}
        onChange={(event) => setEmail(event.target.value)}
        required
      />
      <input
        type="password"
        placeholder="Password"
        value={password}
        onChange={(event) => setPassword(event.target.value)}
        required
      />
      <input
        type="number"
        min="1"
        placeholder="Tenant ID"
        value={tenant}
        onChange={(event) => setTenant(event.target.value)}
        required
      />
      <button type="submit">{label}</button>
    </form>
  );
}
