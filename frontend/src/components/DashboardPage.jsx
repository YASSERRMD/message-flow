import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import useStoredState from "../hooks/useStoredState.js";
import DailySummaryCard from "./DailySummaryCard";
import ChatMessage from "./chat/ChatMessage.jsx";
import Avatar from "./chat/Avatar.jsx";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";
const WS_BASE = import.meta.env.VITE_WS_BASE || API_BASE.replace("http", "ws");

const defaultSummary = {
  total_conversations: 0,
  total_messages: 0,
  important_messages: 0,
  open_action_items: 0
};

export default function DashboardPage({ onNavigate, searchTerm = "" }) {

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
  const [authStatus, setAuthStatus] = useState("signed-out");
  const [qrSession, setQrSession] = useState("");
  const [qrImage, setQrImage] = useState("");
  const [qrStatus, setQrStatus] = useState("idle");
  const [qrError, setQrError] = useState("");
  const [filter, setFilter] = useState("all");
  // ... state declarations ...

  // ... (lines 38-300 skipped)

  // ... state declarations ...
  const [replyText, setReplyText] = useState("");
  const [showSummary, setShowSummary] = useState(false);
  const [summaryData, setSummaryData] = useState(null);
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [dailySummary, setDailySummary] = useState(null);
  const [loadingMore, setLoadingMore] = useState(false);
  const [showAIChat, setShowAIChat] = useState(false);
  const [aiQuestion, setAiQuestion] = useState("");
  const [aiChatLoading, setAiChatLoading] = useState(false);
  const [aiChatError, setAiChatError] = useState("");
  const [aiTurnsByConversation, setAiTurnsByConversation] = useState({});
  const messagesContainerRef = useRef(null);
  const pendingScrollAdjustRef = useRef(null);
  const shouldStickToBottomRef = useRef(true);

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
    const wsUrl = `${WS_BASE}/ws?token=${token}`;
    const ws = new WebSocket(wsUrl);
    const refreshConversations = async () => {
      try {
        const convRes = await fetch(`${API_BASE}/conversations`, { headers: authHeaders });
        if (convRes.ok) {
          const data = await convRes.json();
          setConversations(data.data || []);
        }
      } catch { }
    };

    const fetchMessage = async (messageId) => {
      const res = await fetch(`${API_BASE}/messages/${messageId}`, { headers: authHeaders });
      if (!res.ok) return null;
      return res.json().catch(() => null);
    };

    const appendMessage = (msg) => {
      if (!msg?.id || !msg?.conversation_id) return;

      // Reset per conversation.
      if (selectedConversation?.id !== msg.conversation_id) return;

      setMessages(prev => {
        if (prev.some(m => m.id === msg.id)) return prev;
        return [...prev, msg];
      });
    };

    ws.onmessage = async (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data?.type !== "message.received" && data?.type !== "message.reply") return;

        const msg = await fetchMessage(data.message_id);
        if (msg) {
          appendMessage(msg);
        }
        await refreshConversations();

        if (data.type === "message.received" && Notification.permission === "granted") {
          new Notification("New WhatsApp Message", {
            body: `New message received`,
            icon: "/logo.svg"
          });
        }
      } catch { }
    };
    return () => ws.close();
  }, [token, authStatus, authHeaders, selectedConversation]);

  const authHeaders = useMemo(() => ({
    "Content-Type": "application/json",
    Authorization: token ? `Bearer ${token}` : "",
    "X-CSRF-Token": csrf || ""
  }), [token, csrf]);

  const loadDashboard = useCallback(async () => {
    if (!token) return;
    try {
      const [convRes, summRes, dailyRes] = await Promise.all([
        fetch(`${API_BASE}/conversations`, { headers: authHeaders }),
        fetch(`${API_BASE}/dashboard`, { headers: authHeaders }),
        fetch(`${API_BASE}/daily-summary`, { headers: authHeaders })
      ]);
      if (convRes.ok) {
        const data = await convRes.json();
        setConversations(data.data || []);
      }
      if (summRes.ok) {
        const data = await summRes.json();
        setSummary(data || defaultSummary);
      }
      if (dailyRes.ok) {
        const data = await dailyRes.json();
        setDailySummary(data);
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
        const container = messagesContainerRef.current;
        if (container) {
          pendingScrollAdjustRef.current = {
            prevScrollHeight: container.scrollHeight,
            prevScrollTop: container.scrollTop
          };
        }
        // Page > 1 is older messages. Prepend so overall order stays chronological.
        setMessages(prev => [...newMessages, ...prev]);
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

  useEffect(() => {
    const container = messagesContainerRef.current;
    if (!container) return;

    if (pendingScrollAdjustRef.current) {
      const { prevScrollHeight, prevScrollTop } = pendingScrollAdjustRef.current;
      const nextScrollHeight = container.scrollHeight;
      const delta = nextScrollHeight - prevScrollHeight;
      container.scrollTop = prevScrollTop + delta;
      pendingScrollAdjustRef.current = null;
      return;
    }

    if (shouldStickToBottomRef.current) {
      container.scrollTop = container.scrollHeight;
    }
  }, [messages.length, selectedConversation?.id]);

  const filteredConversations = useMemo(() => {
    let list = conversations;
    if (filter === "groups") {
      list = list.filter(c =>
        (c.whatsapp_jid || "").includes("@g.us") ||
        (c.contact_number || "").includes("@g.us") ||
        (c.contact_number || "").startsWith("12036")
      );
    }
    if (searchTerm) {
      const term = searchTerm.toLowerCase();
      list = list.filter(c =>
        (c.contact_name || "").toLowerCase().includes(term) ||
        (c.contact_number || "").toLowerCase().includes(term) ||
        (c.whatsapp_jid || "").toLowerCase().includes(term)
      );
    }
    return list;
  }, [conversations, filter, searchTerm]);

  const startWhatsAppConnect = async () => {
    setQrStatus("loading");
    setQrError("");
    try {
      const res = await fetch(`${API_BASE}/auth/whatsapp/qr`, {
        method: "GET",
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
        const res = await fetch(`${API_BASE}/auth/whatsapp/status?session_id=${sessionId}`, { headers: authHeaders });
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
        loadDashboard(); // Refresh sorting
      } else {
        const err = await res.json().catch(() => ({}));
        alert(`Failed to send: ${err.error || `HTTP ${res.status}`}`);
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

  const handleAIChat = async (event) => {
    event?.preventDefault?.();
    if (!selectedConversation) return;

    const question = aiQuestion.trim();
    if (!question || aiChatLoading) return;

    setAiQuestion("");
    setAiChatError("");
    setAiChatLoading(true);

    const conversationId = selectedConversation.id;
    setAiTurnsByConversation(prev => ({
      ...prev,
      [conversationId]: [...(prev[conversationId] || []), { role: "user", content: question }]
    }));

    try {
      const res = await fetch(`${API_BASE}/conversations/${conversationId}/chat`, {
        method: "POST",
        headers: authHeaders,
        body: JSON.stringify({ question, limit: 200 })
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data?.error || `HTTP ${res.status}`);
      }
      const answer = (data?.answer || "").trim() || "No answer returned.";
      setAiTurnsByConversation(prev => ({
        ...prev,
        [conversationId]: [...(prev[conversationId] || []), { role: "assistant", content: answer }]
      }));
    } catch (err) {
      setAiChatError(err?.message || "Failed to chat with AI");
      setAiTurnsByConversation(prev => ({
        ...prev,
        [conversationId]: [...(prev[conversationId] || []), { role: "assistant", content: "AI chat failed. Please try again." }]
      }));
    } finally {
      setAiChatLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      await fetch(`${API_BASE}/auth/logout`, { method: "POST", headers: authHeaders });
    } catch { }
    setToken("");
    setCsrf("");
    setAuthStatus("signed-out");
    setConversations([]);
    setMessages([]);
    setSelectedConversation(null);
  };

  const formatTime = (dateStr) => {
    if (!dateStr) return "";
    const d = new Date(dateStr);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  const selectConversation = (conv) => {
    setSelectedConversation(conv);
    setShowAIChat(false);
    setAiChatError("");
  };

  const conversationAvatarSrc = useCallback((conversationId) => {
    if (!conversationId || !token) return "";
    return `${API_BASE}/conversations/${conversationId}/avatar?token=${encodeURIComponent(token)}`;
  }, [token]);

  const handleMessagesScroll = useCallback((e) => {
    const el = e.currentTarget;
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight;
    shouldStickToBottomRef.current = distanceFromBottom < 60;

    if (loadingMore || !selectedConversation || !hasMoreMessages) return;
    if (e.currentTarget.scrollTop <= 60) {
      setLoadingMore(true);
      Promise.resolve(loadMessages(selectedConversation.id, messagesPage + 1))
        .finally(() => setLoadingMore(false));
    }
  }, [loadingMore, selectedConversation, hasMoreMessages, loadMessages, messagesPage]);

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
    <>
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
              const name = conv.contact_name || (conv.contact_number || "").split("@")[0] || "Unknown";
              const isGroup = conv.is_group ?? ((conv.contact_number || "").includes("@g.us") || (conv.contact_number || "").startsWith("12036"));
              const avatarSrc = conversationAvatarSrc(conv.id);
              return (
                <div
                  key={conv.id}
                  className={`conversation-item ${selectedConversation?.id === conv.id ? "active" : ""}`}
                  onClick={() => selectConversation(conv)}
                >
                  <Avatar className="conv-avatar" src={avatarSrc} name={name} />
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
                  <Avatar
                    className="chat-avatar"
                    src={conversationAvatarSrc(selectedConversation.id)}
                    name={selectedConversation.contact_name || selectedConversation.contact_number}
                  />
                  <div className="chat-details">
                    <h3>{selectedConversation.contact_name || (selectedConversation.contact_number || "").split("@")[0]}</h3>
                    <p>{selectedConversation.contact_number}</p>
                  </div>
                </div>
                <div className="chat-actions">
                  <button className="action-btn" onClick={async () => {
                    const btn = document.getElementById('sync-btn');
                    if (btn) btn.innerText = 'Syncing...';
                    try {
                      await fetch(`${API_BASE}/auth/whatsapp/sync-contacts`, { method: 'POST', headers: authHeaders });
                      await loadDashboard();
                    } catch (e) { }
                    if (btn) btn.innerText = 'Sync Contacts';
                  }} id="sync-btn"><i className="fas fa-sync"></i> Sync Contacts</button>
                  {/* <button className="action-btn"><i className="fas fa-search"></i> Search</button> */}
                  <button className="action-btn" onClick={() => { setAiChatError(""); setShowAIChat(true); }}><i className="fas fa-robot"></i> Ask AI</button>
                  <button className="action-btn primary" onClick={handleSummarize}><i className="fas fa-sparkles"></i> Summarize</button>
                </div>
              </div>

                <div className="messages-container" onScroll={handleMessagesScroll} ref={messagesContainerRef}>
                  <div className="message-group">
                    <div className="message-date">Today</div>
                    {loadingMore && hasMoreMessages && (
                      <div className="message-date">Loading older messages...</div>
                    )}
                    {messages.map((msg) => {
                      const isGroup =
                        (selectedConversation?.whatsapp_jid || "").includes("@g.us") ||
                        (selectedConversation?.contact_number || "").includes("@g.us") ||
                        (selectedConversation?.contact_number || "").startsWith("12036");

                      return (
                        <ChatMessage
                          key={msg.id}
                          message={msg}
                          isGroup={isGroup}
                          formatTime={formatTime}
                        />
                      );
                    })}
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
              <div className="stat-value">{summary.total_conversations ?? 0}</div>
              <div className="stat-label">Total Chats</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">{summary.total_messages ?? 0}</div>
              <div className="stat-label">Messages</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">{summary.open_action_items ?? 0}</div>
              <div className="stat-label">Action Items</div>
            </div>
            <div className="stat-box">
              <div className="stat-value">{summary.important_messages ?? 0}</div>
              <div className="stat-label">Urgent</div>
            </div>
          </div>

          {dailySummary && (
            <div className="info-section">
              <DailySummaryCard summary={dailySummary} stats={summary} />
            </div>
          )}

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

      {/* AI Chat Modal */}
      {showAIChat && selectedConversation && (
        <div className="summary-modal-overlay" onClick={() => setShowAIChat(false)}>
          <div className="summary-modal ai-modal" onClick={(e) => e.stopPropagation()}>
            <div className="summary-header">
              <h3>AI Chat</h3>
              <button className="close-btn" onClick={() => setShowAIChat(false)}>×</button>
            </div>
            <div className="summary-content ai-chat-content">
              {aiChatError && (
                <p className="error" style={{ marginBottom: "12px" }}>{aiChatError}</p>
              )}
              <div className="ai-chat-log">
                {(aiTurnsByConversation[selectedConversation.id] || []).length === 0 ? (
                  <p style={{ color: "#6b7280", fontSize: "14px" }}>
                    Ask a question about this chat history. Example: "What did we decide about the meeting?"
                  </p>
                ) : (
                  (aiTurnsByConversation[selectedConversation.id] || []).map((turn, idx) => (
                    <div key={idx} className={`ai-chat-turn ${turn.role}`}>
                      <div className="ai-chat-bubble">{turn.content}</div>
                    </div>
                  ))
                )}
                {aiChatLoading && (
                  <div className="ai-chat-turn assistant">
                    <div className="ai-chat-bubble">Thinking...</div>
                  </div>
                )}
              </div>

              <form className="ai-chat-form" onSubmit={handleAIChat}>
                <input
                  type="text"
                  className="form-input ai-chat-input"
                  placeholder="Ask AI about this conversation..."
                  value={aiQuestion}
                  onChange={(e) => setAiQuestion(e.target.value)}
                />
                <button className="action-btn primary ai-chat-send" type="submit" disabled={aiChatLoading || !aiQuestion.trim()}>
                  Send
                </button>
              </form>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
