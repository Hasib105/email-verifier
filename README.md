# Email Verifier API

A privacy-focused email verification API built with Go and [Fiber](https://gofiber.io/), routing all SMTP checks through the **Tor network** to protect your server's IP address.

## Features

- **Multi-tenant user management** — each user gets their own API key and isolated resources
- **Per-user webhook URLs** — referral/notification callbacks specific to each user
- **Tor-routed SMTP verification** — all outbound connections go through Tor SOCKS5 proxy
- **Syntax validation** — rejects malformed email addresses
- **Disposable domain detection** — flags throwaway email providers
- **MX record lookup** — checks that the domain can receive mail
- **SMTP RCPT TO verification** — confirms the mailbox exists on the mail server
- **Fallback real SMTP probe over Tor** — on uncertain result, sends a real probe email through Tor and tracks bounce outcome
- **Single scheduled bounce recheck** — automatic one-time bounce check after 6 hours
- **Persistent verification cache with sqlx** — stores status/history and returns cached result for repeated requests
- **PostgreSQL-backed storage** — uses latest PostgreSQL for verifications, events, and SMTP account usage tracking
- **IMAP bounce detection** — checks your mailbox for DSN/bounce messages matching token/recipient
- **Webhook notifications** — push status transitions to external systems (supports per-user URLs)
- **SMTP account pool** — attach multiple SMTP accounts per user, auto-pick least-used account, enforce daily limit per account
- **CSV import API** — bulk verify emails from uploaded CSV files
- **JSON batch verify API** — verify up to 1000 emails per request
- **Greylisting detection** — identifies temporary rejections (450/451)
- **STARTTLS support** — upgrades to TLS when the mail server supports it
- **API key authentication** — per-user header-based auth
- **Concurrency control** — limits simultaneous Tor connections
- **Docker Compose deployment** — one-command setup with Tor sidecar
- **Swagger API documentation** — interactive API docs at `/swagger/`
- **Terminal CLI for signup** — create users via command line
- **React dashboard** — modern frontend for managing verification, SMTP, templates, and webhook settings

## Project Structure

```
email-verifier-api/
├── cmd/
│   ├── api/main.go              # API server entrypoint
│   └── cli/main.go              # CLI tool for user management
├── docs/                        # Swagger docs + API integration guides
├── web/                         # React 19 + Vite frontend dashboard
├── internal/
│   ├── config/config.go         # Environment-based configuration
│   ├── handler/verify.go        # HTTP handlers
│   ├── repo/                    # Database repositories
│   ├── service/                 # Business logic services
│   ├── store/models.go          # Data models
│   └── verifier/                # SMTP verification logic
├── Dockerfile                   # Multi-stage Go build
├── tor.Dockerfile               # Alpine-based Tor proxy
├── docker-compose.yml           # Full stack orchestration
├── torrc                        # Tor daemon configuration
├── go.mod / go.sum              # Go module files
└── tools.go                     # Build tool dependencies
```

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & Docker Compose

### Run

```bash
docker-compose up -d --build
```

This starts three containers:

| Service | Description | Port |
|---------|-------------|------|
| `postgres` | PostgreSQL database | 5432 (internal) |
| `tor` | Tor SOCKS5 proxy | 9050 (internal) |
| `api` | Email verifier API | 3000 (exposed) |

The API waits for Tor to become healthy before starting.

### Create a User (Terminal Signup)

Build and run the CLI tool to create users:

```bash
# Build the CLI
go build -o verifier-cli ./cmd/cli

# Run signup
./verifier-cli signup
```

Interactive prompts:
```
=== Email Verifier - User Signup ===

Enter your name: John Doe
Enter your email: john@example.com
Enter webhook URL (optional, press Enter to skip): https://myapp.com/webhooks/email

=== User Created Successfully! ===

User ID:     a1b2c3d4-e5f6-7890-abcd-ef1234567890
Name:        John Doe
Email:       john@example.com
Webhook URL: https://myapp.com/webhooks/email

=== Your API Key (save this securely!) ===

  evk_abc123def456...

Use this API key in the X-API-Key header for all API requests.
```

### List Users

```bash
./verifier-cli list-users
```

### Stop

```bash
docker-compose down
```

### Frontend Dashboard (React 19)

Run the API first, then start the web dashboard:

```bash
cd web
cp .env.example .env
npm install
npm run dev
```

Open: `http://localhost:5173`

Production build:

```bash
npm run build
npm run preview
```

### Env + Deploy Helpers

- Copy `.env.example` to `.env` and set real values.
- See `deploy/README.md` for deploy and CI/CD helpers.
- Upload all `.env` keys to GitHub Secrets with:

```powershell
pwsh -File ./deploy/set-github-secrets.ps1 -EnvFile ./.env
```

## API Documentation

Interactive Swagger UI is available at:

```
http://localhost:3000/swagger/
```

## API Endpoints

### Health Check

```
GET /health
```

```bash
curl http://localhost:3000/health
# OK
```

### Tor Connectivity Check

Verifies that outbound traffic is routed through Tor.

```
GET /check-tor
```

```bash
curl http://localhost:3000/check-tor
```

```json
{
  "is_tor": true,
  "ip": "192.42.116.181",
  "message": "Traffic is routed through Tor network"
}
```

### Get Current User

```
GET /users/me
X-API-Key: <your-api-key>
```

```bash
curl http://localhost:3000/users/me \
  -H "X-API-Key: evk_your_api_key_here"
```

### Update Webhook URL

```
PUT /users/webhook
Content-Type: application/json
X-API-Key: <your-api-key>
```

```json
{
  "webhook_url": "https://myapp.com/webhooks/email-verified"
}
```

### Verify Email

```
POST /verify
Content-Type: application/json
X-API-Key: <your-api-key>
```

```bash
curl -X POST http://localhost:3000/verify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: evk_your_api_key_here" \
  -d '{"email": "user@example.com"}'
```

#### Response Statuses

| Status | Meaning |
|--------|---------|
| `valid` | Mailbox exists (250 accepted) |
| `invalid` | Bad syntax, no MX records, or mailbox rejected (550-559) |
| `disposable` | Known disposable/temporary email domain |
| `greylisted` | Server returned temporary rejection (450/451) — retry later |
| `pending_bounce_check` | Fallback probe sent; waiting for first/follow-up IMAP bounce checks |
| `bounced` | Bounce detected in mailbox; address considered invalid |
| `accepted_no_bounce` | No bounce found after follow-up window |
| `unknown` | Could not determine validity |
| `error` | Connection or SMTP protocol failure |

#### Example Responses

**Valid email:**
```json
{
  "id": "fe632dc1-95af-4dc6-9a88-a1543f6e595f",
  "status": "valid",
  "message": "250 Accepted",
  "email": "real-user@example.com",
  "source": "direct-smtp-check",
  "cached": false,
  "finalized": true
}
```

**Pending fallback result:**
```json
{
  "id": "5f7ea2f8-cf04-4888-b8d8-e50ef9040ac5",
  "status": "pending_bounce_check",
  "message": "probe sent; waiting for bounce check",
  "email": "user@example.com",
  "source": "smtp-probe",
  "cached": false,
  "finalized": false,
  "next_check_at": 1774879200
}
```

If the same email is verified again, the API returns the stored result immediately with `cached: true`.

### Batch Verify Emails (JSON)

Verify a list of emails in one request.

```
POST /verify/batch
Content-Type: application/json
X-API-Key: <your-api-key>
```

```bash
curl -X POST http://localhost:3000/verify/batch \
  -H "Content-Type: application/json" \
  -H "X-API-Key: evk_your_api_key_here" \
  -d '{"emails":["alice@example.com","not-an-email","bob@example.com"]}'
```

```json
{
  "total": 3,
  "accepted": 3,
  "items": [
    {
      "id": "2f06116d-4f3e-4f76-b671-71888fadb5f4",
      "email": "alice@example.com",
      "status": "valid",
      "message": "250 Accepted",
      "source": "direct-smtp-check",
      "cached": false,
      "finalized": true
    },
    {
      "id": "f1d4f67e-54d4-437e-a5d1-3f89c53f6ff9",
      "email": "not-an-email",
      "status": "invalid",
      "message": "invalid syntax",
      "source": "direct-smtp-check",
      "cached": false,
      "finalized": true
    }
  ]
}
```

Limits:

- Max `1000` emails per batch request.
- Each item is processed independently; one bad email does not fail the whole batch.
- `accepted` means successfully processed items, not only `valid` emails.

Integration guide for other services:

- `docs/batch-verify-api.md`

### Import CSV

Upload CSV where first column is email.

```
POST /verify/import-csv
Content-Type: multipart/form-data
X-API-Key: <your-api-key>
```

```bash
curl -X POST http://localhost:3000/verify/import-csv \
  -H "X-API-Key: evk_your_api_key_here" \
  -F "file=@emails.csv"
```

### Add SMTP Account

```
POST /smtp-accounts
Content-Type: application/json
X-API-Key: <your-api-key>
```

```json
{
  "host": "smtp.gmail.com",
  "port": 587,
  "username": "you@example.com",
  "password": "app-password",
  "sender": "you@example.com",
  "imap_host": "imap.gmail.com",
  "imap_port": 993,
  "imap_mailbox": "INBOX",
  "daily_limit": 100,
  "active": true
}
```

`username` and `password` are shared for both SMTP sending and IMAP bounce-check login.

### List SMTP Accounts

```
GET /smtp-accounts
X-API-Key: <your-api-key>
```

Returns each account with `sent_today` and `daily_limit`.

### Create Email Template

```
POST /email-templates
Content-Type: application/json
X-API-Key: <your-api-key>
```

```json
{
  "name": "default-template",
  "subject_template": "Email verification probe {{token}}",
  "body_template": "Hello,\n\nVerification probe for {{email}}.\nToken: {{token}}\nSender: {{sender}}\n",
  "active": true
}
```

Supported placeholders: `{{token}}`, `{{email}}`, `{{sender}}`.

### List Email Templates

```
GET /email-templates
X-API-Key: <your-api-key>
```

**Invalid syntax:**
```json
{
  "status": "invalid",
  "message": "invalid syntax",
  "email": "not-an-email"
}
```

**Disposable domain:**
```json
{
  "status": "disposable",
  "message": "disposable domain detected",
  "email": "test@mailinator.com"
}
```

## Webhook Notifications

Webhooks are sent to the user's configured webhook URL for these events:

| Event | Description |
|-------|-------------|
| `verify.created` | New verification request initiated |
| `verify.bounced` | Bounce detected during scheduled check |
| `verify.check.no_bounce` | Scheduled check completed with no bounce |
| `verify.check.error` | Error during scheduled bounce check |

**Webhook Payload:**
```json
{
  "event": "verify.bounced",
  "id": "ver-123",
  "email": "user@example.com",
  "status": "bounced",
  "message": "Mail delivery failed",
  "source": "smtp-probe",
  "user_id": "user-456",
  "check_count": 1,
  "finalized": true,
  "checked_at": 1711800000
}
```

## Configuration

All settings are configured via environment variables (set in `docker-compose.yml`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | API listen port |
| `TOR_SOCKS_ADDR` | `tor:9050` | Tor SOCKS5 proxy address |
| `MAX_CONCURRENCY` | `5` | Max simultaneous SMTP connections |
| `DATABASE_DSN` | *(empty)* | Full PostgreSQL connection string override (optional) |
| `DB_HOST` | `postgres` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL username |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `verifier` | PostgreSQL database name |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `WEBHOOK_URL` | *(empty)* | Default webhook endpoint (per-user URLs take precedence) |
| `WEBHOOK_TIMEOUT` | `10s` | HTTP timeout for webhook delivery |
| `CHECK_INTERVAL` | `1m` | Scheduler tick interval |
| `SECOND_BOUNCE_DELAY` | `6h` | Delay before single bounce recheck |

## How It Works

1. **User authentication** — validates API key from header, returns user context
2. **Syntax check** — validates email format with regex
3. **Disposable check** — compares domain against a known list
4. **MX lookup** — resolves the domain's mail exchange records
5. **SMTP connection via Tor** — connects to the mail server's port 25 through the Tor SOCKS5 proxy
6. **EHLO + STARTTLS** — negotiates with the mail server, upgrading to TLS if available
7. **MAIL FROM + RCPT TO** — sends the sender and recipient commands to check if the mailbox exists
8. **Fallback (if uncertain)** — sends a real SMTP probe email and marks status `pending_bounce_check`
9. **Background scheduler** — performs one IMAP bounce recheck after 6 hours
10. **SMTP account selection** — picks active account for user with lowest `sent_today` under daily limit (default 100)
11. **Template rendering** — if an active template exists for user, it is used for subject/body
12. **Persistence + webhook** — stores status/events and notifies user's webhook URL

## Notes

- **Rate limiting by mail servers:** Some providers (e.g., Gmail) may reject connections from Tor exit nodes. This is expected behavior — the API will return an `error` status with the server's rejection message.
- **Tor is slow:** SMTP verification through Tor takes longer than direct connections (typically 10-45 seconds). The concurrency limiter prevents overloading.
- **Not 100% accurate:** Some mail servers accept all addresses (catch-all) or reject all during RCPT TO checks. Use results as a signal, not absolute truth.

## License

MIT
