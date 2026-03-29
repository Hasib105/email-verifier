param(
  [string]$EnvFile = ".env",
  [switch]$SkipEmpty = $true
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
  exit 0
}

Write-Host "Uploading $($pairs.Count) secrets to GitHub..."
foreach ($key in $pairs.Keys) {
  $value = $pairs[$key]
  $value | gh secret set $key --body-file -
  Write-Host "Set secret: $key"
}

Write-Host "Done."
