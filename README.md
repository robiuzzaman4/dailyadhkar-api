# Daily Adhkar API

Backend service for Daily Adhkars. The API manages users, tracks subscriptions, and sends daily reminder emails through UnoSend on a cron schedule.

## Tech Stack

- Go (`net/http`, no framework)
- PostgreSQL (`pgx/v5`)
- UnoSend (email delivery)
- Cron scheduler (`robfig/cron/v3`)

## Project Structure

```text
cmd/server/                       # Application entrypoint
internal/application/             # Use-cases and orchestration
internal/domain/                  # Domain models + repository contracts
internal/infrastructure/          # Config, DB, external integrations
internal/interfaces/http/         # HTTP server, routes, middleware
internal/infrastructure/database/migrations/
```

## Prerequisites

- Go 1.24+
- PostgreSQL 14+
- UnoSend API key

## Environment Setup

1. Copy environment template:

```bash
cp .env.example .env
```

2. Fill required variables in `.env`:

```env
APP_ENV=development
SERVER_PORT=8080

DATABASE_URL=postgres://postgres:postgres@localhost:5432/daily_adhkar?sslmode=disable

UNOSEND_API_KEY=your_unosend_api_key
UNOSEND_BASE_URL=https://www.unosend.co/api/v1/emails
DEFAUL_EMAIL_SENDER=Daily Adhkar <noreply@send.deentab.app>

EMAIL_SEND_TIME=10:00AM
EMAIL_SEND_LIMIT=10

CORS_ALLOWED_ORIGINS=http://localhost:3000
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type,X-Request-ID
CORS_ALLOW_CREDENTIALS=true
```

### Config Validation Rules

The service fails fast at startup if required values are missing/invalid:

- Required: `DATABASE_URL`, `UNOSEND_API_KEY`, `UNOSEND_BASE_URL`, `DEFAUL_EMAIL_SENDER`, `EMAIL_SEND_TIME`, `EMAIL_SEND_LIMIT`
- `EMAIL_SEND_TIME` must be 12-hour format (`10:00AM`)
- `EMAIL_SEND_LIMIT` must be a positive integer

## Database Migrations

Migration files:

- `internal/infrastructure/database/migrations/0001_create_users.up.sql`
- `internal/infrastructure/database/migrations/0001_create_users.down.sql`

### Option A: `golang-migrate` (recommended)

```bash
migrate -path internal/infrastructure/database/migrations \
  -database "$DATABASE_URL" up
```

Rollback:

```bash
migrate -path internal/infrastructure/database/migrations \
  -database "$DATABASE_URL" down 1
```

### Option B: Apply SQL manually

```bash
psql "$DATABASE_URL" -f internal/infrastructure/database/migrations/0001_create_users.up.sql
```

## Run Commands

Run from repository root (so `.env` is discovered):

```bash
go run ./cmd/server
```

Build binary:

```bash
go build -o bin/server ./cmd/server
```

Run tests:

```bash
go test ./...
```

## Docker

Build image:

```bash
docker build -t dailyadhkar-api .
```

Run container:

```bash
docker run --rm -p 8080:8080 --env-file .env dailyadhkar-api
```

### Zeabur Note

- Use repository root (`/`) as root directory.
- Enable Dockerfile-based deployment so Zeabur runs this API container.
- Provide runtime env vars from dashboard (do not include single quotes in values).

## API Summary

Detailed spec:

- `openapi.yaml`
- `API_DOCUMENTATION.md`

Current route groups:

- Health: `GET /health`
- Users:
  - `POST /users` (create user with name, email, gender)
  - `GET /users` (list all users)
  - `GET /users/{id}` (get single user)
  - `PATCH /users/{id}` (update `is_subscribed`)
  - `DELETE /users/{id}` (delete user)
- Metadata: `GET /metadata`

## Daily Cron + Email Workflow

- Scheduler starts during app bootstrap.
- Cron schedule is computed from `EMAIL_SEND_TIME` (local server timezone).
- At each run:
  1. Load all subscribed users (`is_subscribed = true`).
  2. Dispatch with worker pool size = `EMAIL_SEND_LIMIT`.
  3. Send each reminder via UnoSend (`/v1/emails`).
  4. Retry each send up to 3 times with 2s delay.
  5. On success, increment `total_email_received` for that user.
- Job uses a timeout window of 2 hours.
- Structured logs include per-job and per-user outcomes.

## Operational Notes

- Graceful shutdown handles HTTP server, cron scheduler, and DB pool cleanup.
- Keep `UNOSEND_API_KEY` out of VCS.
- Rotate credentials immediately if exposed.
