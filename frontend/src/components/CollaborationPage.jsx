import TeamManagementPage from "./collab/TeamManagementPage.jsx";
import WorkflowListPage from "./collab/WorkflowListPage.jsx";
import IntegrationSettingsPage from "./collab/IntegrationSettingsPage.jsx";
import AuditLogPage from "./collab/AuditLogPage.jsx";
import AnalyticsPage from "./collab/AnalyticsPage.jsx";
import NotificationCenter from "./collab/NotificationCenter.jsx";
import ActivityTimeline from "./collab/ActivityTimeline.jsx";
import KanbanBoard from "./collab/KanbanBoard.jsx";

const roleRank = {
  owner: 5,
  admin: 4,
  manager: 3,
  member: 2,
  viewer: 1
};

const roleLabels = {
  owner: "Owner",
  admin: "Admin",
  manager: "Manager",
  member: "Member",
  viewer: "Viewer"
};

export default function CollaborationPage({ token, csrf, role }) {
  const rank = roleRank[role] || 1;
  const hasAnyFeature = rank >= 2;

  return (
  return (
    <div className="main-container">
      <aside className="conversations-sidebar" style={{ width: '280px' }}>
        <div className="sidebar-header">
          <h2 className="sidebar-title">Team Hub</h2>
          <div className="sidebar-subtitle" style={{ fontSize: '13px', color: '#6b7280' }}>
            Role: <span className="sentiment-badge positive">{roleLabels[role] || role}</span>
          </div>
        </div>
        <div className="conversations-list" style={{ padding: '16px' }}>
          <div className="action-list-item">
            <div className="action-icon-small"><i className="fas fa-columns"></i></div>
            <div className="action-text-small">
              <div className="action-title-small">Kanban Board</div>
              <div className="action-desc-small">Manage tasks</div>
            </div>
          </div>
          <div className="action-list-item">
            <div className="action-icon-small"><i className="fas fa-project-diagram"></i></div>
            <div className="action-text-small">
              <div className="action-title-small">Workflows</div>
              <div className="action-desc-small">Automations</div>
            </div>
          </div>
          <div className="action-list-item">
            <div className="action-icon-small"><i className="fas fa-chart-line"></i></div>
            <div className="action-text-small">
              <div className="action-title-small">Analytics</div>
              <div className="action-desc-small">Team performance</div>
            </div>
          </div>
        </div>
      </aside>

      <div className="chat-area">
        <div className="chat-header">
          <div className="chat-user-info">
            <div className="chat-avatar" style={{ background: 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)', color: 'white' }}>
              <i className="fas fa-users"></i>
            </div>
            <div className="chat-details">
              <h3>Collaboration Center</h3>
              <p>Manage projects, tasks, and team settings</p>
            </div>
          </div>
        </div>

        <div className="messages-container" style={{ padding: '24px' }}>
          {!hasAnyFeature && (
            <div className="empty-chat">
              <div className="empty-icon"><i className="fas fa-lock"></i></div>
              <h3>Restricted Access</h3>
              <p>Your current role ({role}) does not have access to these features.</p>
            </div>
          )}

          {hasAnyFeature && (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '24px' }}>
              {rank >= 3 && (
                <div className="connect-card" style={{ maxWidth: '100%', padding: '0', overflow: 'hidden', textAlign: 'left' }}>
                  <div style={{ padding: '20px', borderBottom: '1px solid #e5e7eb', background: '#f9fafb' }}>
                    <h3 style={{ fontSize: '16px', fontWeight: '600' }}>Active Sprint</h3>
                  </div>
                  <div style={{ padding: '20px' }}>
                    <KanbanBoard token={token} csrf={csrf} />
                  </div>
                </div>
              )}

              {rank >= 2 && (
                <div className="connect-card" style={{ maxWidth: '100%', padding: '0', overflow: 'hidden', textAlign: 'left' }}>
                  <div style={{ padding: '20px', borderBottom: '1px solid #e5e7eb', background: '#f9fafb' }}>
                    <h3 style={{ fontSize: '16px', fontWeight: '600' }}>Recent Activity</h3>
                  </div>
                  <div style={{ padding: '20px' }}>
                    <ActivityTimeline token={token} csrf={csrf} />
                  </div>
                </div>
              )}

              {rank >= 4 && (
                <div className="connect-card" style={{ maxWidth: '100%', padding: '0', overflow: 'hidden', textAlign: 'left' }}>
                  <div style={{ padding: '20px', borderBottom: '1px solid #e5e7eb', background: '#f9fafb' }}>
                    <h3 style={{ fontSize: '16px', fontWeight: '600' }}>Team Members</h3>
                  </div>
                  <div style={{ padding: '20px' }}>
                    <TeamManagementPage token={token} csrf={csrf} />
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
