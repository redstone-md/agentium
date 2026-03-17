param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$TargetUrl = "https://www.browserscan.net/bot-detection",
  [string]$SessionMode = "persistent",
  [int]$DelayAfterNavigateSeconds = 12,
  [int]$DelayAfterConsentSeconds = 10
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

$session = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions" -Method Post -ContentType "application/json" -Body (@{
  locale = "en-US"
  session_mode = $SessionMode
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

  Start-Sleep -Seconds $DelayAfterNavigateSeconds

  $snapshot = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/snapshot" -Method Get
  $decline = $snapshot.elements | Where-Object { $_.text -eq "Do not consent" } | Select-Object -First 1
  if (-not $decline) {
    throw "could not find Do not consent button"
  }

  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "click"
    ref_id = $decline.ref_id
  } | ConvertTo-Json) | Out-Null

  Start-Sleep -Seconds $DelayAfterConsentSeconds

  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/action" -Method Post -ContentType "application/json" -Body (@{
    action = "wait_network_idle"
  } | ConvertTo-Json) | Out-Null

  $pageText = Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId/text" -Method Get
  $pageText | ConvertTo-Json -Depth 8

  Assert-Contains -Text $pageText.text -Required @(
    "WebDriver Normal",
    "WebDriver Advance Normal",
    "Selenium Normal",
    "CDP Normal",
    "Dev Tool Normal"
  )
}
finally {
  Invoke-RestMethod -Uri "$BaseUrl/v1/sessions/$sessionId" -Method Delete | Out-Null
}
