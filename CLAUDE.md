# CLAUDE.md

Project conventions, commands, architecture and rules live in **@AGENTS.md** —
read it first. It is the single source of truth for how to work in this repo.

## Claude-specific notes

- This repo uses the `workflow-skills` plugin. For non-trivial tasks route
  through the matching skill (`/plan`, `/tdd`, `/review`, `/verify`, …).
- Go and Node are not on the host: run every build/test via
  `docker compose --profile test run --rm backend-test …` (or `web-test`).
- `sec.gov` is blocked only inside the AustriaCard corporate network; from
  other networks it works (SEC_USER_AGENT in `.env`). Tests use the fixtures
  in `ingest/sec/testdata` and `catalog/testdata/sec` either way.
