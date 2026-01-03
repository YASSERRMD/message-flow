import { useMemo, useState } from "react";

export default function ActionItemsTab({ items, conversations, onCreate, onUpdate, onDelete }) {
  const [status, setStatus] = useState("all");
  const [description, setDescription] = useState("");
  const [conversationId, setConversationId] = useState("");
  const [dueDate, setDueDate] = useState("");

  const filtered = useMemo(() => {
    if (status === "all") return items;
    return items.filter((item) => item.status === status);
  }, [items, status]);

  const handleCreate = (event) => {
    event.preventDefault();
    if (!description || !conversationId) return;
    onCreate({
      conversation_id: Number(conversationId),
      description,
      status: "open",
      due_date: dueDate || null
    });
    setDescription("");
    setConversationId("");
    setDueDate("");
  };

  return (
    <section className="panel">
      <header>
        <h3>Action Items</h3>
        <select value={status} onChange={(event) => setStatus(event.target.value)}>
          <option value="all">All</option>
          <option value="open">Open</option>
          <option value="in-progress">In Progress</option>
          <option value="done">Done</option>
        </select>
      </header>
      <div className="panel-body list">
        {filtered.length === 0 && <p className="empty">No action items.</p>}
        {filtered.map((item) => (
          <div key={item.id} className="list-row">
            <div>
              <h4>{item.description}</h4>
              <p>Status: {item.status}</p>
            </div>
            <div className="action-row">
              <button type="button" onClick={() => onUpdate(item.id, { status: "in-progress" })}>
                In Progress
              </button>
              <button type="button" onClick={() => onUpdate(item.id, { status: "done" })}>
                Done
              </button>
              <button type="button" onClick={() => onDelete(item.id)}>
                Delete
              </button>
            </div>
          </div>
        ))}
      </div>
      <form className="action-form" onSubmit={handleCreate}>
        <h4>Create Action Item</h4>
        <input
          type="text"
          placeholder="Description"
          value={description}
          onChange={(event) => setDescription(event.target.value)}
          required
        />
        <select
          value={conversationId}
          onChange={(event) => setConversationId(event.target.value)}
          required
        >
          <option value="">Conversation</option>
          {conversations.map((convo) => (
            <option key={convo.id} value={convo.id}>
              {convo.contact_name || convo.contact_number}
            </option>
          ))}
        </select>
        <input
          type="date"
          value={dueDate}
          onChange={(event) => setDueDate(event.target.value)}
        />
        <button type="submit">Create</button>
      </form>
    </section>
  );
}
