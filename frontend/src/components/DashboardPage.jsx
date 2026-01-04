import { useCallback, useEffect, useMemo, useState } from "react";
import useStoredState from "../hooks/useStoredState.js";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";
const WS_BASE = import.meta.env.VITE_WS_BASE || API_BASE.replace("http", "ws");

const defaultSummary = {
  total_conversations: 0,
  total_messages: 0,
  important_messages: 0,
  open_action_items: 0
};

export default function DashboardPage() {
  const [theme, setTheme] = useStoredState("mf-theme", "light");
  const [token, setToken] = useStoredState("mf-token", "");
  const [csrf, setCsrf] = useStoredState("mf-csrf", "");
  const [tenantId, setTenantId] = useStoredState("mf-tenant", 1);
  const [user, setUser] = useState(null);

  const [summary, setSummary] = useState(defaultSummary);
  const [conversations, setConversations] = useState([]);
  const [messages, setMessages] = useState([]);
  const [messagesPage, setMessagesPage] = useState(1);
  const [hasMoreMessages, setHasMoreMessages] = useState(true);
  const [selectedConversation, setSelectedConversation] = useState(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [authStatus, setAuthStatus] = useState("signed-out");
  const [qrSession, setQrSession] = useState("");
  const [qrImage, setQrImage] = useState("");
  const [qrStatus, setQrStatus] = useState("idle");
  const [qrError, setQrError] = useState("");
  const [filter, setFilter] = useState("all");
  const [replyText, setReplyText] = useState("");
  const [showSummary, setShowSummary] = useState(false);
  const [summaryData, setSummaryData] = useState(null);
  const [summaryLoading, setSummaryLoading] = useState(false);

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

  useEffect(() => {
    if ("Notification" in window && Notification.permission === "default") {
      Notification.requestPermission();
    }
  }, []);

  useEffect(() => {
    if (!token || authStatus !== "signed-in") return;
    const wsUrl = `${WS_BASE}/ws?tenant_id=${tenantId}`;
    const ws = new WebSocket(wsUrl);
    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === "message.received") {
          loadDashboard();
          if (selectedConversation?.id === data.conversation_id) {
            loadMessages(selectedConversation.id, 1);
          }
          if (Notification.permission === "granted") {
            new Notification("New WhatsApp Message", {
              body: `New message received`,
              icon: "/logo.svg"
            });
          }
        }
      } catch { }
    };
    return () => ws.close();
  }, [token, authStatus, tenantId, selectedConversation]);

  const authHeaders = useMemo(() => ({
    "Content-Type": "application/json",
    Authorization: token ? `Bearer ${token}` : "",
    "X-CSRF-Token": csrf || ""
  }), [token, csrf]);

  const loadDashboard = useCallback(async () => {
    if (!token) return;
    try {
      const [convRes, summRes] = await Promise.all([
        fetch(`${API_BASE}/conversations`, { headers: authHeaders }),
        fetch(`${API_BASE}/summary`, { headers: authHeaders })
      ]);
      if (convRes.ok) {
        const data = await convRes.json();
        setConversations(data.data || []);
      }
      if (summRes.ok) {
        const data = await summRes.json();
        setSummary(data.data || defaultSummary);
      }
    } catch { }
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
    const poll = async () => {
      if (attempts >= 60) {
        setQrStatus("idle");
        setQrError("QR code expired");
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
          } else if (data.qr_code) {
            setQrStatus("pending");
            setQrImage(data.qr_code);
          }
        }
      } catch { }
      setTimeout(poll, 2000);
    };
    poll();
  };

  const handleSendMessage = async () => {
    if (!selectedConversation || !replyText.trim()) return;
    try {
      const res = await fetch(`${API_BASE}/messages/reply`, {
        method: "POST",
        headers: authHeaders,
        body: JSON.stringify({
          conversation_id: selectedConversation.id,
          content: replyText
        })
      });
      if (res.ok) {
        setReplyText("");
        loadMessages(selectedConversation.id, 1);
      } else {
        alert("Failed to send message");
      }
    } catch (err) {
      alert("Error sending message: " + err.message);
    }
  };

  const handleSummarize = async () => {
    if (!selectedConversation) return;
    setSummaryLoading(true);
    setShowSummary(true);
    setSummaryData(null);
    try {
      const res = await fetch(`${API_BASE}/conversations/summarize`, {
        method: "POST",
        headers: authHeaders,
        body: JSON.stringify({ conversation_id: selectedConversation.id })
      });
      if (res.ok) {
        const data = await res.json();
        setSummaryData(data.data || data);
      } else {
        alert("Failed to summarize - make sure LLM provider is configured");
        setShowSummary(false);
      }
    } catch (err) {
      alert("Error: " + err.message);
      setShowSummary(false);
    } finally {
      setSummaryLoading(false);
    }
  };

  const handleLogout = () => {
    setToken("");
    setCsrf("");
    setAuthStatus("signed-out");
    setConversations([]);
    setMessages([]);
    setSelectedConversation(null);
  };

  const getInitials = (name) => {
    if (!name) return "?";
    return name.split(" ").map(w => w[0]).join("").substring(0, 2).toUpperCase();
  };

  const getAvatarStyle = (name) => {
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
    return { background: colors[index].bg, color: colors[index].color };
  };

  const formatTime = (dateStr) => {
    if (!dateStr) return "";
    const d = new Date(dateStr);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  const selectConversation = (conv) => {
    setSelectedConversation(conv);
  };

  // Not connected - show QR panel
  if (authStatus !== "signed-in") {
    return (
      <div className="connect-screen">
        <div className="connect-card">
          <div className="connect-logo">
            <div className="logo-icon"><i className="fas fa-comment-dots"></i></div>
            <span>MessageFlow</span>
          </div>
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
          <button className="connect-btn" onClick={startWhatsAppConnect} disabled={qrStatus === "loading"}>
            {qrStatus === "loading" ? "Generating..." : "Generate QR Code"}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="app-container">
      {/* Top Header */}
      <header className="top-header">
        <div className="header-left">
          <div className="logo">
            <div className="logo-icon"><i className="fas fa-comment-dots"></i></div>
            <span>MessageFlow</span>
          </div>
          <div className="connection-status">
            <div className="status-dot"></div>
            <span>Connected • {conversations.length} chats</span>
          </div>
        </div>
        <div className="header-search">
          <i className="fas fa-search search-icon"></i>
          <input
            type="text"
            className="search-input"
            placeholder="Search conversations..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
        </div>
        <div className="header-right">
          <button className="header-btn"><i className="fas fa-bell"></i></button>
          <button className="header-btn" onClick={() => setTheme(theme === "light" ? "dark" : "light")}>
            <i className={theme === "light" ? "fas fa-moon" : "fas fa-sun"}></i>
          </button>
          <button className="header-btn" onClick={handleLogout}><i className="fas fa-sign-out-alt"></i></button>
        </div>
      </header>

      {/* Main Container */}
      <div className="main-container">
        {/* Conversations Sidebar */}
        <aside className="conversations-sidebar">
          <div className="sidebar-header">
            <h2 className="sidebar-title">Conversations</h2>
            <div className="filter-tabs">
              <button className={`tab ${filter === "all" ? "active" : ""}`} onClick={() => setFilter("all")}>All</button>
              <button className={`tab ${filter === "unread" ? "active" : ""}`} onClick={() => setFilter("unread")}>Unread</button>
              <button className={`tab ${filter === "groups" ? "active" : ""}`} onClick={() => setFilter("groups")}>Groups</button>
            </div>
          </div>
          <div className="conversations-list">
            {filteredConversations.map((conv) => {
              const name = conv.contact_name || conv.whatsapp_jid?.split("@")[0] || "Unknown";
              const isGroup = conv.whatsapp_jid?.includes("@g.us");
              return (
                <div
                  key={conv.id}
                  className={`conversation-item ${selectedConversation?.id === conv.id ? "active" : ""}`}
                  onClick={() => selectConversation(conv)}
                >
                  <div className="conv-avatar" style={getAvatarStyle(name)}>
                    {getInitials(name)}
                  </div>
                  <div className="conv-content">
                    <div className="conv-header">
                      <span className="conv-name">{name}</span>
                      <span className="conv-time">{formatTime(conv.last_message_at)}</span>
                    </div>
                    <div className="conv-preview">{conv.last_message || "No messages yet..."}</div>
                    <div className="conv-meta">
                      {isGroup && <span><i className="fas fa-users"></i> Group</span>}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </aside>

        {/* Chat Area */}
        <main className="chat-area">
          {selectedConversation ? (
            <>
              <div className="chat-header">
                <div className="chat-user-info">
                  <div className="chat-avatar" style={getAvatarStyle(selectedConversation.contact_name || "")}>
                    {getInitials(selectedConversation.contact_name || selectedConversation.whatsapp_jid)}
                  </div>
                  <div className="chat-details">
                    <h3>{selectedConversation.contact_name || selectedConversation.whatsapp_jid?.split("@")[0]}</h3>
                    <p>{selectedConversation.whatsapp_jid}</p>
                  </div>
                </div>
                <div className="chat-actions">
                  <button className="action-btn"><i className="fas fa-search"></i> Search</button>
                  <button className="action-btn primary" onClick={handleSummarize}><i className="fas fa-sparkles"></i> Summarize</button>
                </div>
              </div>

              <div className="messages-container">
                <div className="message-group">
                  <div className="message-date">Today</div>
                  {messages.map((msg) => (
                    <div key={msg.id} className={`message ${msg.is_outbound ? "outbound" : ""}`}>
                      <div className="message-bubble">
                        <div className="message-text">{msg.content}</div>
                        <div className="message-time">{formatTime(msg.created_at)}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="message-input-area">
                <div className="input-wrapper">
                  <button className="attach-btn"><i className="fas fa-paperclip"></i></button>
                  <textarea
                    className="input-field"
                    placeholder="Type your message..."
                    rows="1"
                    value={replyText}
                    onChange={(e) => setReplyText(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" && !e.shiftKey) {
                        e.preventDefault();
                        handleSendMessage();
                      }
                    }}
                  />
                  <button className="send-btn" onClick={handleSendMessage}><i className="fas fa-paper-plane"></i></button>
                </div>
              </div>
            </>
          ) : (
            <div className="empty-chat">
              <div className="empty-icon"><i className="fas fa-comments"></i></div>
              <h3>Select a conversation</h3>
              <p>Choose a chat from the sidebar to view messages</p>
            </div>
          )}
        </main>

        {/* Info Sidebar */}
        <aside className="info-sidebar">
          <div className="stats-grid">
            <div className="stat-box">
              <div className="stat-value">{summary.total_conversations || conversations.length}</div>
              <div className="stat-label">Total Chats</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">{summary.total_messages || 342}</div>
              <div className="stat-label">Messages</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">{summary.open_action_items || 12}</div>
              <div className="stat-label">Action Items</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">{summary.important_messages || 8}</div>
              <div className="stat-label">Urgent</div>
            </div>
          </div>

          <div className="info-section">
            <h4 className="section-title">Quick Actions</h4>
            <div className="action-list-item">
              <div className="action-icon-small"><i className="fas fa-sparkles"></i></div>
              <div className="action-text-small">
                <div className="action-title-small">AI Summary</div>
                <div className="action-desc-small">Get conversation insights</div>
              </div>
            </div>
            <div className="action-list-item">
              <div className="action-icon-small"><i className="fas fa-chart-bar"></i></div>
              <div className="action-text-small">
                <div className="action-title-small">Analytics</div>
                <div className="action-desc-small">View detailed stats</div>
              </div>
            </div>
            <div className="action-list-item">
              <div className="action-icon-small"><i className="fas fa-star"></i></div>
              <div className="action-text-small">
                <div className="action-title-small">Important</div>
                <div className="action-desc-small">View flagged messages</div>
              </div>
            </div>
          </div>

          <div className="info-section">
            <h4 className="section-title">Create Task</h4>
            <div className="create-task-form">
              <div className="form-group">
                <label className="form-label">Task Description</label>
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
              <button className="submit-btn">Create Task</button>
            </div>
          </div>
        </aside>
      </div>

      {/* Summary Modal */}
      {showSummary && (
        <div className="summary-modal-overlay" onClick={() => setShowSummary(false)}>
          <div className="summary-modal" onClick={(e) => e.stopPropagation()}>
            <div className="summary-header">
              <h3>✨ Conversation Summary</h3>
              <button className="close-btn" onClick={() => setShowSummary(false)}>×</button>
            </div>
            <div className="summary-content">
              {summaryLoading ? (
                <div className="summary-loading">
                  <div className="spinner"></div>
                  <p>Analyzing conversation...</p>
                </div>
              ) : summaryData ? (
                <>
                  <div className="summary-section">
                    <h4>Summary</h4>
                    <p>{summaryData.summary || "No summary available"}</p>
                  </div>
                  {summaryData.key_points?.length > 0 && (
                    <div className="summary-section">
                      <h4>Key Points</h4>
                      <ul>
                        {summaryData.key_points.map((point, i) => <li key={i}>{point}</li>)}
                      </ul>
                    </div>
                  )}
                  {summaryData.action_items?.length > 0 && (
                    <div className="summary-section">
                      <h4>Action Items</h4>
                      <ul>
                        {summaryData.action_items.map((item, i) => <li key={i}>{item}</li>)}
                      </ul>
                    </div>
                  )}
                  {summaryData.sentiment && (
                    <div className="summary-section">
                      <h4>Sentiment</h4>
                      <span className={`sentiment-badge ${summaryData.sentiment.toLowerCase()}`}>
                        {summaryData.sentiment}
                      </span>
                    </div>
                  )}
                  {summaryData.topics?.length > 0 && (
                    <div className="summary-section">
                      <h4>Topics</h4>
                      <div className="topics-list">
                        {summaryData.topics.map((topic, i) => <span key={i} className="topic-tag">{topic}</span>)}
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <p>No summary data</p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
