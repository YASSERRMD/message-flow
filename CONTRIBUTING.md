# Contributing to MessageFlow

Thanks for taking the time to contribute.

## Ways to Contribute
- Report bugs (include steps to reproduce and logs)
- Improve WhatsApp Web parity (names, avatars, group sender names, live updates)
- Improve AI features (summary/chat quality, provider support, error handling)
- Fix UI/UX issues and performance problems
- Add tests and documentation

## Development Setup
Recommended:
```bash
docker compose up -d --build
```

Local dev (without Docker) is also supported. See `README.md`.

## Project Conventions
- Keep changes small and focused.
- Prefer pragmatic fixes over big rewrites.
- Avoid breaking existing API shapes used by the frontend.

Go:
- Run `go test ./...` before opening a PR.
- Keep request timeouts and error messages explicit (it helps debugging production issues).

Frontend:
- Keep API calls consistent with auth + CSRF requirements.
- Prefer predictable UI states over silent failures.

## Pull Requests
- Create a branch from `main`.
- Use descriptive branch names (example: `fix/llm-groq-summarize`).
- Keep PRs narrowly scoped.
- Include:
  - What changed
  - How to test
  - Screenshots (UI) or log snippets (backend) when relevant

## Reporting Bugs
Please include:
- What you expected to happen
- What actually happened
- Steps to reproduce
- Browser console errors (frontend)
- `docker compose logs --tail=200 backend` (backend)

