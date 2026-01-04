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
  const [filter, setFilter] = useState("all");
  const [showModal, setShowModal] = useState(false);
  const [activeNav, setActiveNav] = useState("dashboard");

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

  // Request browser notification permission on mount
  useEffect(() => {
    if ("Notification" in window && Notification.permission === "default") {
      Notification.requestPermission();
    }
  }, []);

  // WebSocket for real-time notifications
  useEffect(() => {
    if (!token || authStatus !== "signed-in") return;

    const wsUrl = `${WS_BASE}/ws?tenant_id=${tenantId}`;
    const ws = new WebSocket(wsUrl);

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === "message.received") {
          loadDashboard();
          if (Notification.permission === "granted") {
            const notification = new Notification("New WhatsApp Message", {
              body: `New message in conversation ${data.conversation_id}`,
              icon: "/logo.svg",
              tag: `msg-${data.message_id}`
            });
            notification.onclick = () => {
              window.focus();
              const conv = conversations.find(c => c.id === data.conversation_id);
              if (conv) {
                setSelectedConversation(conv);
                loadMessages(conv.id);
                setShowModal(true);
              }
            };
          }
          // Play notification sound
          try {
            const audio = new Audio("data:audio/wav;base64,UklGRl9vT19...");
            audio.volume = 0.3;
            audio.play().catch(() => { });
          } catch { }
        }
      } catch { }
    };

    return () => ws.close();
  }, [token, authStatus, tenantId, conversations]);

  const authHeaders = useMemo(() => ({
    "Content-Type": "application/json",
    Authorization: token ? `Bearer ${token}` : "",
    "X-CSRF-Token": csrf || ""
  }), [token, csrf]);

  const loadDashboard = useCallback(async () => {
    if (!token) return;
    setStatus("loading");
    try {
      const [convRes, summRes, impRes, actRes, dailyRes] = await Promise.all([
        fetch(`${API_BASE}/conversations`, { headers: authHeaders }),
        fetch(`${API_BASE}/summary`, { headers: authHeaders }),
        fetch(`${API_BASE}/important-messages`, { headers: authHeaders }),
        fetch(`${API_BASE}/action-items`, { headers: authHeaders }),
        fetch(`${API_BASE}/daily-summary`, { headers: authHeaders })
      ]);
      if (convRes.ok) {
        const data = await convRes.json();
        setConversations(data.data || []);
      }
      if (summRes.ok) {
        const data = await summRes.json();
        setSummary(data.data || defaultSummary);
      }
      if (impRes.ok) {
        const data = await impRes.json();
        setImportantMessages(data.data || []);
      }
      if (actRes.ok) {
        const data = await actRes.json();
        setActionItems(data.data || []);
      }
      if (dailyRes.ok) {
        const data = await dailyRes.json();
        setDailySummary(data || null);
      }
    } finally {
      setStatus("idle");
    }
  }, [token, authHeaders]);

  useEffect(() => {
    if (authStatus === "signed-in") {
      loadDashboard();
    }
  }, [authStatus, loadDashboard]);

  const loadMessages = useCallback(async (conversationId, page = 1) => {
    if (!conversationId) return;
    const res = await fetch(`${API_BASE}/conversations/${conversationId}/messages?page=${page}&limit=50`, { headers: authHeaders });
    if (res.ok) {
      const data = await res.json();
      const newMessages = data.data || [];
      if (page === 1) {
        setMessages(newMessages);
      } else {
        setMessages(prev => [...prev, ...newMessages]);
      }
      setMessagesPage(page);
      setHasMoreMessages(newMessages.length === 50);
    }
  }, [authHeaders]);

  useEffect(() => {
    if (selectedConversation) {
      loadMessages(selectedConversation.id, 1);
    } else {
      setMessages([]);
    }
  }, [selectedConversation, loadMessages]);

  const filteredConversations = useMemo(() => {
    let list = conversations;
    if (filter === "groups") {
      list = list.filter(c => c.whatsapp_jid?.includes("@g.us"));
    }
    if (searchTerm) {
      const term = searchTerm.toLowerCase();
      list = list.filter(c =>
        (c.contact_name || "").toLowerCase().includes(term) ||
        (c.whatsapp_jid || "").toLowerCase().includes(term)
      );
    }
    return list;
  }, [conversations, filter, searchTerm]);

  const startWhatsAppConnect = async () => {
    setQrStatus("loading");
    setQrError("");
    try {
      const res = await fetch(`${API_BASE}/whatsapp/start-auth`, {
        method: "POST",
        headers: authHeaders
      });
      if (!res.ok) throw new Error("Failed to start auth");
      const data = await res.json();
      setQrSession(data.session_id || "");
      pollWhatsAppStatus(data.session_id);
    } catch (err) {
      setQrError(err.message);
      setQrStatus("idle");
    }
  };

  const pollWhatsAppStatus = async (sessionId) => {
    if (!sessionId) return;
    let attempts = 0;
    const maxAttempts = 60;
    const poll = async () => {
      if (attempts >= maxAttempts) {
        setQrStatus("idle");
        setQrError("QR code expired. Please try again.");
        return;
      }
      attempts++;
      try {
        const res = await fetch(`${API_BASE}/whatsapp/status?session_id=${sessionId}`, { headers: authHeaders });
        if (res.ok) {
          const data = await res.json();
          if (data.status === "connected") {
            setAuthStatus("signed-in");
            setToken(data.token || token);
            setCsrf(data.csrf || csrf);
            setQrStatus("idle");
            setQrImage("");
            loadDashboard();
            return;
          } else if (data.status === "pending" || data.status === "waiting") {
            setQrStatus("pending");
            if (data.qr_code) {
              setQrImage(data.qr_code);
            }
          } else if (data.status === "expired" || data.status === "failed") {
            setQrStatus("idle");
            setQrError("Session expired or failed. Please try again.");
            return;
          }
        }
      } catch { }
      setTimeout(poll, 2000);
    };
    poll();
  };

  const handleReply = async (text) => {
    if (!selectedConversation || !text.trim()) return;
    try {
      const res = await fetch(`${API_BASE}/conversations/${selectedConversation.id}/messages`, {
        method: "POST",
        headers: authHeaders,
        body: JSON.stringify({ content: text })
      });
      if (!res.ok) {
        alert("Failed to send message. Please try again.");
        return;
      }
      loadMessages(selectedConversation.id, 1);
    } catch (err) {
      alert("Failed to send message: " + err.message);
    }
  };

  const handleForward = async () => { };

  const handleActionCreate = async (payload) => {
    await fetch(`${API_BASE}/action-items`, {
      method: "POST",
      headers: authHeaders,
      body: JSON.stringify(payload)
    });
    loadDashboard();
  };

  const handleActionUpdate = async (id, payload) => {
    await fetch(`${API_BASE}/action-items/${id}`, {
      method: "PATCH",
      headers: authHeaders,
      body: JSON.stringify(payload)
    });
    loadDashboard();
  };

  const handleActionDelete = async (id) => {
    await fetch(`${API_BASE}/action-items/${id}`, {
      method: "DELETE",
      headers: authHeaders
    });
    loadDashboard();
  };

  const handleLogout = () => {
    setToken("");
    setCsrf("");
    setUser(null);
    setAuthStatus("signed-out");
    setQrSession("");
    setQrImage("");
    setQrStatus("idle");
    setConversations([]);
    setMessages([]);
    setSelectedConversation(null);
  };

  const openConversation = (conv) => {
    setSelectedConversation(conv);
    loadMessages(conv.id);
    setShowModal(true);
  };

  const closeModal = () => {
    setShowModal(false);
  };

  const getInitials = (name) => {
    if (!name) return "?";
    return name.split(" ").map(w => w[0]).join("").substring(0, 2).toUpperCase();
  };

  const getAvatarColor = (name) => {
    const colors = [
      { bg: "#f0f9ff", color: "#0369a1" },
      { bg: "#fef3c7", color: "#a16207" },
      { bg: "#f5f3ff", color: "#6b21a8" },
      { bg: "#f0fdf4", color: "#15803d" },
      { bg: "#fdf2f8", color: "#be185d" },
      { bg: "#ecfeff", color: "#0e7490" },
      { bg: "#fff7ed", color: "#c2410c" },
      { bg: "#eff6ff", color: "#1e40af" }
    ];
    const index = (name || "").charCodeAt(0) % colors.length;
    return colors[index];
  };

  return (
    <div className="premium-app">
      {/* Vertical Sidebar */}
      <nav className="vertical-sidebar">
        <div className="brand-mark">
          <img src="/logo.svg" alt="MessageFlow" />
        </div>
        <div className={`nav-icon ${activeNav === "dashboard" ? "active" : ""}`} onClick={() => setActiveNav("dashboard")}>
          <i className="fas fa-chart-pie"></i>
        </div>
        <div className={`nav-icon ${activeNav === "chats" ? "active" : ""}`} onClick={() => setActiveNav("chats")}>
          <i className="fas fa-comments"></i>
        </div>
        <div className={`nav-icon ${activeNav === "analytics" ? "active" : ""}`} onClick={() => setActiveNav("analytics")}>
          <i className="fas fa-chart-line"></i>
        </div>
        <div className={`nav-icon ${activeNav === "settings" ? "active" : ""}`} onClick={() => setActiveNav("settings")}>
          <i className="fas fa-cog"></i>
        </div>
      </nav>

      {/* Main Container */}
      <div className="main-container">
        {/* Header */}
        <header className="top-header">
          <div className="header-left">
            <h1 className="page-heading">MessageFlow</h1>
            {authStatus === "signed-in" && (
              <div className="status-pill">
                <div className="pulse-dot"></div>
                <span>Connected • {conversations.length} chats</span>
              </div>
            )}
          </div>
          <div className="header-right">
            <button className="header-btn"><i className="fas fa-search"></i></button>
            <button className="header-btn"><i className="fas fa-bell"></i></button>
            <button className="header-btn" onClick={() => setTheme(theme === "light" ? "dark" : "light")}>
              {theme === "light" ? <i className="fas fa-moon"></i> : <i className="fas fa-sun"></i>}
            </button>
          </div>
        </header>

        {/* Content */}
        <div className="main-content">
          {authStatus !== "signed-in" ? (
            /* QR Connection Panel */
            <div className="connection-fullscreen">
              <div className="connection-card">
                <h2>Connect WhatsApp</h2>
                <p>Scan the QR code with your WhatsApp mobile app</p>
                <div className="qr-box">
                  {qrStatus === "loading" ? (
                    <div className="qr-loading"><div className="spinner"></div></div>
                  ) : qrImage ? (
                    <img src={qrImage} alt="QR Code" />
                  ) : (
                    <div className="qr-placeholder"><i className="fas fa-qrcode"></i></div>
                  )}
                </div>
                {qrError && <div className="error-msg">{qrError}</div>}
                <button className="btn-primary" onClick={startWhatsAppConnect} disabled={qrStatus === "loading"}>
                  {qrStatus === "loading" ? "Generating..." : "Generate QR Code"}
                </button>
              </div>
            </div>
          ) : (
            <div className="dashboard-grid">
              {/* Metrics Row */}
              <div className="metrics-row">
                <div className="metric-card">
                  <div className="metric-header">
                    <div className="metric-icon" style={{ background: "#f0f9ff", color: "#0369a1" }}>
                      <i className="fas fa-comments"></i>
                    </div>
                    <div className="metric-badge" style={{ background: "#f0f9ff", color: "#0369a1" }}>+12%</div>
                  </div>
                  <div className="metric-value">{summary.total_conversations || conversations.length}</div>
                  <div className="metric-label">Total Chats</div>
                </div>
                <div className="metric-card">
                  <div className="metric-header">
                    <div className="metric-icon" style={{ background: "#f0fdf4", color: "#15803d" }}>
                      <i className="fas fa-envelope"></i>
                    </div>
                    <div className="metric-badge" style={{ background: "#f0fdf4", color: "#15803d" }}>+8%</div>
                  </div>
                  <div className="metric-value">{summary.total_messages || 342}</div>
                  <div className="metric-label">Messages Today</div>
                </div>
                <div className="metric-card">
                  <div className="metric-header">
                    <div className="metric-icon" style={{ background: "#fef3c7", color: "#a16207" }}>
                      <i className="fas fa-star"></i>
                    </div>
                    <div className="metric-badge" style={{ background: "#fef3c7", color: "#a16207" }}>{summary.open_action_items || 12}</div>
                  </div>
                  <div className="metric-value">{summary.open_action_items || 12}</div>
                  <div className="metric-label">Action Items</div>
                </div>
                <div className="metric-card">
                  <div className="metric-header">
                    <div className="metric-icon" style={{ background: "#fef2f2", color: "#b91c1c" }}>
                      <i className="fas fa-fire"></i>
                    </div>
                    <div className="metric-badge" style={{ background: "#fef2f2", color: "#b91c1c" }}>{summary.important_messages || 8}</div>
                  </div>
                  <div className="metric-value">{summary.important_messages || 8}</div>
                  <div className="metric-label">Urgent</div>
                </div>
              </div>

              {/* Main Grid: Conversations + Actions */}
              <div className="content-grid">
                {/* Conversations Panel */}
                <div className="conversations-panel">
                  <div className="panel-header">
                    <h2 className="panel-title">Conversations</h2>
                    <div className="filter-pills">
                      <button className={`pill ${filter === "all" ? "active" : ""}`} onClick={() => setFilter("all")}>All</button>
                      <button className={`pill ${filter === "unread" ? "active" : ""}`} onClick={() => setFilter("unread")}>Unread</button>
                      <button className={`pill ${filter === "groups" ? "active" : ""}`} onClick={() => setFilter("groups")}>Groups</button>
                    </div>
                  </div>
                  <div className="conversations-grid">
                    {filteredConversations.slice(0, 8).map((conv) => {
                      const name = conv.contact_name || conv.whatsapp_jid?.split("@")[0] || "Unknown";
                      const avatar = getAvatarColor(name);
                      const time = conv.last_message_at ? new Date(conv.last_message_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : "";
                      return (
                        <div
                          key={conv.id}
                          className={`conversation-card ${selectedConversation?.id === conv.id ? "active" : ""}`}
                          onClick={() => openConversation(conv)}
                        >
                          <div className="conv-top">
                            <div className="conv-avatar" style={{ background: avatar.bg, color: avatar.color }}>
                              {getInitials(name)}
                            </div>
                            <div className="conv-info">
                              <div className="conv-name">{name}</div>
                              <div className="conv-id">{conv.whatsapp_jid?.split("@")[0] || "status"}</div>
                            </div>
                            <div className="conv-time">{time}</div>
                          </div>
                          <div className="conv-preview">{conv.last_message || "No messages yet..."}</div>
                        </div>
                      );
                    })}
                  </div>
                </div>

                {/* Actions Panel */}
                <div className="actions-panel">
                  <div className="action-box">
                    <h3 className="action-title"><i className="fas fa-bolt"></i> Quick Actions</h3>
                    <div className="action-list">
                      <div className="action-item">
                        <div className="action-item-icon"><i className="fas fa-sparkles"></i></div>
                        <div className="action-item-text">
                          <div className="action-item-title">Summarize All</div>
                          <div className="action-item-desc">AI summary of chats</div>
                        </div>
                      </div>
                      <div className="action-item">
                        <div className="action-item-icon"><i className="fas fa-chart-bar"></i></div>
                        <div className="action-item-text">
                          <div className="action-item-title">Analytics</div>
                          <div className="action-item-desc">View insights</div>
                        </div>
                      </div>
                      <div className="action-item">
                        <div className="action-item-icon"><i className="fas fa-star"></i></div>
                        <div className="action-item-text">
                          <div className="action-item-title">Important</div>
                          <div className="action-item-desc">Flagged messages</div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="action-box create-box">
                    <h3 className="action-title"><i className="fas fa-plus-circle"></i> Create Task</h3>
                    <div className="form-group">
                      <label className="form-label">Description</label>
                      <input type="text" className="form-input" placeholder="Enter task..." />
                    </div>
                    <div className="form-group">
                      <label className="form-label">Category</label>
                      <select className="form-input">
                        <option>Conversation</option>
                        <option>Personal</option>
                        <option>Team</option>
                      </select>
                    </div>
                    <div className="form-group">
                      <label className="form-label">Due Date</label>
                      <input type="date" className="form-input" />
                    </div>
                    <button className="submit-button">Create Task</button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Chat Modal */}
      {showModal && selectedConversation && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal-container" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div className="modal-user">
                <div className="modal-user-avatar" style={getAvatarColor(selectedConversation.contact_name || "")}>
                  {getInitials(selectedConversation.contact_name || selectedConversation.whatsapp_jid)}
                </div>
                <div className="modal-user-info">
                  <h2>{selectedConversation.contact_name || selectedConversation.whatsapp_jid?.split("@")[0]}</h2>
                  <p>{selectedConversation.whatsapp_jid}</p>
                </div>
              </div>
              <button className="modal-close" onClick={closeModal}><i className="fas fa-times"></i></button>
            </div>
            <div className="modal-messages">
              {messages.map((msg) => (
                <div key={msg.id} className={`message-item ${msg.is_outbound ? "outbound" : ""}`}>
                  <div className="message-bubble">
                    <div className="message-text">{msg.content}</div>
                    <div className="message-timestamp">{new Date(msg.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</div>
                  </div>
                </div>
              ))}
            </div>
            <div className="modal-input">
              <div className="input-row">
                <input
                  type="text"
                  className="message-field"
                  placeholder="Type your message..."
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && e.target.value.trim()) {
                      handleReply(e.target.value);
                      e.target.value = "";
                    }
                  }}
                />
                <button className="send-btn"><i className="fas fa-paper-plane"></i></button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
