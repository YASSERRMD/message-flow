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
    <div className="collab-layout">
      {!hasAnyFeature && (
        <div className="empty-state-card">
          <div className="empty-state-icon">👥</div>
          <h3>Team Hub</h3>
          <p>Welcome to the Team Hub! As a <strong>{roleLabels[role] || "Viewer"}</strong>, you currently have read-only access.</p>
          <div className="feature-unlock-info">
            <h4>Features available at higher roles:</h4>
            <ul>
              <li><strong>Member:</strong> Notifications, Activity Timeline</li>
              <li><strong>Manager:</strong> Kanban Board, Analytics</li>
              <li><strong>Admin:</strong> Team Management, Workflows, Integrations, Audit Logs</li>
            </ul>
          </div>
        </div>
      )}
      {rank >= 4 && <TeamManagementPage token={token} csrf={csrf} />}
      {rank >= 3 && <KanbanBoard token={token} csrf={csrf} />}
      {rank >= 4 && <WorkflowListPage token={token} csrf={csrf} />}
      {rank >= 4 && <IntegrationSettingsPage token={token} csrf={csrf} />}
      {rank >= 2 && <NotificationCenter token={token} csrf={csrf} />}
      {rank >= 3 && <AnalyticsPage token={token} csrf={csrf} />}
      {rank >= 2 && <ActivityTimeline token={token} csrf={csrf} />}
      {rank >= 4 && <AuditLogPage token={token} csrf={csrf} />}
    </div>
  );
}
