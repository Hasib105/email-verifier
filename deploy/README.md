# Deploy & CI/CD

This folder contains deployment helpers and CI/CD setup commands.

## Files

- `cicd/commands.md`: one-shot GitHub CLI commands for repo setup and workflow checks.
- `cicd/github-actions.workflow.example.yml`: sample GitHub Actions workflow.
- `set-github-secrets.ps1`: PowerShell script that reads `.env` and writes secrets to GitHub.
- `docker-compose.prod.yml`: production compose file that pulls pre-built images from GHCR.

## Quick Use

1. Copy `.env.example` to `.env` and fill real values.
2. Run `docker compose --env-file .env up -d --build`.
3. Push env values to GitHub Secrets using:
   - `pwsh -File ./deploy/set-github-secrets.ps1 -EnvFile ./.env`
4. Copy `deploy/cicd/github-actions.workflow.example.yml` to `.github/workflows/ci.yml` when ready.

## Main-Branch Deploy Flow

When pushing to `main`, `.github/workflows/deploy.yml` will:

1. Build and push images in GitHub Actions runner:
   - `ghcr.io/<owner>/<repo>-api:latest`
   - `ghcr.io/<owner>/<repo>-web:latest`
   - `ghcr.io/<owner>/<repo>-tor:latest`
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

Your application `.env` values should also exist as repo secrets (for example `API_KEY`, `DB_PASSWORD`, etc.).

GHCR auth is automatic in deploy workflow:

- username: `${{ github.actor }}`
- token: `${{ github.token }}`

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
