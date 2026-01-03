<p align="center">
  <img src="assets/logo.svg" alt="MessageFlow" width="520" />
</p>

# MessageFlow

MessageFlow is a multi-tenant WhatsApp operations platform built with Go, React, and PostgreSQL. It unifies real-time messaging, LLM-driven analysis, and team workflows into a single dashboard optimized for high-volume queues.

## Highlights
- Multi-tenant architecture with row-level security
- Real-time dashboard updates via WebSocket
- LLM provider abstraction with Claude, OpenAI, and Cohere
- Usage, cost, and health monitoring per provider
- Action items, summaries, and prioritization built for operations

## WhatsApp Integration Disclaimer
MessageFlow uses `whatsmeow` for WhatsApp connectivity. This is an unofficial library and is not endorsed by WhatsApp. Use at your own risk and ensure compliance with WhatsApp policies.

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
- `GET /api/v1/llm/providers/:id`
- `PATCH /api/v1/llm/providers/:id`
- `DELETE /api/v1/llm/providers/:id`
- `POST /api/v1/llm/providers/:id/test`
- `POST /api/v1/messages/analyze`
- `POST /api/v1/messages/batch-analyze`
- `POST /api/v1/conversations/summarize`
- `GET /api/v1/llm/usage`
- `GET /api/v1/llm/costs`
- `GET /api/v1/llm/health`

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

## Security
- JWT authentication on all `/api/v1/*` endpoints
- CSRF protection on state-changing requests
- Per-user rate limiting (60 req/min)
- Prepared statements for all SQL access
- RLS policies enforce tenant isolation

## Frontend Components
- `DashboardPage`
- `DailySummaryCard`
- `ImportantMessagesTab`
- `ConversationsSidebar`
- `MessagesList`
- `ActionItemsTab`
