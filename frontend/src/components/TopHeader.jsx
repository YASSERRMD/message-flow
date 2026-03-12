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
                <div className="logo-section" onClick={() => onNavigate && onNavigate("operations")}>
                    <div className="logo-box">M</div>
                    <span className="brand-text">MessageFlow</span>
                </div>
                <div className="nav-status-badge">
                    <div className="status-dot" style={{ background: connected ? "#10b981" : "#f59e0b" }}></div>
                    <span>{connected ? `WhatsApp live • ${conversationsCount} chats` : "WhatsApp disconnected"}</span>
                    {unreadCount > 0 && <strong>{unreadCount} unread</strong>}
                </div>
            </div>

            <div className="nav-center">
                <div className="nav-search-box">
                    <i className="fas fa-search nav-search-icon"></i>
                    <input
                        type="text"
                        className="nav-search-input"
                        placeholder="Search conversations, models..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm && setSearchTerm(e.target.value)}
                    />
                </div>
            </div>

            <div className="nav-right">
                <div className={`nav-icon-btn ${activeView === "operations" ? "active" : ""}`} title="Chats" onClick={() => onNavigate && onNavigate('operations')}>
                    <i className="fas fa-comments"></i>
                </div>
                <div className={`nav-icon-btn ${activeView === "collab" ? "active" : ""}`} title="Team Hub" onClick={() => onNavigate && onNavigate('collab')}>
                    <i className="fas fa-users"></i>
                </div>
                <div className={`nav-icon-btn ${activeView === "llm" ? "active" : ""}`} title="LLM Control" onClick={() => onNavigate && onNavigate('llm')}>
                    <i className="fas fa-brain"></i>
                </div>

                <div style={{ width: '1px', background: '#e5e7eb', margin: '0 8px' }}></div>

                <div className="nav-icon-btn" title="Notifications & Activity" onClick={() => onNavigate && onNavigate('collab')}>
                    <i className="fas fa-bell"></i>
                </div>
                <div className="nav-icon-btn" title="Toggle Theme" onClick={() => setTheme(theme === "light" ? "dark" : "light")}>
                    <i className={theme === "light" ? "fas fa-moon" : "fas fa-sun"}></i>
                </div>
                <div className="nav-icon-btn" title="Logout" onClick={onLogout}>
                    <i className="fas fa-sign-out-alt"></i>
                </div>
            </div>
        </nav>
    );
}
