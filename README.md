# Agency Service — consultprompts.com

Manages B2B client leads for custom web design / agency work. Owns mockup
requests submitted through the site and their status (pending/completed).
Sits behind the API Gateway — never exposed directly to the internet.

---

## What this service owns

- Lead capture — name, email, business, message, selected package
- Lead status tracking (`pending` / `completed`)
- Nothing else yet (onboarding questionnaires, milestone tracking, and
  project management are future additions per the original architecture
  plan — not built in this v1)

## What this service does NOT do

- No JWT verification — trusts `X-User-ID` / `X-User-Roles` headers set
  by the API Gateway. Must never be reachable directly from outside the
  Docker network in production.
- No email sending (that's auth-service's job for account-related email;
  lead notifications aren't built yet).
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
    service/
      lead_service.go             # business logic, status validation
    repository/
      lead_repository.go          # SQL queries (pgx)
    model/
      lead.go                     # Lead struct
    middleware/
      trusted_headers.go          # RequireUserID, RequireAdminRole
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

---

## Endpoints

| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | /healthz | No | Health check with DB connectivity verification |
| POST | /agency/leads | Yes (any authenticated user) | Submit a mockup request |
| GET | /agency/leads | Yes (admin only) | List all leads |
| PATCH | /agency/leads/:id/status | Yes (admin only) | Update lead status |

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

---

## TODO

- [ ] Onboarding questionnaire (per original architecture scope)
- [ ] Milestone tracking for custom website builds
- [ ] Email notification when a new lead is submitted
- [ ] Automated tests (service layer, mirroring auth-service's approach)
- [ ] Pagination on `GET /agency/leads` (fine for now at low volume)

