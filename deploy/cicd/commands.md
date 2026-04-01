# CI/CD Commands

## Required tools

- GitHub CLI (`gh`) authenticated: `gh auth login`
- Docker / Docker Compose

## Local deploy

```powershell
Copy-Item .env.example .env
# edit .env values
docker compose --env-file .env up -d --build
docker compose --env-file .env ps
```

## GitHub repo setup

```powershell
gh repo view
gh workflow list
```

## Push env values from `.env` to GitHub Secrets

```powershell
pwsh -File ./deploy/set-github-secrets.ps1 -EnvFile ./.env
```

## Push env + deploy secrets together

```powershell
pwsh -File ./deploy/set-github-secrets.ps1 \
	-EnvFile ./.env \
	-ServerHost "<server-ip>" \
	-ServerUser "<server-user>" \
	-ServerPassword "<server-password>" \
	-ServerSshPort "22" \
	-ServerAppDir "/opt/email-verifier-api"
```

## Manual single-secret set examples

```powershell
gh secret set API_KEY --body "your-api-key"
gh secret set DB_PASSWORD --body "your-db-password"
```

## Trigger/check workflow runs

```powershell
gh workflow run deploy.yml
gh run list --workflow deploy.yml
gh run watch
```
