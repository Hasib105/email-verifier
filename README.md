# Email Verifier API

A privacy-focused email verification API built with Go and [Fiber](https://gofiber.io/), routing all SMTP checks through the **Tor network** to protect your server's IP address.

## Features

- **Tor-routed SMTP verification** — all outbound connections go through Tor SOCKS5 proxy
- **Syntax validation** — rejects malformed email addresses
- **Disposable domain detection** — flags throwaway email providers
- **MX record lookup** — checks that the domain can receive mail
- **SMTP RCPT TO verification** — confirms the mailbox exists on the mail server
- **Greylisting detection** — identifies temporary rejections (450/451)
- **STARTTLS support** — upgrades to TLS when the mail server supports it
- **API key authentication** — simple header-based auth
- **Concurrency control** — limits simultaneous Tor connections
- **Docker Compose deployment** — one-command setup with Tor sidecar

## Project Structure

```
email-verifier-api/
├── cmd/api/main.go              # Application entrypoint
├── internal/
│   ├── config/config.go         # Environment-based configuration
│   ├── handler/verify.go        # HTTP handlers (verify, check-tor)
│   └── verifier/
│       ├── smtp.go              # SMTP email verification logic
│       └── tor_check.go         # Tor connectivity verification
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

This starts two containers:

| Service | Description | Port |
|---------|-------------|------|
| `tor` | Tor SOCKS5 proxy | 9050 (internal) |
| `api` | Email verifier API | 3000 (exposed) |

The API waits for Tor to become healthy before starting.

### Stop

```bash
docker-compose down
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

### Verify Email

```
POST /verify
Content-Type: application/json
X-API-Key: <your-api-key>
```

```bash
curl -X POST http://localhost:3000/verify \
  -H "Content-Type: application/json" \
  -H "X-API-Key: super-secret-key-123" \
  -d '{"email": "user@example.com"}'
```

#### Response Statuses

| Status | Meaning |
|--------|---------|
| `valid` | Mailbox exists (250 accepted) |
| `invalid` | Bad syntax, no MX records, or mailbox rejected (550-559) |
| `disposable` | Known disposable/temporary email domain |
| `greylisted` | Server returned temporary rejection (450/451) — retry later |
| `unknown` | Could not determine validity |
| `error` | Connection or SMTP protocol failure |

#### Example Responses

**Valid email:**
```json
{
  "status": "valid",
  "message": "250 Accepted",
  "email": "real-user@example.com"
}
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

## Configuration

All settings are configured via environment variables (set in `docker-compose.yml`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | API listen port |
| `API_KEY` | `super-secret-key-123` | Authentication key for `/verify` |
| `TOR_SOCKS_ADDR` | `tor:9050` | Tor SOCKS5 proxy address |
| `MAX_CONCURRENCY` | `5` | Max simultaneous SMTP connections |

> **Important:** Change `API_KEY` to a strong secret before deploying to production.

## How It Works

1. **Syntax check** — validates email format with regex
2. **Disposable check** — compares domain against a known list
3. **MX lookup** — resolves the domain's mail exchange records
4. **SMTP connection via Tor** — connects to the mail server's port 25 through the Tor SOCKS5 proxy
5. **EHLO + STARTTLS** — negotiates with the mail server, upgrading to TLS if available
6. **MAIL FROM + RCPT TO** — sends the sender and recipient commands to check if the mailbox exists
7. **Response parsing** — interprets SMTP response codes to determine validity

## Notes

- **Rate limiting by mail servers:** Some providers (e.g., Gmail) may reject connections from Tor exit nodes. This is expected behavior — the API will return an `error` status with the server's rejection message.
- **Tor is slow:** SMTP verification through Tor takes longer than direct connections (typically 10-45 seconds). The concurrency limiter prevents overloading.
- **Not 100% accurate:** Some mail servers accept all addresses (catch-all) or reject all during RCPT TO checks. Use results as a signal, not absolute truth.

## License

MIT
