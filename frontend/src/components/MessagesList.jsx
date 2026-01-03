import { useState } from "react";

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
    <section className="panel messages">
      <header>
        <h3>Messages</h3>
        <span>{conversation ? conversation.contact_name || conversation.contact_number : "Select a conversation"}</span>
      </header>
      <div className="panel-body message-list" onScroll={handleScroll}>
        {!conversation && <p className="empty">Pick a conversation to load messages.</p>}
        {conversation && messages.length === 0 && <p className="empty">No messages yet.</p>}
        {messages.map((message) => (
          <div key={message.id} className={`message ${message.sender === "agent" ? "from-agent" : "from-contact"}`}>
            <div>
              <p className="message-text">{message.content}</p>
              <span>{formatDate(message.timestamp)}</span>
            </div>
            <button type="button" onClick={() => setForwardMessageId(message.id)}>Forward</button>
          </div>
        ))}
        {hasMore && conversation && <p className="empty">Loading more…</p>}
      </div>
      <div className="message-actions">
        <form onSubmit={handleReply}>
          <input
            type="text"
            placeholder="Type a reply"
            value={reply}
            onChange={(event) => setReply(event.target.value)}
          />
          <button type="submit">Reply</button>
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
          <button type="submit">Forward</button>
        </form>
      </div>
    </section>
  );
}
