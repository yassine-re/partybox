# Repository Guidelines

## Project Structure & Module Organization

PartyBox is a monorepo:

- `frontend/`: SvelteKit/Svelte 5 application. Routes live in `src/routes`, reusable UI in `src/lib/components`, and API helpers in `src/lib`.
- `backend/`: Go API. `cmd/api` wires the server; `internal/handlers`, `services`, and `repositories` separate HTTP, business rules, and PostgreSQL access. Specialized packages include `ai`, `chaos`, and `realtime`.
- `database/`: ordered SQL migrations and the idempotent `seed.sql` catalog.
- `firmware/`: PlatformIO/Arduino ESP32 scaffold. `ml/` is reserved for later Python work.

Do not commit `frontend/build`, `.svelte-kit`, `.pio`, or local environment files.

## Build, Test, and Development Commands

From the repository root:

```sh
cp .env.example .env
docker compose up --build
docker compose config --quiet
docker compose run --rm migrate
docker compose run --rm seed
```

Compose starts PostgreSQL, applies migrations, seeds `PB001`, and serves the app. For native development, run `go run ./cmd/api` in `backend/` and `npm ci && npm run dev` in `frontend/`.

Before submitting changes, run:

```sh
cd backend && go test ./... && go vet ./...
cd frontend && npm run check && npm run build
```

Build firmware with `pio run -d firmware` when firmware code changes.

## Coding Style & Naming Conventions

Format Go with `gofmt`; use lowercase package names and exported `PascalCase` identifiers. Keep handlers thin and transaction-sensitive rules in services/repositories. In TypeScript, preserve strict typing, use `camelCase` values and `PascalCase.svelte` component names. Follow the existing two-space indentation. Centralize game-mode metadata instead of duplicating strings or flows.

## Testing Guidelines

Go tests use the standard `testing` package and `httptest`; files end in `_test.go` and functions start with `Test`. PostgreSQL integration tests require `TEST_DATABASE_URL` and create isolated schemas. Add focused tests for authorization, game isolation, transactions, and scoring changes. There is no enforced coverage percentage or frontend unit-test framework; `svelte-check` and the production build are required.

## Database, Security & Configuration

Add numbered `*.up.sql` and `*.down.sql` migrations; never edit an already-applied migration. Keep seeds idempotent. Store configuration in environment variables and update `.env.example` with safe placeholders. Never commit API keys, player tokens, database credentials, or tunnel URLs.

## Commit & Pull Request Guidelines

History favors concise imperative Conventional Commit subjects such as `feat: add chaos game mode` and `fix: preserve mission assignment`. Keep commits scoped. Pull requests should describe behavior, migrations, configuration changes, and validation commands; link an issue when applicable and include mobile screenshots for visible UI changes.
