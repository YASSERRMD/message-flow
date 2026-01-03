export default function ConversationsSidebar({ conversations, selected, onSelect, searchTerm, onSearch }) {
  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <h3>Conversations</h3>
        <input
          type="search"
          placeholder="Search conversations"
          value={searchTerm}
          onChange={(event) => onSearch(event.target.value)}
        />
      </div>
      <div className="sidebar-list">
        {conversations.length === 0 && <p className="empty">No conversations yet.</p>}
        {conversations.map((convo) => (
          <button
            key={convo.id}
            type="button"
            className={`sidebar-item ${selected?.id === convo.id ? "is-active" : ""}`}
            onClick={() => onSelect(convo)}
          >
            <div>
              <h4>{convo.contact_name || "Unknown"}</h4>
              <p>{convo.contact_number}</p>
            </div>
            <span>{convo.last_message_at ? new Date(convo.last_message_at).toLocaleTimeString() : "--"}</span>
          </button>
        ))}
      </div>
    </aside>
  );
}
