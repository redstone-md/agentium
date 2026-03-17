$ErrorActionPreference = "Stop"

param(
  [string]$BinaryPath = ".\agentium.exe"
)

$request = @(
  @{ jsonrpc = "2.0"; id = 1; method = "initialize"; params = @{ protocolVersion = "2025-03-26"; capabilities = @{}; clientInfo = @{ name = "agentium-smoke"; version = "1.0.0" } } },
  @{ jsonrpc = "2.0"; method = "notifications/initialized"; params = @{} },
  @{ jsonrpc = "2.0"; id = 2; method = "tools/list"; params = @{} }
) | ForEach-Object { ($_ | ConvertTo-Json -Compress -Depth 10) }

$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = $BinaryPath
$psi.Arguments = "-mode mcp-stdio"
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.RedirectStandardError = $true
$psi.UseShellExecute = $false
$psi.CreateNoWindow = $true

$process = New-Object System.Diagnostics.Process
$process.StartInfo = $psi
$null = $process.Start()

try {
  foreach ($line in $request) {
    $process.StandardInput.WriteLine($line)
  }
  $process.StandardInput.Flush()
  Start-Sleep -Milliseconds 500

  $fullOutput = New-Object System.Collections.Generic.List[string]

  while (-not $process.StandardOutput.EndOfStream) {
    $output = $process.StandardOutput.ReadLine()
    if ($output) {
      $fullOutput.Add($output)
      Write-Host $output
    }
    if ($output -match '"id":2') {
      break
    }
  }

  $joined = ($fullOutput -join "`n")
  foreach ($toolName in @(
    "agentium_create_session",
    "agentium_get_snapshot",
    "agentium_perform_action",
    "agentium_close_session"
  )) {
    if ($joined -notmatch $toolName) {
      throw "Expected MCP tool not found in tools/list response: $toolName"
    }
  }
}
finally {
  if (-not $process.HasExited) {
    $process.Kill()
  }
}
