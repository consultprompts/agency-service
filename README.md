# Agency Service — consultprompts.com

Manages B2B client leads for custom web design / agency work. Owns mockup
requests submitted through the site and the full project workflow — milestones,
mockup review, payment, and launch.
Sits behind the API Gateway — never exposed directly to the internet.

---

## What this service owns

- Lead capture — name, email, business, selected package, and the project
  brief (goals, pages, branding, timeline, …)
- Lead status tracking (`pending` / `accepted` / `completed` / `launched`)
- Project milestone tracking as a single `milestone_index` int per lead,
  walking a fixed stage list shared with the frontend
  (`website/src/lib/milestones.ts` ↔ the `core*` constants in
  `lead_service.go`): Designing Your Website → Design Ready for Your Review
  → Design Approved → Building Your Website → Website Ready → Payment →
  Waiting for Launch → Website Is Live, optionally preceded by Discovery
  Call Completed when the lead requested a call
- The client-facing project workflow: mockup delivery & review
  (accept / request changes), site completion, payment recording, and launch
- Email notifications via Resend (optional — disabled when the env vars
  aren't set): new-lead alerts to the site owner, plus client emails for
  acceptance, mockup delivery, revision requests, payment requests,
  receipts, and launch

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
    service/
      lead_service.go             # business logic, milestone flow, notification triggers
    repository/
      lead_repository.go          # SQL queries (pgx)
    model/
      lead.go                     # Lead struct
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
| `leads` | id, user_id, name, email, business, message, package, status, milestone_index, wants_call, project-brief fields (site_goal, pages_needed, branding, timeline, …), workflow fields (mockup_url, revision_feedback, site_url), payment fields (is_paid, paid_at, payment_amount, wants_maintenance, domain_renewal_date), created_at |

---

## Endpoints

| Method | Path | Auth Required | Description |
|--------|------|---------------|-------------|
| GET | /healthz | No | Health check with DB connectivity verification |
| POST | /agency/leads | Yes (any authenticated user) | Submit a mockup request |
| GET | /agency/leads/mine | Yes (any authenticated user) | List the caller's own leads |
| POST | /agency/leads/:id/review | Yes (lead owner) | Accept the mockup or request changes |
| PATCH | /agency/leads/:id/maintenance | Yes (lead owner) | Toggle the monthly-maintenance preference |
| POST | /agency/leads/:id/pay | Yes (lead owner) | Record payment; advances milestone to Waiting for Launch |
| GET | /agency/leads | Yes (admin only) | List all leads |
| PATCH | /agency/leads/:id/milestone | Yes (admin only) | Set a lead's milestone_index |
| PATCH | /agency/leads/:id/mockup | Yes (admin only) | Save the mockup URL and notify the client to review |
| PATCH | /agency/leads/:id/complete | Yes (admin only) | Mark the site ready and email the client to pay |
| PATCH | /agency/leads/:id/launch | Yes (admin only) | Set the live site URL and mark the lead launched (requires payment) |

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

