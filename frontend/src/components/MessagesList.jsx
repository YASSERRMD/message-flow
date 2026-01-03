import { useMemo, useState } from "react";

export default function MessagesList({
  conversation,
  messages,
  onLoadMore,
  hasMore,
  onReply,
  onForward,
  formatDate
}) {
  const [reply, setReply] = useState("");
  const [forwardMessageId, setForwardMessageId] = useState("");
  const [targetConversation, setTargetConversation] = useState("");

  const parsedMessages = useMemo(() => {
    return messages.map((message) => {
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

  const handleScroll = (event) => {
    const { scrollTop, scrollHeight, clientHeight } = event.currentTarget;
    if (scrollTop + clientHeight >= scrollHeight - 32 && hasMore) {
      onLoadMore();
    }
  };

  const handleReply = (event) => {
    event.preventDefault();
    if (!conversation || !reply.trim()) return;
    onReply({ conversationId: conversation.id, content: reply.trim() });
    setReply("");
  };

  const handleForward = (event) => {
    event.preventDefault();
    if (!forwardMessageId || !targetConversation) return;
    onForward({
      messageId: Number(forwardMessageId),
      targetConversationId: Number(targetConversation)
    });
    setForwardMessageId("");
    setTargetConversation("");
  };

  return (
    <section className="wa-chat-panel">
      <header className="wa-chat-header">
        <div>
          <h3>{conversation ? conversation.contact_name || conversation.contact_number : "Select a chat"}</h3>
          <div className="wa-chat-sub">
            {conversation ? conversation.contact_number : "Choose a conversation to load messages"}
          </div>
        </div>
        <div className="wa-chat-sub">AI insights active</div>
      </header>
      <div className="wa-message-list" onScroll={handleScroll}>
        {!conversation && <p className="empty">Pick a conversation to load messages.</p>}
        {conversation && messages.length === 0 && <p className="empty">No messages yet.</p>}
        {parsedMessages.map((message) => {
          const isOutbound = message.sender === "agent" || message.sender === "me";
          return (
            <div key={message.id} className={`wa-message ${isOutbound ? "is-outbound" : ""}`}>
              <div className="wa-message-bubble">
                <p className="wa-message-text">{message.content}</p>
                <div className="wa-message-meta">
                  <span>{formatDate(message.timestamp)}</span>
                  <button type="button" className="ghost" onClick={() => setForwardMessageId(message.id)}>
                    Forward
                  </button>
                </div>
                {message.analysis?.is_important && (
                  <div className="wa-message-tags">
                    <span className="wa-tag">{message.analysis.priority || "important"}</span>
                    {message.analysis.sentiment && <span className="wa-tag">{message.analysis.sentiment}</span>}
                  </div>
                )}
              </div>
            </div>
          );
        })}
        {hasMore && conversation && <p className="empty">Loading more…</p>}
      </div>
      <div className="wa-composer">
        <form onSubmit={handleReply}>
          <input
            type="text"
            placeholder="Type a reply"
            value={reply}
            onChange={(event) => setReply(event.target.value)}
          />
          <button className="primary" type="submit">Send</button>
        </form>
        <form onSubmit={handleForward} className="forward-form">
          <input
            type="number"
            placeholder="Message ID"
            value={forwardMessageId}
            onChange={(event) => setForwardMessageId(event.target.value)}
          />
          <input
            type="number"
            placeholder="Target conversation ID"
            value={targetConversation}
            onChange={(event) => setTargetConversation(event.target.value)}
          />
          <button className="ghost" type="submit">Forward</button>
        </form>
      </div>
    </section>
  );
}
