# Email Verifier API

Email verification API built with Go and [Fiber](https://gofiber.io/). This hardened V1 keeps the existing probe-and-bounce workflow, but removes Tor, makes direct SMTP results more conservative, and exposes confidence and evidence fields so weak signals are no longer presented as hard proof.

## What Changed In Hardened V1

- Tor is fully removed from runtime, config, and deployment.
- Direct SMTP is still attempted first when the host, or a configured SOCKS5 proxy, can reach recipient MX hosts on port 25.
- Direct acceptance is treated as a bounded signal, not guaranteed mailbox proof.
- Probe-and-bounce fallback remains in place for inconclusive cases.
- First no-bounce window upgrades a record to provisional `valid` while a second check remains scheduled.
- Results now include `confidence`, `deterministic`, `reason_code`, `verification_path`, `signal_summary`, and `expires_at`.
- Cached results expire and are re-verified instead of being reused indefinitely.

## Features

- Multi-tenant user management with per-user API keys and webhook URLs
- Direct SMTP verification with syntax checks, disposable detection, MX lookup, A/AAAA fallback, port-25 EHLO, optional SOCKS5 proxy pool, catch-all probing, STARTTLS, and provider-aware result handling
- SMTP probe fallback with scheduled IMAP bounce checks
- SMTP account pool with per-account daily limits
- Email templates for probe content
- Verification history, events, webhook delivery, CSV import, and batch verification
- React dashboard for verification, history, SMTP account management, templates, and admin views

## Status Model

The public statuses remain compatible with V1:

| Status | Meaning |
| --- | --- |
| `valid` | Direct SMTP accepted on a non-strict provider, or no bounce was observed during the first configured probe window |
| `invalid` | Syntax failure, no mail routing, explicit hard SMTP rejection, or bounce evidence matched the probe |
| `disposable` | Known disposable mailbox domain |
| `greylisted` | Recipient MX returned a temporary failure and the system queued probe fallback |
| `pending_bounce_check` | Probe was sent and bounce evidence is still pending |
| `unknown` | Direct SMTP was policy-blocked, verification-disabled, or otherwise inconclusive |
| `error` | Verification infrastructure failed or probe fallback could not complete |

The new additive evidence fields make those statuses safer to interpret:

- `confidence`: `high`, `medium`, or `low`
- `deterministic`: whether the result comes from a hard signal
- `reason_code`: machine-readable primary explanation
- `verification_path`: `direct_smtp`, `probe_bounce`, or `hybrid`
- `signal_summary`: short evidence summary meant for users and logs
- `expires_at`: Unix timestamp after which the cached result should be refreshed

## Quick Start

### Prerequisites

- Docker and Docker Compose

### Run

```bash
docker-compose up -d --build
```

This starts:

| Service | Description | Port |
| --- | --- | --- |
| `postgres` | PostgreSQL database | internal |
| `api` | Email verifier API | `3000` |
| `web` | React dashboard | `80` |

### Create A User

```bash
go build -o verifier-cli ./cmd/cli
./verifier-cli signup
```

### Create A Superuser In Docker

Build the image, then run the CLI entrypoint from the same runtime image:

```bash
docker build -t email-verifier-api .
docker run --rm -it --entrypoint ./cli email-verifier-api createsuperuser
```

### Frontend

```bash
cd web
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173).

## Core API

### Health

```http
GET /health
```

Example:

```json
{
  "status": "ok",
  "mode": "v1-hardened",
  "direct_smtp_status": "available",
  "last_checked_at": 1775000000,
  "message": "250 recipient accepted",
  "verifier_mail_from": "verify@localhost",
  "verifier_ehlo_domain": "localhost"
}
```

### Verify Email

```http
POST /verify
Content-Type: application/json
X-API-Key: <your-api-key>
```

```json
{
  "email": "user@example.com"
}
```

Example direct result:

```json
{
  "id": "fe632dc1-95af-4dc6-9a88-a1543f6e595f",
  "status": "valid",
  "message": "250 recipient accepted",
  "email": "user@example.com",
  "source": "direct-smtp-check",
  "cached": false,
  "finalized": true,
  "confidence": "medium",
  "deterministic": false,
  "reason_code": "direct_accept_non_strict",
  "verification_path": "direct_smtp",
  "signal_summary": "Recipient MX accepted RCPT on a non-strict provider.",
  "expires_at": 1775259200
}
```

Example fallback result:

```json
{
  "id": "5f7ea2f8-cf04-4888-b8d8-e50ef9040ac5",
  "status": "pending_bounce_check",
  "message": "probe sent via smtp account smtp-123; waiting for bounce window",
  "email": "user@example.com",
  "source": "smtp-probe",
  "cached": false,
  "finalized": false,
  "next_check_at": 1774879200,
  "confidence": "low",
  "deterministic": false,
  "reason_code": "probe_sent_waiting_bounce",
  "verification_path": "hybrid",
  "signal_summary": "Direct SMTP evidence was insufficient, so a probe was sent via SMTP account smtp-123 and the system is waiting for bounce evidence.",
  "expires_at": 1774900800
}
```

### Batch Verify

```http
POST /verify/batch
Content-Type: application/json
X-API-Key: <your-api-key>
```

Body:

```json
{
  "emails": ["alice@example.com", "bob@example.com"]
}
```

Batch size limit: `1000`.

### CSV Import

```http
POST /verify/import-csv
Content-Type: multipart/form-data
X-API-Key: <your-api-key>
```

### Verification History

- `GET /verifications`
- `GET /verifications/:id`
- `GET /verifications/stats`

### Webhooks

- `PUT /users/webhook`
- `POST /users/webhook/test`

For async states like `pending_bounce_check`, and provisional probe results with `finalized=false`, use webhook events as the primary update channel and keep `GET /verifications/:id` for reconciliation only.
Webhook notifications apply to results created by both `POST /verify` and `POST /verify/batch`.
When a verification status changes later (for example, `pending_bounce_check` -> `valid`, or provisional `valid` -> `invalid`), the service triggers another webhook event for the same verification `id` so your system can update status immediately.

### SMTP Accounts And Templates

- `POST /smtp-accounts`
- `GET /smtp-accounts`
- `GET /smtp-accounts/:id`
- `PUT /smtp-accounts/:id`
- `DELETE /smtp-accounts/:id`
- `POST /email-templates`
- `GET /email-templates`
- `GET /email-templates/:id`
- `PUT /email-templates/:id`
- `DELETE /email-templates/:id`

## Probe And Bounce Semantics

Hardened V1 still supports the legacy fallback flow, but its interpretation is stricter:

1. Direct SMTP runs first.
2. If direct SMTP is inconclusive, policy-blocked, greylisted, or unavailable, the service sends a probe through a configured SMTP account.
3. The first no-bounce check marks the record `valid`, with `finalized=false` while a second check remains scheduled.
4. A detected bounce finalizes as `invalid`.
5. If the second no-bounce window also passes, the record remains `valid`, with `finalized=true`, `confidence=low`, and `deterministic=false`.

The absence of a bounce is treated as heuristic evidence, not mailbox proof.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `3000` | API listen port |
| `MAX_CONCURRENCY` | `5` | Max concurrent direct SMTP checks |
| `TIMEOUT` | `20s` | Timeout for direct SMTP attempts |
| `VERIFIER_MAIL_FROM` | `verify@localhost` | Envelope sender used for direct SMTP callouts |
| `VERIFIER_EHLO_DOMAIN` | `localhost` | EHLO hostname used for direct SMTP callouts |
| `SMTP_PROXY_POOL` | empty | Optional comma or space separated SOCKS5 proxy list for outbound MX port-25 checks |
| `DATABASE_DSN` | empty | Full PostgreSQL DSN override |
| `DB_HOST` | `postgres` | PostgreSQL host |
| `DB_PORT` | `15432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL username |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `verifier` | PostgreSQL database name |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `WEBHOOK_URL` | empty | Default webhook URL |
| `WEBHOOK_TIMEOUT` | `10s` | Webhook HTTP timeout |
| `CHECK_INTERVAL` | `15s` | Scheduler tick interval |
| `FIRST_BOUNCE_DELAY` | `1m` | Delay before first IMAP bounce check |
| `SECOND_BOUNCE_DELAY` | `6h` | Delay before second IMAP bounce check |
| `HARD_RESULT_TTL` | `168h` | TTL for `invalid` and `disposable` |
| `DIRECT_VALID_TTL` | `72h` | TTL for direct `valid` results |
| `PROBE_VALID_TTL` | `24h` | TTL for probe-derived `valid` results |
| `TRANSIENT_RESULT_TTL` | `6h` | TTL for `unknown`, `greylisted`, `error`, and pending states |

## Deployment Notes

- This branch no longer includes a Tor sidecar.
- Direct SMTP still depends on the host being able to reach recipient MX servers on port 25.
- If `SMTP_PROXY_POOL` is set, direct SMTP callouts rotate through that proxy pool first. Leave it empty to use the server machine's own network path.
- If a proxy cannot establish the SMTP/EHLO path, the verifier falls back to direct machine egress.
- Catch-all domains are handled by direct SMTP control-recipient probing and returned as `valid` with `reason_code=catch_all_domain`.
- If direct SMTP is blocked by infrastructure, the system degrades to probe-first behavior and keeps the result grounded with low-confidence metadata.

## Additional Docs

- [Batch verify API guide](./docs/batch-verify-api.md)
- [Webhook integration guide](./docs/webhook-api.md)
- [Deploy notes](./deploy/README.md)

## License

MIT
