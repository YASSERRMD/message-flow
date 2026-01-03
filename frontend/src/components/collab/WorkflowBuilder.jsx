import { useState } from "react";

export default function WorkflowBuilder({ onSubmit, onClose }) {
  const [name, setName] = useState("");
  const [trigger, setTrigger] = useState("");
  const [actions, setActions] = useState("{\n  \"actions\": []\n}");
  const [enabled, setEnabled] = useState(true);

  const handleSubmit = (event) => {
    event.preventDefault();
    onSubmit({
      name,
      trigger,
      actions: actions,
      enabled
    });
  };

  return (
    <div className="modal-overlay">
      <div className="modal">
        <header className="modal-header">
          <h3>Create Workflow</h3>
          <button className="ghost" onClick={onClose}>Close</button>
        </header>
        <form className="collab-form vertical modal-body" onSubmit={handleSubmit}>
          <label>
            Name
            <input value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label>
            Trigger
            <input value={trigger} onChange={(event) => setTrigger(event.target.value)} />
          </label>
          <label>
            Actions (JSON)
            <textarea rows={6} value={actions} onChange={(event) => setActions(event.target.value)} />
          </label>
          <label className="checkbox">
            <input type="checkbox" checked={enabled} onChange={(event) => setEnabled(event.target.checked)} />
            Enabled
          </label>
          <button className="primary" type="submit">Save Workflow</button>
        </form>
      </div>
    </div>
  );
}
