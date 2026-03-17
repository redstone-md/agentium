param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$TargetUrl = "https://pixelscan.net/fingerprint-check",
  [string]$Locale = "",
  [string]$ProfileId = "",
  [int]$DelayAfterNavigateSeconds = 15
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

$sessionOptions = @{}
if ($Locale) {
  $sessionOptions.locale = $Locale
}
if ($ProfileId) {
  $sessionOptions.profile_id = $ProfileId
}

$session = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions" -Method Post -ContentType "application/json" -Body ($sessionOptions | ConvertTo-Json)

$sessionId = $session.session_id
if (-not $sessionId) {
  throw "session_id was empty"
}

try {
  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "navigate"
    value  = $TargetUrl
  } | ConvertTo-Json) | Out-Null

  Start-Sleep -Seconds $DelayAfterNavigateSeconds

  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "wait_network_idle"
  } | ConvertTo-Json) | Out-Null

  $snapshot = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/snapshot" -Method Get
  "=== Snapshot ==="
  $snapshot | ConvertTo-Json -Depth 8

  $pageText = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/text" -Method Get
  "=== Page Text ==="
  $pageText | ConvertTo-Json -Depth 8
}
finally {
  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId" -Method Delete | Out-Null
}
