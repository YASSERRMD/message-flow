<p align="center">
  <img src="assets/logo.svg" alt="MessageFlow" width="120" />
</p>

# MessageFlow

MessageFlow is a multi-tenant WhatsApp operations platform built with Go, React, and PostgreSQL. It unifies real-time messaging, LLM-driven analysis, and team workflows into a single dashboard optimized for high-volume queues.

## Highlights
- Multi-tenant architecture with row-level security
- Real-time dashboard updates via WebSocket
- LLM provider abstraction with Claude, OpenAI, and Cohere
- Usage, cost, and health monitoring per provider
- Team collaboration with RBAC, workflows, and integrations
- Action items, summaries, and prioritization built for operations

> [!WARNING]
> WhatsApp integration uses `whatsmeow`, an unofficial library not endorsed by WhatsApp. Use at your own risk and ensure compliance with WhatsApp policies.

## Architecture
- Go API (`backend`) with JWT authentication, rate limiting, CSRF protection, and tenant isolation
- React dashboard (`frontend`) with componentized UI, dark/light mode, and real-time streaming
- PostgreSQL with RLS policies and LLM usage logs
- Redis queue for batch analysis (optional)

## Tech Stack
- Go 1.21+ (backend)
- React + Vite (frontend)
- PostgreSQL (data)
- Redis (analysis queue)
- Slack, Email, Webhooks (integrations)

## Quick Start
Backend:
- `cd backend`
- `export DATABASE_URL=...`
- `export JWT_SECRET=...`
- `export MASTER_KEY=...`
- `go run ./cmd/server`

Frontend:
- `cd frontend`
- `npm install`
- `npm run dev`

## Environment
Backend:
- `DATABASE_URL` (required)
- `JWT_SECRET` (required)
- `MASTER_KEY` (required, encrypts provider API keys)
- `PORT` (default: 8080)
- `FRONTEND_ORIGIN` (default: http://localhost:5173)
- `REDIS_URL` (optional, enables batch queue)

Frontend:
- `VITE_API_BASE` (default: http://localhost:8080/api/v1)
- `VITE_WS_BASE` (default: ws://localhost:8080/api/v1)

## Database Migrations
- `backend/migrations/001_init.sql`
- `backend/migrations/002_phase2_llm.sql`
- `backend/migrations/003_phase3_llm_management.sql`
- `backend/migrations/004_phase3_llm_provider_fields.sql`
- `backend/migrations/005_phase4_team_collaboration.sql`

## API Endpoints
Core:
- `GET /api/v1/dashboard`
- `GET /api/v1/conversations`
- `GET /api/v1/conversations/:id/messages`
- `POST /api/v1/messages/reply`
- `POST /api/v1/messages/forward`
- `GET /api/v1/important-messages`
- `POST /api/v1/action-items`
- `PATCH /api/v1/action-items/:id`
- `DELETE /api/v1/action-items/:id`
- `GET /api/v1/action-items`
- `GET /api/v1/daily-summary`

Auth:
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/register`
- `GET /api/v1/auth/me`

LLM:
- `POST /api/v1/llm/providers`
- `GET /api/v1/llm/providers`
- `GET /api/v1/llm/providers/comparison`
- `GET /api/v1/llm/providers/:id`
- `GET /api/v1/llm/providers/:id/history`
- `PATCH /api/v1/llm/providers/:id`
- `DELETE /api/v1/llm/providers/:id`
- `POST /api/v1/llm/providers/:id/test`
- `POST /api/v1/messages/analyze`
- `POST /api/v1/messages/batch-analyze`
- `POST /api/v1/conversations/summarize`
- `GET /api/v1/llm/usage`
- `GET /api/v1/llm/costs`
- `GET /api/v1/llm/health`
- `GET /api/v1/llm/features`
- `POST /api/v1/llm/features/:name/assign-provider`
- `GET /api/v1/llm/features/:name/providers`
- `DELETE /api/v1/llm/features/:name/providers/:id`
- `GET /api/v1/llm/analytics/cost-breakdown`
- `GET /api/v1/llm/analytics/usage-by-feature`
- `POST /api/v1/llm/bulk-test`
- `GET /api/v1/llm/recommendations`

Team:
- `POST /api/v1/team/users`
- `GET /api/v1/team/users`
- `PATCH /api/v1/team/users/:id/role`
- `DELETE /api/v1/team/users/:id`
- `POST /api/v1/team/invitations`
- `GET /api/v1/team/activity`

Workflows:
- `POST /api/v1/workflows`
- `GET /api/v1/workflows`
- `PATCH /api/v1/workflows/:id`
- `DELETE /api/v1/workflows/:id`
- `POST /api/v1/workflows/:id/test`
- `GET /api/v1/workflows/:id/executions`

Integrations:
- `POST /api/v1/integrations/:type`
- `GET /api/v1/integrations`
- `DELETE /api/v1/integrations/:id`
- `GET /api/v1/integrations/:id/config`
- `POST /api/v1/webhooks/incoming`

Audit + Notifications:
- `POST /api/v1/audit-logs`
- `POST /api/v1/notifications`
- `PATCH /api/v1/notifications/:id`

Labels + Comments:
- `POST /api/v1/labels`
- `POST /api/v1/messages/:id/labels`
- `GET /api/v1/action-items/:id/comments`
- `POST /api/v1/action-items/:id/comments`
- `DELETE /api/v1/comments/:id`

WebSocket:
- `GET /api/v1/ws?token=<jwt>`

## LLM Providers
Supported providers and defaults:
- Claude: `claude-3-opus-20240229`
- OpenAI: `gpt-4-turbo`
- Cohere: `command-r-plus`

Each provider is configured per tenant with rate limits, temperature, token caps, and cost tracking.

## Testing
Backend tests:
- `cd backend`
- `go test ./...`

## Docker
- `docker compose up --build`
- Migrations run automatically on startup via the `migrate` service.

## Security
- JWT authentication on all `/api/v1/*` endpoints
- CSRF protection on state-changing requests
- Per-user rate limiting (60 req/min)
- Prepared statements for all SQL access
- RLS policies enforce tenant isolation
- RBAC enforcement with audited permission checks

## Frontend Components
- `DashboardPage`
- `DailySummaryCard`
- `ImportantMessagesTab`
- `ConversationsSidebar`
- `MessagesList`
- `ActionItemsTab`
- `TeamManagementPage`
- `WorkflowBuilder`
- `WorkflowListPage`
- `IntegrationSettingsPage`
- `AuditLogPage`
- `AnalyticsPage`
- `NotificationCenter`
- `CommentThread`
- `ActivityTimeline`
- `KanbanBoard`
