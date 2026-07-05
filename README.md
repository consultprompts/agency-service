# Agency Service — consultprompts.com

Manages B2B client leads for custom web design / agency work. Owns mockup
requests submitted through the site and their status (pending/completed).
Sits behind the API Gateway — never exposed directly to the internet.

---

## What this service owns

- Lead capture — name, email, business, message, selected package
- Lead status tracking (`pending` / `completed`)
- Email notification to the site owner when a new lead is submitted
  (via Resend; optional — disabled when the env vars aren't set)
- Milestone tracking for custom website builds — admins create/update/delete
  milestones on a lead; the lead's submitter can view their own project's
  milestones (`pending` / `in_progress` / `completed`)
- Nothing else yet (onboarding questionnaires and full project management
  are future additions per the original architecture plan)

## What this service does NOT do

- No JWT verification — trusts `X-User-ID` / `X-User-Roles` headers set
  by the API Gateway. Must never be reachable directly from outside the
  Docker network in production.
- No account-related email (that's auth-service's job) — this service only
  sends new-lead notifications.
- No payment processing (that's Order & Payment service's job).

---

## Tech Stack

- Go + Gin (HTTP routing)
- PostgreSQL via `pgx`/`pgxpool` — own database (`agency-database`),
  fully separate instance from auth-service's Postgres
- `golang-migrate` — automated migrations, run on startup
- Docker + Docker Compose

---

## Architecture

```
HTTP request → Handler → Service → Repository → Postgres
```

Same layered pattern as `auth-service`:
- **Handler** — HTTP layer only, parses/validates JSON, standardized response shape
- **Service** — business logic (status validation)
- **Repository** — SQL only

## Trust model

This service sits entirely behind the API Gateway. It never verifies JWTs
itself — the Gateway already did that and forwards two trusted headers:

- `X-User-ID` — the authenticated user's ID
- `X-User-Roles` — comma-separated role list (e.g. `student,admin`)

`internal/middleware/trusted_headers.go` reads these headers and rejects
requests missing them (401) or lacking the required role for admin routes
(403). If this service is ever exposed directly to the internet without
the Gateway in front, these headers can be spoofed by anyone — it must
always run on a private network reachable only through the Gateway.

---

## Project Structure

```
agency-service/
  main.go                        # entry point, dependency wiring
  database/
    db.go                        # Postgres connection pool + golang-migrate runner
  internal/
    handler/
      response.go                # standardized {success, data, error} response helpers
      lead_handler.go             # HTTP handlers for lead endpoints
      milestone_handler.go        # HTTP handlers for milestone endpoints
    service/
      lead_service.go             # business logic, status validation, notification trigger
      milestone_service.go        # milestone logic, ownership checks
    repository/
      lead_repository.go          # SQL queries (pgx)
      milestone_repository.go
    model/
      lead.go                     # Lead struct
      milestone.go                # Milestone struct
    middleware/
      trusted_headers.go          # RequireUserID, RequireAdminRole, IsAdmin
    email/
      email.go                    # Resend client, new-lead notification
  migrations/
    0001_init.up.sql / .down.sql
  .env                            # local secrets (gitignored)
  .env.example
  Dockerfile
  .dockerignore
```

---

## Data Model

| Table | Description |
|-------|-------------|
| `leads` | id, user_id, name, email, business, message (nullable), package (nullable), status, created_at |
| `milestones` | id, lead_id (FK → leads, cascade delete), title, description (nullable), status, sort_order, due_date (nullable), completed_at (nullable, set automatically on status transitions), created_at |

---

## Endpoints

| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | /healthz | No | Health check with DB connectivity verification |
| POST | /agency/leads | Yes (any authenticated user) | Submit a mockup request |
| GET | /agency/leads | Yes (admin only) | List all leads |
| PATCH | /agency/leads/:id/status | Yes (admin only) | Update lead status |
| GET | /agency/leads/:id/milestones | Yes (lead owner or admin) | List a lead's milestones |
| POST | /agency/leads/:id/milestones | Yes (admin only) | Create a milestone on a lead |
| PATCH | /agency/milestones/:id | Yes (admin only) | Partially update a milestone (title, description, status, sort_order, due_date) |
| DELETE | /agency/milestones/:id | Yes (admin only) | Delete a milestone |

All routes except `/healthz` require the request to have passed through
the API Gateway's JWT verification — this service reads `X-User-ID` and
`X-User-Roles` from the request headers rather than checking a token itself.

### Response Shape

Matches auth-service's convention:

```json
// Success
{
  "success": true,
  "data": { ... }
}

// Error
{
  "success": false,
  "error": {
    "code": "INVALID_INPUT",
    "message": "..."
  }
}
```

---

## Setup (Local Development)

**Prerequisites**: Go 1.26+, PostgreSQL

**1. Create `.env` from template:**
```bash
cp .env.example .env
```

**2. Create a local Postgres database** named `agency-database`, update
`.env` with your credentials (`DB_HOST=localhost` for local dev).

**3. Run:**
```bash
go run .
```

Migrations run automatically on startup.

---

## Running with Docker

This service is included in `api-gateway`'s `docker-compose.yml`, which
runs the entire stack (both Postgres instances, auth-service,
agency-service, and the Gateway) together.

From the `api-gateway` repo:
```bash
docker compose up --build
```

**Ports**:
- `agency-service` — `8082` (internal + host-mapped)
- `agency-postgres` — host-mapped to `5434` (container's internal port stays `5432`)

> **Note**: `DB_HOST`/`DB_PORT` in this service's `.env` are only used for
> local (non-Docker) development. When running via Docker Compose, the
> Gateway's `docker-compose.yml` overrides `DB_HOST` to `agency-postgres`
> regardless of what's in `.env`. To connect an external tool (DataGrip,
> pgAdmin) to this database from your host machine, use `localhost:5434`
> — not `5432`, which is the internal Docker-network port.

---

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| PORT | Port the service listens on | 8082 |
| DB_HOST | Postgres host (local dev only — Docker overrides this) | localhost |
| DB_PORT | Postgres port (local dev only) | 5434 |
| DB_USER | Postgres username | postgres |
| DB_PASSWORD | Postgres password | yourpassword |
| DB_NAME | Database name | agency-database |
| DB_SSLMODE | SSL mode | disable |
| RESEND_API_KEY | Resend API key for lead notifications (optional) | re_... |
| RESEND_FROM | From address for notification emails (optional) | noreply@consultprompts.com |
| LEAD_NOTIFICATION_EMAIL | Where new-lead notifications are sent (optional) | you@example.com |

Lead email notifications require all three `RESEND_*`/`LEAD_NOTIFICATION_EMAIL`
variables — if any is missing, the service starts normally with notifications
disabled (logged as a warning at startup). Sending happens in a background
goroutine after the lead is stored, so email failures never fail lead creation.

---

## TODO

- [ ] Automated tests (service layer, mirroring auth-service's approach)

