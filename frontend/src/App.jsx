import { useEffect, useState } from "react";
import DashboardPage from "./components/DashboardPage.jsx";
import LLMProviderDashboard from "./components/llm/LLMProviderDashboard.jsx";
import ErrorBoundary from "./components/llm/ErrorBoundary.jsx";
import CollaborationPage from "./components/CollaborationPage.jsx";
import useStoredState from "./hooks/useStoredState.js";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";

export default function App() {
  const [view, setView] = useState("operations");
  const [role, setRole] = useState("viewer");
  const [token] = useStoredState("mf-token", "");
  const [csrf] = useStoredState("mf-csrf", "");

  useEffect(() => {
    if (!token) return;
    fetch(`${API_BASE}/auth/me`, {
      headers: {
        Authorization: `Bearer ${token}`,
        "X-CSRF-Token": csrf
      }
    })
      .then((res) => res.json())
      .then((data) => {
        if (data?.role) {
          setRole(data.role);
        }
      })
      .catch(() => { });
  }, [token, csrf]);

  return (
    <div>
      <nav className="app-nav">
        <div className="nav-brand">
          <img src="/logo.svg" alt="MessageFlow logo" />
        </div>
        <div className="nav-actions">
          <button
            className={view === "operations" ? "primary" : "ghost"}
            onClick={() => setView("operations")}
          >
            Operations
          </button>
          <button
            className={view === "collab" ? "primary" : "ghost"}
            onClick={() => setView("collab")}
          >
            Team Hub
          </button>
          <button
            className={view === "llm" ? "primary" : "ghost"}
            onClick={() => setView("llm")}
          >
            LLM Control
          </button>
        </div>
      </nav>
      {view === "operations" && <DashboardPage onNavigate={setView} />}
      {view === "collab" && (
        <CollaborationPage token={token} csrf={csrf} role={role} />
      )}
      {view === "llm" && (
        <ErrorBoundary>
          <LLMProviderDashboard token={token} csrf={csrf} />
        </ErrorBoundary>
      )}
    </div>
  );
}
