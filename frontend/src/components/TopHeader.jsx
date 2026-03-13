export default function TopHeader({
  onNavigate,
  activeView,
  theme,
  setTheme,
  onLogout,
  connected = false,
  conversationsCount = 0,
  unreadCount = 0,
  searchTerm = "",
  setSearchTerm
}) {
  return (
    <nav className="navbar">
      <div className="nav-left">
        <button type="button" className="logo-section" onClick={() => onNavigate && onNavigate("operations")}>
          <div className="logo-box">M</div>
          <span className="brand-text">MessageFlow</span>
        </button>
        <div className="nav-status-badge">
          <div className="status-dot" style={{ background: connected ? "#10b981" : "#f59e0b" }}></div>
          <div className="nav-status-copy">
            <span className="nav-status-label">{connected ? "WhatsApp live" : "WhatsApp disconnected"}</span>
            <small>{connected ? `${conversationsCount} synced conversations` : "Reconnect from the operations workspace"}</small>
          </div>
          {unreadCount > 0 && <strong>{unreadCount}</strong>}
        </div>
      </div>

      <div className="nav-center">
        <div className="nav-search-box">
          <i className="fas fa-search nav-search-icon"></i>
          <input
            type="text"
            className="nav-search-input"
            placeholder="Search conversations, senders, models..."
            value={searchTerm}
            onChange={(e) => setSearchTerm && setSearchTerm(e.target.value)}
          />
        </div>
      </div>

      <div className="nav-right">
        <div className="nav-mode-switch">
          <button
            type="button"
            className={`nav-mode-btn ${activeView === "operations" ? "active" : ""}`}
            onClick={() => onNavigate && onNavigate("operations")}
          >
            <i className="fas fa-comments"></i>
            <span>Chats</span>
          </button>
          <button
            type="button"
            className={`nav-mode-btn ${activeView === "collab" ? "active" : ""}`}
            onClick={() => onNavigate && onNavigate("collab")}
          >
            <i className="fas fa-users"></i>
            <span>Team</span>
          </button>
          <button
            type="button"
            className={`nav-mode-btn ${activeView === "llm" ? "active" : ""}`}
            onClick={() => onNavigate && onNavigate("llm")}
          >
            <i className="fas fa-brain"></i>
            <span>LLM</span>
          </button>
        </div>

        <div className="nav-divider"></div>

        <div className="nav-utility-group">
          <button type="button" className="nav-icon-btn" title="Notifications & Activity" onClick={() => onNavigate && onNavigate("collab")}>
            <i className="fas fa-bell"></i>
          </button>
          <button type="button" className="nav-icon-btn" title="Toggle Theme" onClick={() => setTheme(theme === "light" ? "dark" : "light")}>
            <i className={theme === "light" ? "fas fa-moon" : "fas fa-sun"}></i>
          </button>
          <button type="button" className="nav-icon-btn" title="Logout" onClick={onLogout}>
            <i className="fas fa-sign-out-alt"></i>
          </button>
        </div>
      </div>
    </nav>
  );
}
