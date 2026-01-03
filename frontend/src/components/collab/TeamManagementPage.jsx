import { useEffect, useMemo, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

const roles = ["owner", "admin", "manager", "member", "viewer"];

export default function TeamManagementPage({ token, csrf }) {
  const [users, setUsers] = useState([]);
  const [invitations, setInvitations] = useState([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("member");
  const [loading, setLoading] = useState(true);

  const headers = useMemo(() => {
    return {
      "Content-Type": "application/json",
      Authorization: token ? `Bearer ${token}` : "",
      "X-CSRF-Token": csrf || ""
    };
  }, [token, csrf]);

  const loadTeam = async () => {
    setLoading(true);
    const response = await fetch(`${API_BASE}/team/users`, { headers });
    if (response.ok) {
      const data = await response.json();
      setUsers(data.users || []);
      setInvitations(data.invitations || []);
    }
    setLoading(false);
  };

  useEffect(() => {
    if (!token) return;
    loadTeam();
  }, [token]);

  const handleAdd = async (event) => {
    event.preventDefault();
    if (!email) return;
    await fetch(`${API_BASE}/team/users`, {
      method: "POST",
      headers,
      body: JSON.stringify({ email, role })
    });
    setEmail("");
    setRole("member");
    loadTeam();
  };

  const handleRoleChange = async (id, nextRole) => {
    await fetch(`${API_BASE}/team/users/${id}/role`, {
      method: "PATCH",
      headers,
      body: JSON.stringify({ role: nextRole })
    });
    loadTeam();
  };

  const handleRemove = async (id) => {
    await fetch(`${API_BASE}/team/users/${id}`, { method: "DELETE", headers });
    loadTeam();
  };

  const handleResend = async (invite) => {
    await fetch(`${API_BASE}/team/invitations`, {
      method: "POST",
      headers,
      body: JSON.stringify({ email: invite.email, role: invite.role })
    });
    loadTeam();
  };

  return (
    <section className="collab-card">
      <header className="collab-header">
        <div>
          <h2>Team Management</h2>
          <p>Invite teammates, assign roles, and manage access.</p>
        </div>
        <div className="tag">{loading ? "Syncing" : `${users.length} members`}</div>
      </header>

      <form className="collab-form" onSubmit={handleAdd}>
        <input
          type="email"
          placeholder="teammate@company.com"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
        <select value={role} onChange={(event) => setRole(event.target.value)}>
          {roles.map((item) => (
            <option key={item} value={item}>{item}</option>
          ))}
        </select>
        <button className="primary" type="submit">Add or Invite</button>
      </form>

      <div className="collab-list">
        {users.map((user) => (
          <div className="collab-row" key={user.id}>
            <div>
              <strong>{user.email}</strong>
              <p>{user.role || "viewer"}</p>
            </div>
            <div className="collab-row-actions">
              <select
                value={user.role || "viewer"}
                onChange={(event) => handleRoleChange(user.id, event.target.value)}
              >
                {roles.map((item) => (
                  <option key={item} value={item}>{item}</option>
                ))}
              </select>
              <button className="ghost" onClick={() => handleRemove(user.id)}>Remove</button>
            </div>
          </div>
        ))}
      </div>

      {invitations.length > 0 && (
        <div className="collab-subsection">
          <h3>Pending Invitations</h3>
          {invitations.map((invite) => (
            <div className="collab-row" key={invite.id}>
              <div>
                <strong>{invite.email}</strong>
                <p>{invite.role} · {invite.status}</p>
              </div>
              <div className="collab-row-actions">
                <button className="ghost" onClick={() => handleResend(invite)}>Resend</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
