import { useCallback, useEffect, useMemo, useState } from "react";
import ActionItemsTab from "./ActionItemsTab.jsx";
import ConversationsSidebar from "./ConversationsSidebar.jsx";
import DailySummaryCard from "./DailySummaryCard.jsx";
import ImportantMessagesTab from "./ImportantMessagesTab.jsx";
import MessagesList from "./MessagesList.jsx";
import useStoredState from "../hooks/useStoredState.js";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";
const WS_BASE = import.meta.env.VITE_WS_BASE || API_BASE.replace("http", "ws");

const defaultSummary = {
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
  const [qrSession, setQrSession] = useState("");
  const [qrImage, setQrImage] = useState("");
  const [qrTimeout, setQrTimeout] = useState(0);
  const [qrStatus, setQrStatus] = useState("idle");
  const [qrError, setQrError] = useState("");

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  useEffect(() => {
    if (token) {
      setAuthStatus("signed-in");
    } else {
      setAuthStatus("signed-out");
    }
  }, [token]);

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

  const startWhatsAppConnect = useCallback(async () => {
    setQrStatus("loading");
    setQrError("");
    const res = await fetch(`${API_BASE}/auth/whatsapp/qr`, { method: "GET" });
    if (!res.ok) {
      setQrStatus("error");
      setQrError("Failed to generate QR code");
      return;
    }
    const data = await res.json();
    setQrSession(data.session_id || "");
    setQrImage(data.qr_code || "");
    setQrTimeout(data.timeout_seconds || 0);
    setQrStatus(data.status || "pending");
  }, []);

  const pollWhatsAppStatus = useCallback(async () => {
    if (!qrSession) return;
    const res = await fetch(`${API_BASE}/auth/whatsapp/status?session_id=${encodeURIComponent(qrSession)}`, {
      method: "GET"
    });
    if (!res.ok) {
      setQrStatus("error");
      setQrError("Unable to check status");
      return;
    }
    const data = await res.json();
    if (data.status === "connected" && data.token) {
      setToken(data.token || "");
      setCsrf(data.csrf || "");
      setTenantId(Number(data.tenant_id || tenantId));
      setAuthStatus("signed-in");
      setQrStatus("connected");
      setQrError("");
      return;
    }
    if (data.qr_code) {
      setQrImage(data.qr_code);
    }
    setQrStatus(data.status || "pending");
    if (data.error) {
      setQrError(data.error);
    }
  }, [qrSession, tenantId, setToken, setCsrf, setTenantId]);

  useEffect(() => {
    if (!qrSession || authStatus === "signed-in") return;
    const timer = setInterval(() => {
      pollWhatsAppStatus();
    }, 3000);
    return () => clearInterval(timer);
  }, [qrSession, pollWhatsAppStatus, authStatus]);

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

  return (
    <div className="app">
      <div className="wa-layout">
        <aside className="wa-sidebar">
          <div className="wa-sidebar-top">
            <div className="wa-workspace">
              <div>
                <p className="wa-label">Workspace</p>
                <h2>Tenant {tenantId}</h2>
                <div className="wa-meta">{user ? user.email : "Not signed in"}</div>
              </div>
              <button
                type="button"
                onClick={() => setTheme(theme === "light" ? "dark" : "light")}
                className="ghost-button"
              >
                {theme === "light" ? "Dark" : "Light"} mode
              </button>
            </div>
            <div className={`wa-connect ${authStatus === "signed-in" ? "is-connected" : ""}`}>
              <div className="wa-connect-header">
                <strong>{authStatus === "signed-in" ? "WhatsApp Connected" : "Connect WhatsApp"}</strong>
                <span className="wa-meta">{authStatus === "signed-in" ? "Live sync" : "QR pairing"}</span>
              </div>
              <div className="wa-connect-qr">
                {qrImage ? (
                  <img src={qrImage} alt="WhatsApp QR code" />
                ) : (
                  <span className="wa-meta">Generate QR to connect</span>
                )}
              </div>
              <div className="wa-connect-actions">
                <button className="primary" type="button" onClick={startWhatsAppConnect}>
                  {qrStatus === "loading" ? "Generating…" : "Generate QR"}
                </button>
                <span className="wa-connect-status">
                  {qrTimeout ? `Refreshes in ${qrTimeout}s` : `Status: ${qrStatus}`}
                </span>
              </div>
              {qrError && <p className="error-text">{qrError}</p>}
            </div>
          </div>

          <ConversationsSidebar
            conversations={filteredConversations}
            selected={selectedConversation}
            onSelect={setSelectedConversation}
            searchTerm={searchTerm}
            onSearch={setSearchTerm}
          />
        </aside>

        <main className="wa-chat">
          <MessagesList
            conversation={selectedConversation}
            messages={messages}
            onLoadMore={() => loadMessages(selectedConversation?.id, messagesPage + 1)}
            hasMore={hasMoreMessages}
            onReply={handleReply}
            onForward={handleForward}
            formatDate={formatDate}
          />
        </main>

        <aside className="wa-ai">
          <DailySummaryCard summary={dailySummary} stats={summary} />
          <ImportantMessagesTab items={importantMessages} formatDate={formatDate} />
          <ActionItemsTab
            items={actionItems}
            conversations={conversations}
            onCreate={handleActionCreate}
            onUpdate={handleActionUpdate}
            onDelete={handleActionDelete}
          />
        </aside>
      </div>
    </div>
  );
}
