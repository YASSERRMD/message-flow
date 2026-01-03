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

export default function CollaborationPage({ token, csrf, role }) {
  const rank = roleRank[role] || 1;

  return (
    <div className="collab-layout">
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
