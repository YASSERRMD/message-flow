import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import useStoredState from "../hooks/useStoredState.js";
import DailySummaryCard from "./DailySummaryCard";
import ChatMessage from "./chat/ChatMessage.jsx";
import Avatar from "./chat/Avatar.jsx";
import MediaViewerModal from "./chat/MediaViewerModal.jsx";
import { buildMediaItem, getMediaLabel } from "./chat/messageUtils.js";
import { getApiBase, getWsBase } from "../lib/runtimeConfig.js";

const API_BASE = getApiBase();
const WS_BASE = getWsBase(API_BASE);

const defaultSummary = {
  total_conversations: 0,
  total_messages: 0,
  important_messages: 0,
  open_action_items: 0
};

export default function DashboardPage({ onNavigate, searchTerm = "", onMetaChange }) {

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
  const [selectedConversationId, setSelectedConversationId] = useState(null);
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
  const [syncingContacts, setSyncingContacts] = useState(false);
  const [taskDescription, setTaskDescription] = useState("");
  const [taskDueDate, setTaskDueDate] = useState("");
  const [taskSaving, setTaskSaving] = useState(false);
  const [mediaFilter, setMediaFilter] = useState("all");
  const [activeMediaMessageId, setActiveMediaMessageId] = useState(null);
  const [highlightedMessageId, setHighlightedMessageId] = useState(null);
  const messagesContainerRef = useRef(null);
  const pendingScrollAdjustRef = useRef(null);
  const shouldStickToBottomRef = useRef(true);
  const highlightTimerRef = useRef(null);
  const selectedConversation = useMemo(
    () => conversations.find((conversation) => conversation.id === selectedConversationId) || null,
    [conversations, selectedConversationId]
  );

  const sortConversations = useCallback((list) => {
    const copy = [...(list || [])];
    copy.sort((a, b) => {
      const at = a?.last_message_at ? new Date(a.last_message_at).getTime() : -Infinity;
      const bt = b?.last_message_at ? new Date(b.last_message_at).getTime() : -Infinity;
      if (at !== bt) return bt - at;
      const ac = a?.created_at ? new Date(a.created_at).getTime() : -Infinity;
      const bc = b?.created_at ? new Date(b.created_at).getTime() : -Infinity;
      return bc - ac;
    });
    return copy;
  }, []);

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

  useEffect(() => () => {
    if (highlightTimerRef.current) {
      window.clearTimeout(highlightTimerRef.current);
    }
  }, []);

  const authHeaders = useMemo(() => ({
    "Content-Type": "application/json",
    Authorization: token ? `Bearer ${token}` : "",
    "X-CSRF-Token": csrf || ""
  }), [token, csrf]);

  const markConversationRead = useCallback(async (conversationId) => {
    if (!conversationId) return;
    try {
      await fetch(`${API_BASE}/conversations/${conversationId}/read`, {
        method: "POST",
        headers: authHeaders
      });
    } catch { }
    setConversations((prev) =>
      prev.map((c) => (c.id === conversationId ? { ...c, unread_count: 0 } : c))
    );
  }, [authHeaders]);

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
        setConversations(sortConversations(data.data || []));
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
  }, [token, authHeaders, sortConversations]);

  useEffect(() => {
    if (!token || authStatus !== "signed-in") return;
    const wsUrl = `${WS_BASE}/ws?token=${token}`;
    let ws = null;
    let cancelled = false;
    let retryTimer = null;
    let backoffMs = 750;

    const fetchMessage = async (messageId) => {
      const res = await fetch(`${API_BASE}/messages/${messageId}`, { headers: authHeaders });
      if (!res.ok) return null;
      return res.json().catch(() => null);
    };

    const appendMessage = (msg) => {
      if (!msg?.id || !msg?.conversation_id) return;

      if (selectedConversationId !== msg.conversation_id) return;

      setMessages(prev => {
        if (prev.some(m => m.id === msg.id)) return prev;
        return [...prev, msg];
      });
    };

    const applyConversationUpdate = (update) => {
      if (!update?.id) return;
      let found = false;
      setConversations((prev) => {
        found = prev.some((c) => c.id === update.id);
        if (!found) return prev;
        return sortConversations(prev.map((c) => (c.id === update.id ? { ...c, ...update } : c)));
      });
      // WS conversation payloads may be partial (e.g. only last_message/unread_count).
      // If we don't know this conversation yet, refresh the list to pull full metadata (name/avatar, etc).
      if (!found) {
        loadDashboard();
      }
    };

    const onMessage = async (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data?.type === "auth.logout") {
          setToken("");
          setCsrf("");
          setAuthStatus("signed-out");
          setConversations([]);
          setMessages([]);
          setSelectedConversationId(null);
          return;
        }

        if (data?.type !== "message.received" && data?.type !== "message.reply" && data?.type !== "message.forward") return;

        const msg = data?.message || await fetchMessage(data.message_id);
        if (msg) {
          appendMessage(msg);
          if (data.type === "message.received" && selectedConversationId === msg.conversation_id) {
            await markConversationRead(msg.conversation_id);
          }
        }

        if (data?.conversation) {
          applyConversationUpdate(data.conversation);
        } else if (msg?.conversation_id) {
          // Best-effort update when older backend doesn't send convo payload.
          applyConversationUpdate({
            id: msg.conversation_id,
            last_message: msg.content,
            last_message_at: msg.timestamp
          });
        }

        if (data.type === "message.received" && Notification.permission === "granted") {
          new Notification("New WhatsApp Message", {
            body: `New message received`,
            icon: "/logo.svg"
          });
        }
      } catch { }
    };

    const scheduleReconnect = () => {
      if (cancelled) return;
      if (retryTimer) return;
      retryTimer = window.setTimeout(() => {
        retryTimer = null;
        connect();
      }, backoffMs);
      backoffMs = Math.min(Math.floor(backoffMs * 1.6), 8000);
    };

    const connect = () => {
      if (cancelled) return;
      try {
        ws = new WebSocket(wsUrl);
      } catch {
        scheduleReconnect();
        return;
      }

      ws.onopen = () => {
        backoffMs = 750;
      };
      ws.onmessage = onMessage;
      ws.onerror = () => {
        // Some browsers won't always emit onclose after onerror; force close to trigger reconnect.
        try { ws?.close(); } catch { }
      };
      ws.onclose = () => {
        scheduleReconnect();
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (retryTimer) {
        window.clearTimeout(retryTimer);
      }
      try { ws?.close(); } catch { }
    };
  }, [token, authStatus, authHeaders, selectedConversationId, markConversationRead, sortConversations, loadDashboard]);

  useEffect(() => {
    if (authStatus === "signed-in") {
      loadDashboard();
    }
  }, [authStatus, loadDashboard]);

  // Backstop refresh in case the websocket misses updates (mobile networks, browser sleep, etc).
  useEffect(() => {
    if (authStatus !== "signed-in") return;
    const timer = window.setInterval(() => {
      loadDashboard();
    }, 15000);
    return () => window.clearInterval(timer);
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
      markConversationRead(selectedConversation.id);
    } else {
      setMessages([]);
    }
  }, [selectedConversation, loadMessages, markConversationRead]);

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
    } else if (filter === "unread") {
      list = list.filter(c => (c.unread_count || 0) > 0);
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

  const unreadCount = useMemo(
    () => conversations.reduce((total, conversation) => total + (conversation.unread_count || 0), 0),
    [conversations]
  );

  useEffect(() => {
    if (typeof onMetaChange !== "function") return;
    onMetaChange({
      connected: authStatus === "signed-in",
      conversationsCount: conversations.length,
      unreadCount
    });
  }, [authStatus, conversations.length, unreadCount, onMetaChange]);

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
    const conversationId = selectedConversation.id;
    const content = replyText.trim();
    try {
      const res = await fetch(`${API_BASE}/messages/reply`, {
        method: "POST",
        headers: authHeaders,
        body: JSON.stringify({
          conversation_id: conversationId,
          content
        })
      });

      if (res.ok) {
        const msg = await res.json().catch(() => null);
        setReplyText("");
        if (msg && msg.conversation_id === conversationId) {
          shouldStickToBottomRef.current = true;
          setMessages((prev) => {
            if (prev.some((m) => m.id === msg.id)) return prev;
            return [...prev, msg];
          });
          setConversations((prev) => sortConversations(
            prev.map((c) =>
              c.id === conversationId
                ? {
                  ...c,
                  last_message: msg.content || c.last_message,
                  last_message_at: msg.timestamp || c.last_message_at
                }
                : c
            )
          ));
        }
      } else {
        const err = await res.json().catch(() => ({}));
        alert(`Failed to send: ${err.error || `HTTP ${res.status}`}`);
      }
    } catch (err) {
      alert("Error sending message: " + err.message);
    }
  };

  const handleSyncContacts = async () => {
    setSyncingContacts(true);
    try {
      await fetch(`${API_BASE}/auth/whatsapp/sync-contacts`, {
        method: "POST",
        headers: authHeaders
      });
      await loadDashboard();
      if (selectedConversation) {
        await loadMessages(selectedConversation.id, 1);
      }
    } catch {
      alert("Failed to sync contacts");
    } finally {
      setSyncingContacts(false);
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
        const err = await res.json().catch(() => ({}));
        alert(err?.error || "Failed to summarize - make sure LLM provider is configured");
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
    setSelectedConversationId(null);
  };

  const handleCreateTask = async (event) => {
    event.preventDefault();
    if (!selectedConversation || !taskDescription.trim() || taskSaving) return;
    setTaskSaving(true);
    try {
      const res = await fetch(`${API_BASE}/action-items`, {
        method: "POST",
        headers: authHeaders,
        body: JSON.stringify({
          conversation_id: selectedConversation.id,
          description: taskDescription.trim(),
          status: "new",
          due_date: taskDueDate || null
        })
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data?.error || `HTTP ${res.status}`);
      }
      setTaskDescription("");
      setTaskDueDate("");
    } catch (err) {
      alert(err?.message || "Failed to create task");
    } finally {
      setTaskSaving(false);
    }
  };

  const formatTime = (dateStr) => {
    if (!dateStr) return "";
    const d = new Date(dateStr);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  const formatConversationTime = (dateStr) => {
    if (!dateStr) return "";
    const date = new Date(dateStr);
    if (Number.isNaN(date.getTime())) return "";
    const now = new Date();
    if (date.toDateString() === now.toDateString()) {
      return formatTime(dateStr);
    }
    const diffDays = Math.floor((now - date) / (1000 * 60 * 60 * 24));
    if (diffDays < 6) {
      return date.toLocaleDateString([], { weekday: "short" });
    }
    return date.toLocaleDateString([], { month: "short", day: "numeric" });
  };

  const formatDateTimeLabel = (dateStr) => {
    if (!dateStr) return "No recent activity";
    const date = new Date(dateStr);
    if (Number.isNaN(date.getTime())) return "No recent activity";
    return date.toLocaleString([], {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit"
    });
  };

  const toDayKey = (dateStr) => {
    if (!dateStr) return "unknown";
    const d = new Date(dateStr);
    if (Number.isNaN(d.getTime())) return "unknown";
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    return `${y}-${m}-${day}`;
  };

  const formatDayLabel = (dateStr) => {
    const d = new Date(dateStr);
    if (Number.isNaN(d.getTime())) return "";
    const today = new Date();
    const todayKey = toDayKey(today.toISOString());
    const y = new Date(today);
    y.setDate(today.getDate() - 1);
    const yesterdayKey = toDayKey(y.toISOString());
    const key = toDayKey(d.toISOString());
    if (key === todayKey) return "Today";
    if (key === yesterdayKey) return "Yesterday";
    return d.toLocaleDateString([], { weekday: "short", month: "short", day: "numeric" });
  };

  const isGroupConversation = useMemo(() => {
    if (!selectedConversation) return false;
    return (
      (selectedConversation.whatsapp_jid || "").includes("@g.us") ||
      (selectedConversation.contact_number || "").includes("@g.us") ||
      (selectedConversation.contact_number || "").startsWith("12036")
    );
  }, [selectedConversation]);

  const selectedConversationName = selectedConversation
    ? (selectedConversation.contact_name || (selectedConversation.contact_number || "").split("@")[0] || "Unknown")
    : "";
  const selectedConversationAddress = selectedConversation?.contact_number || "Unknown contact";
  const selectedConversationLastActive = formatDateTimeLabel(selectedConversation?.last_message_at);

  const loadedMediaItems = useMemo(() => (
    messages
      .map((message) => buildMediaItem(message, API_BASE, token))
      .filter(Boolean)
      .sort((a, b) => new Date(b.timestamp || 0).getTime() - new Date(a.timestamp || 0).getTime())
  ), [messages, token]);

  const mediaFilterOptions = useMemo(() => {
    const counts = loadedMediaItems.reduce((acc, item) => {
      acc[item.type] = (acc[item.type] || 0) + 1;
      return acc;
    }, {});

    return [
      { id: "all", label: "All media", count: loadedMediaItems.length },
      ...["image", "video", "audio", "document", "sticker"]
        .filter((type) => counts[type])
        .map((type) => ({ id: type, label: getMediaLabel(type), count: counts[type] }))
    ];
  }, [loadedMediaItems]);

  const filteredMediaItems = useMemo(() => {
    if (mediaFilter === "all") return loadedMediaItems;
    return loadedMediaItems.filter((item) => item.type === mediaFilter);
  }, [loadedMediaItems, mediaFilter]);

  const activeMediaIndex = useMemo(() => (
    filteredMediaItems.findIndex((item) => item.id === activeMediaMessageId)
  ), [filteredMediaItems, activeMediaMessageId]);

  const activeMediaItem = activeMediaIndex >= 0 ? filteredMediaItems[activeMediaIndex] : null;

  useEffect(() => {
    setMediaFilter("all");
    setActiveMediaMessageId(null);
    setHighlightedMessageId(null);
    if (highlightTimerRef.current) {
      window.clearTimeout(highlightTimerRef.current);
      highlightTimerRef.current = null;
    }
  }, [selectedConversationId]);

  useEffect(() => {
    if (!activeMediaMessageId) return;
    if (filteredMediaItems.some((item) => item.id === activeMediaMessageId)) return;
    setActiveMediaMessageId(filteredMediaItems[0]?.id || null);
  }, [activeMediaMessageId, filteredMediaItems]);

  const renderedMessages = useMemo(() => {
    const out = [];
    let lastDayKey = null;
    for (const msg of messages) {
      const t = msg.timestamp || msg.created_at;
      const dayKey = toDayKey(t);
      if (dayKey !== lastDayKey) {
        const label = formatDayLabel(t);
        if (label) {
          out.push(
            <div key={`day-${dayKey}`} className="message-date">
              {label}
            </div>
          );
        }
        lastDayKey = dayKey;
      }
      out.push(
        <ChatMessage
          key={msg.id}
          message={msg}
          isGroup={isGroupConversation}
          formatTime={formatTime}
          token={token}
          apiBase={API_BASE}
          onPreviewMedia={setActiveMediaMessageId}
          isHighlighted={highlightedMessageId === msg.id}
        />
      );
    }
    return out;
  }, [messages, isGroupConversation, token, highlightedMessageId]);

  const selectConversation = (conv) => {
    setSelectedConversationId(conv.id);
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

  const focusMessage = useCallback((messageId) => {
    const element = document.getElementById(`message-${messageId}`);
    if (!element) return;
    if (highlightTimerRef.current) {
      window.clearTimeout(highlightTimerRef.current);
    }
    setHighlightedMessageId(messageId);
    element.scrollIntoView({ behavior: "smooth", block: "center" });
    highlightTimerRef.current = window.setTimeout(() => {
      setHighlightedMessageId((current) => (current === messageId ? null : current));
      highlightTimerRef.current = null;
    }, 1800);
  }, []);

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
            <div className="sidebar-heading-row">
              <div>
                <h2 className="sidebar-title">Conversations</h2>
                <p className="sidebar-subtitle">
                  {conversations.length} chats · {unreadCount} unread messages
                </p>
              </div>
              <button className="sidebar-sync-btn" onClick={handleSyncContacts} disabled={syncingContacts}>
                <i className={`fas ${syncingContacts ? "fa-spinner fa-spin" : "fa-rotate"}`}></i>
              </button>
            </div>
            <div className="filter-tabs">
              <button className={`tab ${filter === "all" ? "active" : ""}`} onClick={() => setFilter("all")}>All</button>
              <button className={`tab ${filter === "unread" ? "active" : ""}`} onClick={() => setFilter("unread")}>Unread</button>
              <button className={`tab ${filter === "groups" ? "active" : ""}`} onClick={() => setFilter("groups")}>Groups</button>
            </div>
          </div>
          <div className="conversations-list">
            {filteredConversations.length === 0 && (
              <div className="sidebar-empty-state">
                <div className="sidebar-empty-icon"><i className="fas fa-comments"></i></div>
                <h3>No conversations yet</h3>
                <p>Pair WhatsApp and run a sync to load chats into the inbox.</p>
              </div>
            )}
            {filteredConversations.map((conv) => {
              const name = conv.contact_name || (conv.contact_number || "").split("@")[0] || "Unknown";
              const isGroup = conv.is_group ?? ((conv.contact_number || "").includes("@g.us") || (conv.contact_number || "").startsWith("12036"));
              const avatarSrc = conversationAvatarSrc(conv.id);
              return (
                <div
                  key={conv.id}
                  className={`conversation-item ${selectedConversationId === conv.id ? "active" : ""}`}
                  onClick={() => selectConversation(conv)}
                >
                  <Avatar className="conv-avatar" src={avatarSrc} name={name} />
                  <div className="conv-content">
                    <div className="conv-header">
                      <span className="conv-name">{name}</span>
                      <span className="conv-time">{formatConversationTime(conv.last_message_at)}</span>
                    </div>
                    <div className="conv-preview">{conv.last_message || "No messages yet..."}</div>
                    <div className="conv-meta">
                      {isGroup && <span><i className="fas fa-users"></i> Group</span>}
                      {(conv.unread_count || 0) > 0 && (
                        <span className="unread-badge">{conv.unread_count}</span>
                      )}
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
                <div className="chat-hero">
                  <div className="chat-user-info">
                    <Avatar
                      className="chat-avatar"
                      src={conversationAvatarSrc(selectedConversation.id)}
                      name={selectedConversationName}
                    />
                    <div className="chat-details">
                      <h3>{selectedConversationName}</h3>
                      <p className="chat-subline">{selectedConversationAddress}</p>
                      <div className="chat-detail-chips">
                        <span className="chat-chip">{isGroupConversation ? "Group conversation" : "Direct conversation"}</span>
                        <span className="chat-chip">{selectedConversationLastActive}</span>
                        <span className="chat-chip">{loadedMediaItems.length} media loaded</span>
                      </div>
                    </div>
                  </div>
                  <div className="chat-hero-stats">
                    <div className="chat-hero-stat">
                      <span>Timeline</span>
                      <strong>{messages.length}</strong>
                    </div>
                    <div className="chat-hero-stat">
                      <span>Unread</span>
                      <strong>{selectedConversation.unread_count || 0}</strong>
                    </div>
                    <div className="chat-hero-stat">
                      <span>Attachments</span>
                      <strong>{loadedMediaItems.length}</strong>
                    </div>
                    <div className="chat-hero-stat">
                      <span>Last event</span>
                      <strong>{formatConversationTime(selectedConversation.last_message_at) || "Now"}</strong>
                    </div>
                  </div>
                </div>
                <div className="chat-actions">
                  <button className="action-btn" onClick={handleSyncContacts} disabled={syncingContacts}>
                    <i className={`fas ${syncingContacts ? "fa-spinner fa-spin" : "fa-sync"}`}></i>
                    {syncingContacts ? "Syncing..." : "Sync Contacts"}
                  </button>
                  <button className="action-btn" onClick={() => { setAiChatError(""); setShowAIChat(true); }}><i className="fas fa-robot"></i> Ask AI</button>
                  <button className="action-btn primary" onClick={handleSummarize}><i className="fas fa-sparkles"></i> Summarize</button>
                </div>
              </div>

              <div className="messages-container" onScroll={handleMessagesScroll} ref={messagesContainerRef}>
                <div className="message-group">
                  {loadingMore && hasMoreMessages && (
                    <div className="message-date">Loading older messages...</div>
                  )}
                  {renderedMessages}
                </div>
              </div>

              <div className="message-input-area">
                <div className="input-wrapper">
                  <textarea
                    className="input-field"
                    placeholder="Type a WhatsApp reply..."
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
          {selectedConversation && (
            <div className="info-section conversation-brief-card">
              <div className="conversation-brief-top">
                <Avatar
                  className="conversation-brief-avatar"
                  src={conversationAvatarSrc(selectedConversation.id)}
                  name={selectedConversationName}
                />
                <div className="conversation-brief-copy">
                  <h4>{selectedConversationName}</h4>
                  <p>{selectedConversationAddress}</p>
                </div>
                {(selectedConversation.unread_count || 0) > 0 && (
                  <span className="conversation-brief-alert">{selectedConversation.unread_count} new</span>
                )}
              </div>
              <div className="conversation-brief-grid">
                <div className="conversation-brief-metric">
                  <span>Type</span>
                  <strong>{isGroupConversation ? "Group" : "Direct"}</strong>
                </div>
                <div className="conversation-brief-metric">
                  <span>Last active</span>
                  <strong>{selectedConversationLastActive}</strong>
                </div>
                <div className="conversation-brief-metric">
                  <span>Loaded history</span>
                  <strong>{messages.length} msgs</strong>
                </div>
                <div className="conversation-brief-metric">
                  <span>Media</span>
                  <strong>{loadedMediaItems.length} files</strong>
                </div>
              </div>
            </div>
          )}

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
            <button className="action-list-item" onClick={handleSummarize} disabled={!selectedConversation}>
              <div className="action-icon-small"><i className="fas fa-sparkles"></i></div>
              <div className="action-text-small">
                <div className="action-title-small">AI Summary</div>
                <div className="action-desc-small">Get conversation insights</div>
              </div>
            </button>
            <button className="action-list-item" onClick={() => { setAiChatError(""); setShowAIChat(true); }} disabled={!selectedConversation}>
              <div className="action-icon-small"><i className="fas fa-robot"></i></div>
              <div className="action-text-small">
                <div className="action-title-small">Ask AI</div>
                <div className="action-desc-small">Query the selected conversation</div>
              </div>
            </button>
            <button className="action-list-item" onClick={handleSyncContacts} disabled={syncingContacts}>
              <div className="action-icon-small"><i className={`fas ${syncingContacts ? "fa-spinner fa-spin" : "fa-rotate"}`}></i></div>
              <div className="action-text-small">
                <div className="action-title-small">Sync WhatsApp</div>
                <div className="action-desc-small">Refresh contacts and conversation metadata</div>
              </div>
            </button>
          </div>

          {selectedConversation && (
            <div className="info-section media-library-section">
              <div className="section-heading-row">
                <div>
                  <h4 className="section-title">Media Library</h4>
                  <p className="section-subtitle">
                    {loadedMediaItems.length} attachments in the loaded timeline
                  </p>
                </div>
                {filteredMediaItems[0] && (
                  <button
                    type="button"
                    className="section-link-btn"
                    onClick={() => setActiveMediaMessageId(filteredMediaItems[0].id)}
                  >
                    Open viewer
                  </button>
                )}
              </div>

              {loadedMediaItems.length > 0 ? (
                <>
                  <div className="media-filter-row">
                    {mediaFilterOptions.map((option) => (
                      <button
                        key={option.id}
                        type="button"
                        className={`media-filter-pill ${mediaFilter === option.id ? "active" : ""}`}
                        onClick={() => setMediaFilter(option.id)}
                      >
                        <span>{option.label}</span>
                        <strong>{option.count}</strong>
                      </button>
                    ))}
                  </div>

                  <div className="media-library-grid">
                    {filteredMediaItems.slice(0, 8).map((item) => (
                      <div key={item.id} className={`media-library-card ${item.type}`}>
                        <button
                          type="button"
                          className="media-library-card-main"
                          onClick={() => setActiveMediaMessageId(item.id)}
                        >
                          {item.url && (item.type === "image" || item.type === "sticker") ? (
                            <div className="media-library-thumb">
                              <img src={item.url} alt={item.fileName} loading="lazy" />
                            </div>
                          ) : (
                            <div className="media-library-icon-tile">
                              <i className={`fas ${item.icon}`}></i>
                            </div>
                          )}
                          <div className="media-library-copy">
                            <span className="media-library-kind">{item.label}</span>
                            <strong>{item.fileName}</strong>
                            <small>{[item.durationLabel, item.sizeLabel].filter(Boolean).join(" • ") || formatTime(item.timestamp)}</small>
                          </div>
                        </button>
                        <div className="media-library-actions">
                          <button type="button" className="library-mini-btn" onClick={() => focusMessage(item.id)}>
                            Locate
                          </button>
                          {item.url && (
                            <a
                              href={item.url}
                              download={item.fileName}
                              className="library-mini-btn"
                            >
                              Save
                            </a>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="media-library-empty">
                  <div className="media-library-empty-icon"><i className="fas fa-photo-film"></i></div>
                  <p>Images, videos, audio notes, and documents from this chat will appear here.</p>
                </div>
              )}
            </div>
          )}

          <div className="info-section">
            <h4 className="section-title">Create Task</h4>
            <form className="create-task-form" onSubmit={handleCreateTask}>
              <div className="form-group">
                <label className="form-label">Task Description</label>
                <input
                  type="text"
                  className="form-input"
                  placeholder={selectedConversation ? "Create a follow-up task..." : "Select a conversation first"}
                  value={taskDescription}
                  onChange={(event) => setTaskDescription(event.target.value)}
                  disabled={!selectedConversation || taskSaving}
                />
              </div>
              <div className="form-group">
                <label className="form-label">Due Date</label>
                <input
                  type="date"
                  className="form-input"
                  value={taskDueDate}
                  onChange={(event) => setTaskDueDate(event.target.value)}
                  disabled={!selectedConversation || taskSaving}
                />
              </div>
              <button className="submit-btn" type="submit" disabled={!selectedConversation || !taskDescription.trim() || taskSaving}>
                {taskSaving ? "Creating..." : "Create Task"}
              </button>
            </form>
          </div>
        </aside>
      </div>

      {activeMediaItem && (
        <MediaViewerModal
          item={activeMediaItem}
          items={filteredMediaItems}
          activeIndex={activeMediaIndex}
          onClose={() => setActiveMediaMessageId(null)}
          onPrev={() => {
            if (activeMediaIndex > 0) {
              setActiveMediaMessageId(filteredMediaItems[activeMediaIndex - 1].id);
            }
          }}
          onNext={() => {
            if (activeMediaIndex < filteredMediaItems.length - 1) {
              setActiveMediaMessageId(filteredMediaItems[activeMediaIndex + 1].id);
            }
          }}
          onSelect={setActiveMediaMessageId}
          onLocate={(messageId) => {
            setActiveMediaMessageId(null);
            focusMessage(messageId);
          }}
        />
      )}

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
