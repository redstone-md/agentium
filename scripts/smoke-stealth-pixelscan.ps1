param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$TargetUrl = "https://pixelscan.net/fingerprint-check",
  [string]$Locale = "",
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

function Assert-Contains {
  param(
    [string]$Text,
    [string[]]$Required
  )

  foreach ($needle in $Required) {
    if (-not $Text.Contains($needle, [System.StringComparison]::OrdinalIgnoreCase)) {
      throw "page text did not contain required marker: $needle"
    }
  }
}

Wait-ForHealth -Url $BaseUrl | Out-Null

$sessionOptions = @{}
if ($Locale) {
  $sessionOptions.locale = $Locale
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

  $pageText = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/text" -Method Get
  $pageText | ConvertTo-Json -Depth 8

  Assert-Contains -Text $pageText.text -Required @(
    "No automated behavior detected"
  )
}
finally {
  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId" -Method Delete | Out-Null
}
