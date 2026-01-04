import { useState } from "react";

export default function TopHeader({
    onNavigate,
    theme,
    setTheme,
    onLogout,
    conversationsCount = 0,
    searchTerm = "",
    setSearchTerm
}) {
    // Determine active tab based on current navigation action isn't directly props passed sadly for highlighting active tab
    // We'd ideally need a 'view' prop passed down to highlight the active button.
    // For now, we just implement the layout. The user can click to navigate.

    return (
        <nav className="navbar">
            <div className="nav-left">
                <div className="logo-section" onClick={() => onNavigate && onNavigate("operations")}>
                    <div className="logo-box">M</div>
                    <span className="brand-text">MessageFlow</span>
                </div>
                <div className="nav-status-badge">
                    <div className="status-dot" style={{ width: '6px', height: '6px', background: '#10b981', borderRadius: '50%' }}></div>
                    <span>Connected • {conversationsCount} chats</span>
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
                <div className="nav-icon-btn" title="Chats" onClick={() => onNavigate && onNavigate('operations')}>
                    <i className="fas fa-comments"></i>
                </div>
                <div className="nav-icon-btn" title="Team Hub" onClick={() => onNavigate && onNavigate('collab')}>
                    <i className="fas fa-users"></i>
                </div>
                <div className="nav-icon-btn" title="LLM Control" onClick={() => onNavigate && onNavigate('llm')}>
                    <i className="fas fa-brain"></i>
                </div>

                <div style={{ width: '1px', background: '#e5e7eb', margin: '0 8px' }}></div>

                <div className="nav-icon-btn" title="Notifications">
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
