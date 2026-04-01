param(
  [string]$EnvFile = ".env",
  [switch]$SkipEmpty = $true,
  [string]$ServerHost = "",
  [string]$ServerUser = "",
  [string]$ServerPassword = "",
  [string]$ServerSshPort = "22",
  [string]$ServerAppDir = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if (-not (Test-Path $EnvFile)) {
  throw "Env file not found: $EnvFile"
}

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
  throw "GitHub CLI 'gh' is not installed or not in PATH."
}

$lines = Get-Content -Path $EnvFile
$pairs = @{}

foreach ($line in $lines) {
  $trimmed = $line.Trim()
  if ($trimmed -eq "" -or $trimmed.StartsWith("#")) { continue }

  $parts = $trimmed -split "=", 2
  if ($parts.Count -ne 2) { continue }

  $key = $parts[0].Trim()
  $value = $parts[1]

  if ($key -eq "") { continue }
  if ($SkipEmpty -and [string]::IsNullOrWhiteSpace($value)) { continue }

  $pairs[$key] = $value
}

if ($pairs.Count -eq 0) {
  Write-Host "No env entries found to upload."
}

if (-not [string]::IsNullOrWhiteSpace($ServerHost)) {
  $pairs["SERVER_HOST"] = $ServerHost
}
if (-not [string]::IsNullOrWhiteSpace($ServerUser)) {
  $pairs["SERVER_USER"] = $ServerUser
}
if (-not [string]::IsNullOrWhiteSpace($ServerPassword)) {
  $pairs["SERVER_PASSWORD"] = $ServerPassword
}
if (-not [string]::IsNullOrWhiteSpace($ServerSshPort)) {
  $pairs["SERVER_SSH_PORT"] = $ServerSshPort
}
if (-not [string]::IsNullOrWhiteSpace($ServerAppDir)) {
  $pairs["SERVER_APP_DIR"] = $ServerAppDir
}

if ($pairs.Count -eq 0) {
  Write-Host "No secrets found to upload."
  exit 0
}

Write-Host "Uploading $($pairs.Count) secrets to GitHub..."
foreach ($key in $pairs.Keys) {
  $value = $pairs[$key]
  # Use stdin input since some gh versions do not support --body-file.
  $null = $value | gh secret set $key
  if ($LASTEXITCODE -ne 0) {
    throw "Failed to set secret: $key"
  }
  Write-Host "Set secret: $key"
}

Write-Host "Done."
