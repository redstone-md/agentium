$ErrorActionPreference = "Stop"

param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$TargetUrl = "https://example.com"
)

Write-Host "Health check..."
$health = Invoke-RestMethod -Uri "$BaseUrl/healthz" -Method Get
$health | ConvertTo-Json -Depth 5

Write-Host "Creating session..."
$session = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions" -Method Post -ContentType "application/json" -Body (@{
  locale = "en-US"
} | ConvertTo-Json)
$session | ConvertTo-Json -Depth 5

$sessionId = $session.session_id
if (-not $sessionId) {
  throw "session_id was empty"
}

try {
  Write-Host "Navigating..."
  $navigate = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "navigate"
    value  = $TargetUrl
  } | ConvertTo-Json)
  $navigate | ConvertTo-Json -Depth 8

  Write-Host "Waiting for network idle..."
  $idle = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "wait_network_idle"
  } | ConvertTo-Json)
  $idle | ConvertTo-Json -Depth 8

  Write-Host "Fetching snapshot..."
  $snapshot = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/snapshot" -Method Get
  $snapshotJson = $snapshot | ConvertTo-Json -Depth 8 -Compress
  $snapshotBytes = [Text.Encoding]::UTF8.GetByteCount($snapshotJson)
  Write-Host "Snapshot size: $snapshotBytes bytes"
  if ($snapshotBytes -gt 20480) {
    throw "snapshot exceeded 20KB limit: $snapshotBytes bytes"
  }
  $snapshot | ConvertTo-Json -Depth 8
}
finally {
  Write-Host "Closing session..."
  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId" -Method Delete | Out-Null
}
