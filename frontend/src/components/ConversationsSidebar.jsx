export default function ConversationsSidebar({ conversations, selected, onSelect, searchTerm, onSearch }) {
  return (
    <div className="wa-conversations">
      <div className="wa-conversations-header">
        <div>
          <h3>Chats</h3>
          <span className="wa-meta">{conversations.length} conversations</span>
        </div>
        <input
          type="search"
          placeholder="Search conversations"
          value={searchTerm}
          onChange={(event) => onSearch(event.target.value)}
          className="wa-search"
        />
      </div>
      <div className="wa-conversations-list">
        {conversations.length === 0 && <p className="empty">No conversations yet.</p>}
        {conversations.map((convo) => (
          <button
            key={convo.id}
            type="button"
            className={`wa-convo ${selected?.id === convo.id ? "is-active" : ""}`}
            onClick={() => onSelect(convo)}
          >
            <div>
              <div className="wa-convo-name">{convo.contact_name || "Unknown"}</div>
              <div className="wa-convo-meta">{convo.contact_number}</div>
            </div>
            <span className="wa-convo-time">
              {convo.last_message_at ? new Date(convo.last_message_at).toLocaleTimeString() : "--"}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}
