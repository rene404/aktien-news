# CLAUDE.md

Project conventions, commands, architecture, and rules live in **@AGENTS.md** —
read it first. It is the single source of truth for how to work in this repo.

## Claude-specific notes

- This repo installs the `workflow-skills` plugin. For non-trivial tasks, route
  through the appropriate skill (`/plan`, `/tdd`, `/review`, `/security-review`,
  `/verify`, etc.) rather than jumping straight to code.
- Before declaring backend work done: `cd backend && uv run pytest -q`
  (requires the `aktien_news_test` database — see @AGENTS.md → Commands).
- Before declaring frontend work done: `cd frontend && npm run lint && npm run build`.
- Phase 2 will add Claude API integration (article summaries). When that work
  starts, use the latest Claude models — default to `claude-opus-4-8` /
  `claude-sonnet-4-6`; verify model IDs and pricing against current docs rather
  than memory.
