# Deploy & CI/CD

This folder contains deployment helpers and CI/CD setup commands.

## Files

- `cicd/commands.md`: one-shot GitHub CLI commands for repo setup and workflow checks.
- `cicd/github-actions.workflow.example.yml`: sample GitHub Actions workflow.
- `set-github-secrets.ps1`: PowerShell script that reads `.env` and writes secrets to GitHub.

## Quick Use

1. Copy `.env.example` to `.env` and fill real values.
2. Run `docker compose --env-file .env up -d --build`.
3. Push env values to GitHub Secrets using:
   - `pwsh -File ./deploy/set-github-secrets.ps1 -EnvFile ./.env`
4. Copy `deploy/cicd/github-actions.workflow.example.yml` to `.github/workflows/ci.yml` when ready.
