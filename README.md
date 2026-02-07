# MessageFlow

MessageFlow is a simple WhatsApp Web style dashboard for operations teams, backed by a Go API and PostgreSQL. It uses the `whatsmeow` Go library to connect to WhatsApp, sync conversations/messages, and stream real-time updates to the UI. It also supports LLM-powered conversation summaries and "chat with this conversation" using OpenAI-compatible providers (including Groq).

## What You Get
- WhatsApp pairing via QR code (multi-tenant sessions)
- Conversation list (names, avatars, last message, unread counts)
- Conversation view (history, group sender names)
- Reply + forward messages
- Real-time updates via WebSocket (new messages, unread count, last message)
- AI features
  - Summarize a conversation
  - Ask questions about a conversation (RAG-like over recent messages)
- Provider management UI (add/test providers, usage/cost/health panels)

> Warning: WhatsApp integration uses `whatsmeow`, which is unofficial and not endorsed by WhatsApp. Use at your own risk and in compliance with WhatsApp policies.

## Architecture
- `backend/`: Go HTTP API + WebSocket hub
- `frontend/`: React (Vite build) served by Nginx in Docker
- `db`: PostgreSQL 15
- `redis`: optional queue for batch LLM analysis

## Quick Start (Recommended: Docker Compose)
Prereqs:
- Docker Desktop
- GitHub CLI (optional, only for PRs)

Start:
```bash
docker compose up -d --build
```

Open:
- Frontend: [http://localhost:5173](http://localhost:5173)
- Backend API: [http://localhost:8081/api/v1](http://localhost:8081/api/v1)

Ports:
- Frontend: `5173`
- Backend: `8081`
- Postgres: `5432`
- Redis: `6379`

Migrations run automatically via the `migrate` service.

## First Run Checklist
1. Open the app at [http://localhost:5173](http://localhost:5173).
2. Pair WhatsApp.
   - Use the connect screen to generate a QR code.
   - Scan the QR code with WhatsApp on your phone.
3. Wait for initial sync to populate conversations and messages.
4. Configure an LLM provider for summaries/chat (optional but recommended).

## LLM Providers (Groq / OpenAI-Compatible)
Providers are configured per tenant and stored encrypted in the DB (encrypted with `MASTER_KEY`).

Groq works through the OpenAI-compatible API:
- Base URL: `https://api.groq.com/openai/v1`
- Model: `llama-3.3-70b-versatile` (or any Groq-supported chat model)

In the UI:
- Go to `LLM Control`
- `Add Provider`
- Choose `Groq`
- Set `API Key`, `Model Name`, and (optionally) `Base URL`

Notes:
- Summaries use the default provider, and then (optionally) providers marked as fallback.
- If you have seed/demo providers with invalid keys, keep them inactive or not-fallback to avoid confusing failures.

## Configuration
Docker Compose provides default env vars for local usage in `docker-compose.yml`.

Backend env:
- `DATABASE_URL` (required)
- `JWT_SECRET` (required)
- `MASTER_KEY` (required, encrypts LLM provider API keys)
- `PORT` (default `8080` in container)
- `FRONTEND_ORIGIN` (default `http://localhost:5173`)
- `REDIS_URL` (optional)

Frontend env/build args:
- `VITE_API_BASE` (default `http://localhost:8081/api/v1`)
- `VITE_WS_BASE` (default `ws://localhost:8081/api/v1`)

## Development (Without Docker)
Backend:
```bash
cd backend
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/messageflow?sslmode=disable"
export JWT_SECRET="changeme"
export MASTER_KEY="changemechangemechangemechangeme"
export FRONTEND_ORIGIN="http://localhost:5173"
go run ./cmd/server
```

Frontend:
```bash
cd frontend
npm install
npm run dev
```

## Testing
Backend:
```bash
cd backend
go test ./...
```

## Troubleshooting
- `500` on `/api/v1/conversations` or `/api/v1/dashboard`
  - Check backend logs: `docker compose logs -f backend`
  - Verify DB is healthy: `docker compose ps`
- LLM summary/chat fails
  - Ensure your user has a role (member/owner). `/api/v1/auth/me` returns `role`.
  - Verify the provider in `LLM Control` is active and has a valid model + key.
  - If using Groq, ensure model is a Groq chat model and base URL is `https://api.groq.com/openai/v1`.
- WhatsApp not syncing after scan
  - Check backend logs for session reconnect and sync status.
  - Reconnect and re-scan from the UI if needed.

## Contributing
Contributions are welcome. If you want to help improve WhatsApp parity, UI behavior, or LLM features:
- Read [CONTRIBUTING.md](CONTRIBUTING.md)
- Open an issue or a PR with a clear description and screenshots/logs when relevant

## License
MIT. See [LICENSE](LICENSE).

