CREATE TABLE IF NOT EXISTS users_extended (
  user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  tenant_id BIGINT NOT NULL,
  role TEXT NOT NULL DEFAULT 'viewer',
  team_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS users_extended_tenant_idx ON users_extended (tenant_id, role);

CREATE TABLE IF NOT EXISTS user_roles (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  tenant_id BIGINT NOT NULL,
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS user_roles_tenant_idx ON user_roles (tenant_id, user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS team_members (
  id BIGSERIAL PRIMARY KEY,
  team_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS team_members_unique ON team_members (team_id, user_id);

CREATE TABLE IF NOT EXISTS team_invitations (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  email TEXT NOT NULL,
  role TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  token TEXT NOT NULL,
  invited_by BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS team_invitations_tenant_idx ON team_invitations (tenant_id, status, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS team_invitations_unique ON team_invitations (tenant_id, email, status);

CREATE TABLE IF NOT EXISTS workflows (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  name TEXT NOT NULL,
  trigger TEXT NOT NULL,
  actions_json JSONB NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workflow_executions (
  id BIGSERIAL PRIMARY KEY,
  workflow_id BIGINT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  execution_time TIMESTAMPTZ NOT NULL,
  success BOOLEAN NOT NULL DEFAULT TRUE,
  error_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS workflow_executions_idx ON workflow_executions (workflow_id, execution_time DESC);

CREATE TABLE IF NOT EXISTS integrations (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  type TEXT NOT NULL,
  config_json JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS integrations_tenant_idx ON integrations (tenant_id, type);
CREATE UNIQUE INDEX IF NOT EXISTS integrations_unique ON integrations (tenant_id, type);

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  user_id BIGINT,
  action TEXT NOT NULL,
  resource_type TEXT,
  resource_id BIGINT,
  changes_before_json JSONB,
  changes_after_json JSONB,
  ip_address TEXT,
  user_agent TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS audit_logs_tenant_idx ON audit_logs (tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS labels (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  name TEXT NOT NULL,
  color TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS labels_unique ON labels (tenant_id, name);

CREATE TABLE IF NOT EXISTS message_labels (
  message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  label_id BIGINT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
  tenant_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (message_id, label_id)
);

CREATE TABLE IF NOT EXISTS comments (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  action_item_id BIGINT NOT NULL REFERENCES action_items(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notifications (
  id BIGSERIAL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  content TEXT NOT NULL,
  read BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users_extended ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE team_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE team_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE integrations ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE labels ENABLE ROW LEVEL SECURITY;
ALTER TABLE message_labels ENABLE ROW LEVEL SECURITY;
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_users_extended ON users_extended
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_user_roles ON user_roles
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_team_members ON team_members
  USING (team_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (team_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_team_invitations ON team_invitations
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_workflows ON workflows
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_workflow_executions ON workflow_executions
  USING (workflow_id IN (SELECT id FROM workflows WHERE tenant_id = current_setting('app.tenant_id')::bigint))
  WITH CHECK (workflow_id IN (SELECT id FROM workflows WHERE tenant_id = current_setting('app.tenant_id')::bigint));

CREATE POLICY tenant_isolation_integrations ON integrations
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_audit_logs ON audit_logs
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_labels ON labels
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_message_labels ON message_labels
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_comments ON comments
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

CREATE POLICY tenant_isolation_notifications ON notifications
  USING (tenant_id = current_setting('app.tenant_id')::bigint)
  WITH CHECK (tenant_id = current_setting('app.tenant_id')::bigint);

ALTER TABLE action_items
  ADD COLUMN IF NOT EXISTS watchers_json JSONB;
