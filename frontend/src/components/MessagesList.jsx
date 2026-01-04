import { useMemo, useState, useEffect, useRef } from "react";

export default function MessagesList({
  conversation,
  messages,
  onLoadMore,
  hasMore,
  onReply,
  formatDate
}) {
  const [reply, setReply] = useState("");
  const messagesEndRef = useRef(null);
  const listRef = useRef(null);

  // Auto-scroll to bottom only when new messages arrive (and we were at the bottom)
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages.length, conversation?.id]);

  const parsedMessages = useMemo(() => {
    // Sort messages by ID or timestamp to ensure chronological order
    const sorted = [...messages].sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));

    return sorted.map((message) => {
      let analysis = null;
      if (message.metadata_json) {
        try {
          const parsed = JSON.parse(message.metadata_json);
          analysis = parsed?.analysis || parsed;
        } catch (error) {
          analysis = null;
        }
      }
      return { ...message, analysis };
    });
  }, [messages]);

  const handleReply = (event) => {
    event.preventDefault();
    if (!conversation || !reply.trim()) return;
    onReply({ conversationId: conversation.id, content: reply.trim() });
    setReply("");
  };

  const handleScroll = (e) => {
    // In strict reverse column, scrollHeight - scrollTop = clientHeight means top
    // But since we are rendering normally, scrollTop = 0 means top.
    if (e.currentTarget.scrollTop === 0 && hasMore) {
      onLoadMore();
    }
  }

  if (!conversation) {
    return (
      <section className="wa-chat-panel empty">
        <div className="empty-state-content">
          <div className="icon">💬</div>
          <h3>Select a conversation</h3>
          <p>Choose a chat from the sidebar to start messaging</p>
        </div>
      </section>
    );
  }

  return (
    <section className="wa-chat-panel">
      <header className="wa-chat-header">
        <div className="header-info">
          <h3>{conversation.contact_name || conversation.contact_number}</h3>
          <div className="wa-chat-sub">
            {conversation.contact_number}
          </div>
        </div>
      </header>

      <div className="wa-message-list" ref={listRef}>
        {hasMore && (
          <div className="load-more-trigger" onClick={onLoadMore}>
            Load older messages
          </div>
        )}

        {parsedMessages.length === 0 && (
          <p className="empty-text">No messages yet. Send one to start!</p>
        )}

        {parsedMessages.map((message) => {
          const isOutbound = message.sender === "agent" || message.sender === "me";
          // Detect group conversation (contact_number contains @g.us or starts with 12036)
          const isGroup = conversation.contact_number?.includes("@g.us") ||
            conversation.contact_number?.startsWith("12036");
          // Extract sender display name
          const senderDisplay = message.sender ? message.sender.split("@")[0] : "";
          const showSender = !isOutbound && isGroup && senderDisplay;

          return (
            <div key={message.id} className={`wa-message ${isOutbound ? "is-outbound" : ""}`}>
              <div className="wa-message-bubble">
                {showSender && (
                  <div className="wa-sender-name">
                    {senderDisplay}
                  </div>
                )}
                <p className="wa-message-text">{message.content}</p>
                <div className="wa-message-meta">
                  <span className="timestamp">
                    {new Date(message.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                  </span>
                  {isOutbound && (
                    <span className="tick">✓</span>
                  )}
                </div>
                {message.analysis?.is_important && (
                  <div className="wa-message-importance" title="Marked as important by AI">
                    ⭐️
                  </div>
                )}
              </div>
            </div>
          );
        })}
        <div ref={messagesEndRef} />
      </div>

      <div className="wa-composer">
        <form onSubmit={handleReply}>
          <input
            type="text"
            placeholder="Type a message..."
            value={reply}
            onChange={(event) => setReply(event.target.value)}
            disabled={!conversation}
          />
          <button className="primary" type="submit" disabled={!conversation || !reply.trim()}>
            <svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"></path>
            </svg>
          </button>
        </form>
      </div>
    </section>
  );
}
