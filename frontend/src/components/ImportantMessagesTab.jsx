import { useMemo, useState } from "react";

export default function ImportantMessagesTab({ items, formatDate }) {
  const [priority, setPriority] = useState("all");

  const filtered = useMemo(() => {
    if (priority === "all") return items;
    return items.filter((item) => item.priority === priority);
  }, [items, priority]);

  return (
    <section className="panel">
      <header>
        <h3>Important Messages</h3>
        <select value={priority} onChange={(event) => setPriority(event.target.value)}>
          <option value="all">All</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
      </header>
      <div className="panel-body list">
        {filtered.length === 0 && <p className="empty">No important messages.</p>}
        {filtered.map((item) => (
          <div key={item.id} className="list-row">
            <div>
              <h4>{item.priority}</h4>
              <p>{item.content}</p>
            </div>
            <span>{formatDate(item.timestamp)}</span>
          </div>
        ))}
      </div>
    </section>
  );
}
