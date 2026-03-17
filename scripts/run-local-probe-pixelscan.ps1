param(
  [string]$BinaryPath = ".\agentium.exe",
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$TargetUrl = "https://pixelscan.net/fingerprint-check",
  [string]$ProfileId = "",
  [switch]$DisableLeakless
)

$ErrorActionPreference = "Stop"

$previousLeakless = $env:AGENTIUM_LEAKLESS

if ($DisableLeakless) {
  $env:AGENTIUM_LEAKLESS = "false"
}

$process = Start-Process -FilePath $BinaryPath -ArgumentList "-mode", "http" -PassThru -WindowStyle Hidden

try {
  $args = @(
    "-NoProfile",
    "-File", ".\scripts\probe-pixelscan.ps1",
    "-BaseUrl", $BaseUrl,
    "-TargetUrl", $TargetUrl
  )
  if ($ProfileId) {
    $args += @("-ProfileId", $ProfileId)
  }

  pwsh @args
}
finally {
  if ($process -and -not $process.HasExited) {
    Stop-Process -Id $process.Id -Force
  }

  if ($DisableLeakless) {
    if ($null -eq $previousLeakless) {
      Remove-Item Env:AGENTIUM_LEAKLESS -ErrorAction SilentlyContinue
    }
    else {
      $env:AGENTIUM_LEAKLESS = $previousLeakless
    }
  }
}
