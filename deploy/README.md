# Deploy & CI/CD

This folder contains deployment helpers and CI/CD setup commands.

## Files

- `cicd/commands.md`: one-shot GitHub CLI commands for repo setup and workflow checks.
- `cicd/github-actions.workflow.example.yml`: sample GitHub Actions workflow.
- `set-github-secrets.ps1`: PowerShell script that reads `.env` and writes secrets to GitHub.
- `docker-compose.prod.yml`: production compose file that pulls pre-built images from GHCR.

## Quick Use

1. Copy `.env.example` to `.env` and fill real values.
   - If host port `80` is already used on your server, set `WEB_PORT` (default is `8080`).
2. Run `docker compose --env-file .env up -d --build`.
3. Push env values to GitHub Secrets using:
   - `pwsh -File ./deploy/set-github-secrets.ps1 -EnvFile ./.env`
4. Copy `deploy/cicd/github-actions.workflow.example.yml` to `.github/workflows/ci.yml` when ready.

## Main-Branch Deploy Flow

When pushing to `main`, `.github/workflows/deploy.yml` will:

1. Build and push images in GitHub Actions runner:
   - `ghcr.io/<owner>/<repo>-api:latest`
   - `ghcr.io/<owner>/<repo>-web:latest`
2. SSH into your server.
3. Pull images and restart services using:
   - `docker compose --env-file .env -f deploy/docker-compose.prod.yml pull`
   - `docker compose --env-file .env -f deploy/docker-compose.prod.yml up -d`

## Required GitHub Secrets

- `SERVER_HOST`
- `SERVER_USER`
- `SERVER_PASSWORD`
- `SERVER_SSH_PORT`
- `SERVER_APP_DIR` (optional, default is `~/email-verifier-api`)

Your application `.env` values should also exist as repo secrets (for example `DB_PASSWORD`, `VERIFIER_MAIL_FROM`, and `VERIFIER_EHLO_DOMAIN`).

GHCR auth is automatic in deploy workflow:

- username: `${{ github.actor }}`
- token: `${{ github.token }}`

## V2 Runtime Notes

- Production no longer runs a Tor sidecar.
- The API now performs direct SMTP callouts and optional enrichment only.
- Make sure the server can egress to recipient MX hosts on TCP/25 and that `VERIFIER_EHLO_DOMAIN` and `VERIFIER_MAIL_FROM` are set to values you control.

## One Command To Upload Secrets

```powershell
pwsh -File ./deploy/set-github-secrets.ps1 \
  -EnvFile ./.env \
  -ServerHost "<server-ip>" \
  -ServerUser "<server-user>" \
  -ServerPassword "<server-password>" \
  -ServerSshPort "22" \
   -ServerAppDir "/opt/email-verifier-api"
```
