import { useEffect, useState } from "react";
import DashboardPage from "./components/DashboardPage.jsx";
import LLMProviderDashboard from "./components/llm/LLMProviderDashboard.jsx";
import ErrorBoundary from "./components/llm/ErrorBoundary.jsx";
import CollaborationPage from "./components/CollaborationPage.jsx";
import TopHeader from "./components/TopHeader.jsx";
import useStoredState from "./hooks/useStoredState.js";
import { getApiBase } from "./lib/runtimeConfig.js";

const API_BASE = getApiBase();

export default function App() {
  const [view, setView] = useState("operations");
  const [role, setRole] = useState("viewer");
  const [token, setToken] = useStoredState("mf-token", "");
  const [csrf, setCsrf] = useStoredState("mf-csrf", "");
  const [theme, setTheme] = useStoredState("mf-theme", "light");
  const [searchTerm, setSearchTerm] = useState("");
  const [operationsMeta, setOperationsMeta] = useState({
    connected: false,
    conversationsCount: 0,
    unreadCount: 0
  });

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  const handleLogout = async () => {
    // Ensure WhatsApp session is disconnected server-side too.
    try {
      if (token) {
        await fetch(`${API_BASE}/auth/logout`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
            "X-CSRF-Token": csrf || ""
          }
        });
      }
    } catch { }
    setToken("");
    setCsrf("");
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
        if (data?.error === "unauthorized" || data?.error === "forbidden") {
          setToken("");
          setCsrf("");
          return;
        }
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
          activeView={view}
          theme={theme}
          setTheme={setTheme}
          onLogout={handleLogout}
          connected={operationsMeta.connected}
          conversationsCount={operationsMeta.conversationsCount}
          unreadCount={operationsMeta.unreadCount}
          searchTerm={searchTerm}
          setSearchTerm={setSearchTerm}
        />
      )}
      {view === "operations" && (
        <DashboardPage
          onNavigate={setView}
          searchTerm={searchTerm}
          onMetaChange={setOperationsMeta}
        />
      )}
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
