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
    return (
        <header className="top-header">
            <div className="header-left">
                <div className="logo" onClick={() => onNavigate && onNavigate("operations")} style={{ cursor: "pointer" }}>
                    <div className="logo-icon"><i className="fas fa-comment-dots"></i></div>
                    <span>MessageFlow</span>
                </div>
                <div className="connection-status">
                    <div className="status-dot"></div>
                    <span>Connected • {conversationsCount} chats</span>
                </div>
            </div>
            <div className="header-search">
                <i className="fas fa-search search-icon"></i>
                <input
                    type="text"
                    className="search-input"
                    placeholder="Search conversations..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm && setSearchTerm(e.target.value)}
                />
            </div>
            <div className="header-right">
                <button className="header-btn" title="Chats" onClick={() => onNavigate && onNavigate('operations')}>
                    <i className="fas fa-comments"></i>
                </button>
                <button className="header-btn" title="Team Hub" onClick={() => onNavigate && onNavigate('collab')}>
                    <i className="fas fa-users"></i>
                </button>
                <button className="header-btn" title="LLM Control" onClick={() => onNavigate && onNavigate('llm')}>
                    <i className="fas fa-brain"></i>
                </button>
                <button className="header-btn" title="Notifications"><i className="fas fa-bell"></i></button>
                <button className="header-btn" onClick={() => setTheme(theme === "light" ? "dark" : "light")}>
                    <i className={theme === "light" ? "fas fa-moon" : "fas fa-sun"}></i>
                </button>
                <button className="header-btn" onClick={onLogout}><i className="fas fa-sign-out-alt"></i></button>
            </div>
        </header>
    );
}
