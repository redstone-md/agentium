param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$TargetUrl = "https://www.browserscan.net/"
)

$ErrorActionPreference = "Stop"

function Wait-ForHealth {
  param(
    [string]$Url,
    [int]$Attempts = 30,
    [int]$DelaySeconds = 1
  )

  for ($i = 0; $i -lt $Attempts; $i++) {
    try {
      return Invoke-RestMethod -Uri "$Url/healthz" -Method Get
    }
    catch {
      Start-Sleep -Seconds $DelaySeconds
    }
  }

  throw "Health endpoint did not become ready in time: $Url/healthz"
}

Wait-ForHealth -Url $BaseUrl | Out-Null

$session = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions" -Method Post -ContentType "application/json" -Body (@{
  locale = "en-US"
} | ConvertTo-Json)

$sessionId = $session.session_id
if (-not $sessionId) {
  throw "session_id was empty"
}

try {
  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "navigate"
    value  = $TargetUrl
  } | ConvertTo-Json) | Out-Null

  Start-Sleep -Seconds 12

  $snapshot = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/snapshot" -Method Get
  "=== Initial Snapshot ==="
  $snapshot | ConvertTo-Json -Depth 8

  $decline = $snapshot.elements | Where-Object { $_.text -eq "Do not consent" } | Select-Object -First 1
  if (-not $decline) {
    throw "Could not find Do not consent button"
  }

  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "click"
    ref_id = $decline.ref_id
  } | ConvertTo-Json) | Out-Null

  Start-Sleep -Seconds 10

  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "wait_network_idle"
  } | ConvertTo-Json) | Out-Null

  $postClickSnapshot = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/snapshot" -Method Get
  "=== Post Click Snapshot ==="
  $postClickSnapshot | ConvertTo-Json -Depth 8
}
finally {
  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId" -Method Delete | Out-Null
}
