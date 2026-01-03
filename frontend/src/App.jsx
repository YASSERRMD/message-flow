import { useState } from "react";
import DashboardPage from "./components/DashboardPage.jsx";
import LLMProviderDashboard from "./components/llm/LLMProviderDashboard.jsx";
import ErrorBoundary from "./components/llm/ErrorBoundary.jsx";

export default function App() {
  const [view, setView] = useState("operations");
  const token = localStorage.getItem("mf-token") || "";
  const csrf = localStorage.getItem("mf-csrf") || "";

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
            className={view === "llm" ? "primary" : "ghost"}
            onClick={() => setView("llm")}
          >
            LLM Control
          </button>
        </div>
      </nav>
      {view === "operations" ? (
        <DashboardPage />
      ) : (
        <ErrorBoundary>
          <LLMProviderDashboard token={token} csrf={csrf} />
        </ErrorBoundary>
      )}
    </div>
  );
}
