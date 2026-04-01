# Email Verifier V2

Email Verifier V2 is a verifier-only system built with Go, Fiber, PostgreSQL, and React. It classifies addresses using direct SMTP callouts to recipient MX hosts, a cached control-recipient baseline per domain, and an evidence-backed enrichment layer for ambiguous cases.

## What Changed

V2 removes the old probe-email and bounce-monitoring model entirely:

- no Tor sidecar
- no Gmail/app-password submission path for verification
- no probe templates or SMTP account pools
- no IMAP bounce watching
- no delayed "no bounce means valid" logic

The new classifier returns one of:

- `deliverable`
- `undeliverable`
- `accept_all`
- `unknown`

## How V2 Works

1. Normalize and validate the email address.
2. Resolve MX records, with A/AAAA fallback when no MX exists.
3. Perform direct SMTP RCPT callouts against recipient MX hosts.
4. If the target recipient is accepted, run or reuse a cached control-recipient callout for the same domain fingerprint.
5. Classify the result and attach protocol evidence.
6. For `accept_all` and `unknown`, run asynchronous enrichment to improve confidence and summarize risk.

## API Overview

- `POST /auth/register`
- `POST /auth/login`
- `GET /users/me`
- `POST /users/api-key/regenerate`
- `POST /verifications`
- `POST /verifications/batch`
- `POST /verifications/import-csv`
- `GET /verifications`
- `GET /verifications/:id`
- `GET /verifications/stats`
- `DELETE /verifications/:id`
- `GET /health`

Admin:

- `GET /admin/users`
- `PUT /admin/users/:id`
- `DELETE /admin/users/:id`
- `GET /admin/verifications`
- `DELETE /admin/verifications/:id`

## Local Development

### Backend

```bash
go test ./...
go run ./cmd/api
```

### Frontend

```bash
cd web
npm install
npm run build
npm run dev
```

### Docker Compose

```bash
docker-compose up -d --build
```

This starts:

- `postgres`
- `api`
- `web`

## Documentation

- [Legacy risks](./docs/legacy-risks.md)
- [V2 architecture](./docs/v2-architecture.md)
- [ADR: retire probe/bounce/Tor](./docs/adr/0001-retire-probe-bounce-tor.md)
