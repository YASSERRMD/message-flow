import { useEffect, useState } from "react";
import DashboardPage from "./components/DashboardPage.jsx";
import LLMProviderDashboard from "./components/llm/LLMProviderDashboard.jsx";
import ErrorBoundary from "./components/llm/ErrorBoundary.jsx";
import CollaborationPage from "./components/CollaborationPage.jsx";
import TopHeader from "./components/TopHeader.jsx";
import useStoredState from "./hooks/useStoredState.js";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8081/api/v1";

export default function App() {
  const [view, setView] = useState("operations");
  const [role, setRole] = useState("viewer");
  const [token, setToken] = useStoredState("mf-token", "");
  const [csrf] = useStoredState("mf-csrf", "");
  const [theme, setTheme] = useStoredState("mf-theme", "light");
  const [searchTerm, setSearchTerm] = useState("");

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  const handleLogout = () => {
    setToken("");
    window.location.reload();
  };

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
    <div className="app-container">
      {token && (
        <TopHeader
          onNavigate={setView}
          theme={theme}
          setTheme={setTheme}
          onLogout={handleLogout}
          conversationsCount={0} // TODO: Lift state if needed
          searchTerm={searchTerm}
          setSearchTerm={setSearchTerm}
        />
      )}
      {view === "operations" && <DashboardPage onNavigate={setView} searchTerm={searchTerm} />}
      {view === "collab" && (
        <CollaborationPage token={token} csrf={csrf} role={role} onNavigate={setView} />
      )}
      {view === "llm" && (
        <ErrorBoundary>
          <LLMProviderDashboard token={token} csrf={csrf} onNavigate={setView} />
        </ErrorBoundary>
      )}
    </div>
  );
}
