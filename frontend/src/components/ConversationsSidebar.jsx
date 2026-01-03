export default function ConversationsSidebar({ conversations, selected, onSelect, searchTerm, onSearch }) {
  const getInitials = (name) => {
    if (!name) return "?";
    return name.split(" ").map(n => n[0]).join("").toUpperCase().slice(0, 2);
  };

  return (
    <div className="wa-conversations">
      <input
        type="search"
        placeholder="Search conversations..."
        value={searchTerm}
        onChange={(event) => onSearch(event.target.value)}
        className="wa-search"
      />
      <div className="wa-conversations-list">
        {conversations.length === 0 && (
          <p style={{ padding: "20px", textAlign: "center", color: "var(--text-muted)", fontSize: "14px" }}>
            No conversations yet.
          </p>
        )}
        {conversations.map((convo) => (
          <button
            key={convo.id}
            type="button"
            className={`wa-convo ${selected?.id === convo.id ? "is-active" : ""}`}
            onClick={() => onSelect(convo)}
          >
            <div className="wa-convo-avatar">
              {getInitials(convo.contact_name)}
            </div>
            <div className="wa-convo-info">
              <div className="wa-convo-name">{convo.contact_name || "Unknown"}</div>
              <div className="wa-convo-meta">{convo.contact_number}</div>
            </div>
            <span className="wa-convo-time">
              {convo.last_message_at ? new Date(convo.last_message_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : "--"}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}
