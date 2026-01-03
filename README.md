# MessageFlow

MessageFlow is a multi-tenant WhatsApp message management dashboard built with Go, React, and PostgreSQL. Phase 1 delivers the foundational dashboard, real-time updates, authentication, and operational tooling.

## Architecture
- Go API (`backend`) with JWT authentication, rate limiting, CSRF protection, and row-level security.
- React dashboard (`frontend`) with componentized UI, WebSocket updates, and theme toggle.
- PostgreSQL with tenant-isolated schema.

## WhatsApp Integration
MessageFlow uses `whatsmeow` for WhatsApp connectivity. WhatsApp integration is powered by an unofficial library and is not endorsed by WhatsApp. Use at your own risk and ensure compliance with WhatsApp policies.

## Environment
Backend environment variables:
- `DATABASE_URL` (required)
- `JWT_SECRET` (required)
- `PORT` (default: 8080)
- `FRONTEND_ORIGIN` (default: http://localhost:5173)

Frontend environment variables:
- `VITE_API_BASE` (default: http://localhost:8080/api/v1)
- `VITE_WS_BASE` (default: ws://localhost:8080/api/v1)

## Database
Apply the migration:
- `backend/migrations/001_init.sql`

Row-level security (RLS) policies use `app.tenant_id` per request. The API sets this value for every query.

## API Endpoints
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
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/register`
- `GET /api/v1/auth/me`

Authentication:
- Use `Authorization: Bearer <token>` on all `/api/v1/*` endpoints except login/register.
- For state-changing requests, include `X-CSRF-Token` returned by login/register.

## WebSocket
- `GET /api/v1/ws?token=<jwt>`
- Broadcasts events for replies, forwards, and action item changes.

## Frontend
The dashboard is composed of:
- `DashboardPage`
- `DailySummaryCard`
- `ImportantMessagesTab`
- `ConversationsSidebar`
- `MessagesList`
- `ActionItemsTab`

## Development
Backend:
- `cd backend`
- `go run ./cmd/server`

Frontend:
- `cd frontend`
- `npm install`
- `npm run dev`

## Testing
- `cd backend`
- `go test ./...`

## Security Notes
- JWT authentication with per-request tenant isolation
- CSRF protection for state-changing operations
- Rate limiting at 60 requests/minute per user
- Prepared statements for all SQL queries
- CORS restricted to `FRONTEND_ORIGIN`
