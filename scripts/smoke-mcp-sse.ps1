param(
  [string]$BaseUrl = "http://127.0.0.1:8080"
)

$ErrorActionPreference = "Stop"

function Read-SseEvent {
  param(
    [System.IO.StreamReader]$Reader,
    [int]$TimeoutSeconds = 10
  )

  $lines = New-Object System.Collections.Generic.List[string]
  $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)

  while ([DateTime]::UtcNow -lt $deadline) {
    $readTask = $Reader.ReadLineAsync()
    $null = $readTask.Wait(250)
    if (-not $readTask.IsCompleted) {
      continue
    }

    $line = $readTask.Result
    if ($null -eq $line) {
      Start-Sleep -Milliseconds 100
      continue
    }

    if ($line -eq "") {
      if ($lines.Count -gt 0) {
        return $lines
      }
      continue
    }

    $lines.Add($line)
    Write-Host $line
  }

  throw "Timed out waiting for SSE event"
}

function Wait-ForMatchingMessage {
  param(
    [System.IO.StreamReader]$Reader,
    [string]$Pattern,
    [int]$TimeoutSeconds = 10
  )

  $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
  while ([DateTime]::UtcNow -lt $deadline) {
    $eventLines = Read-SseEvent -Reader $Reader -TimeoutSeconds $TimeoutSeconds
    $eventName = $eventLines | Where-Object { $_ -like "event:*" } | Select-Object -First 1
    $dataLine = $eventLines | Where-Object { $_ -like "data:*" } | Select-Object -First 1
    if ($eventName -ne "event: message" -or -not $dataLine) {
      continue
    }

    if ($dataLine -match $Pattern) {
      return $dataLine
    }
  }

  throw "Timed out waiting for SSE message matching pattern: $Pattern"
}

function Post-JsonRpcMessage {
  param(
    [System.Net.Http.HttpClient]$Client,
    [string]$Url,
    [hashtable]$Payload
  )

  $json = $Payload | ConvertTo-Json -Compress -Depth 20
  $content = New-Object System.Net.Http.StringContent($json, [Text.Encoding]::UTF8, "application/json")
  try {
    $response = $Client.PostAsync($Url, $content).GetAwaiter().GetResult()
    if (-not $response.IsSuccessStatusCode) {
      throw "POST $Url failed with status $($response.StatusCode)"
    }
  }
  finally {
    if ($response) { $response.Dispose() }
    $content.Dispose()
  }
}

$handler = New-Object System.Net.Http.HttpClientHandler
$client = New-Object System.Net.Http.HttpClient($handler)
$client.Timeout = [TimeSpan]::FromSeconds(15)
$client.DefaultRequestHeaders.Accept.ParseAdd("text/event-stream")

try {
  $response = $client.GetAsync("$BaseUrl/mcp", [System.Net.Http.HttpCompletionOption]::ResponseHeadersRead).GetAwaiter().GetResult()
  if (-not $response.IsSuccessStatusCode) {
    throw "Unexpected status code: $($response.StatusCode)"
  }

  $contentType = $response.Content.Headers.ContentType.MediaType
  if ($contentType -ne "text/event-stream") {
    throw "Unexpected content type: $contentType"
  }

  $stream = $response.Content.ReadAsStreamAsync().GetAwaiter().GetResult()
  $reader = New-Object System.IO.StreamReader($stream)

  $endpointEvent = Read-SseEvent -Reader $reader
  $eventLine = $endpointEvent | Where-Object { $_ -like "event:*" } | Select-Object -First 1
  $dataLine = $endpointEvent | Where-Object { $_ -like "data:*" } | Select-Object -First 1

  if ($eventLine -ne "event: endpoint") {
    throw "Expected first SSE event to be 'endpoint', got '$eventLine'"
  }

  if (-not $dataLine -or $dataLine -notmatch "sessionid=") {
    throw "Expected endpoint event data to contain sessionid, got '$dataLine'"
  }

  $messagesUrl = "$BaseUrl/mcp" + ($dataLine -replace "^data:\s*", "")

  Post-JsonRpcMessage -Client $client -Url $messagesUrl -Payload @{
    jsonrpc = "2.0"
    id = 1
    method = "initialize"
    params = @{
      protocolVersion = "2025-03-26"
      capabilities = @{}
      clientInfo = @{
        name = "agentium-smoke-sse"
        version = "1.0.0"
      }
    }
  }

  $initData = Wait-ForMatchingMessage -Reader $reader -Pattern '"id":1'

  Post-JsonRpcMessage -Client $client -Url $messagesUrl -Payload @{
    jsonrpc = "2.0"
    method = "notifications/initialized"
    params = @{}
  }

  Post-JsonRpcMessage -Client $client -Url $messagesUrl -Payload @{
    jsonrpc = "2.0"
    id = 2
    method = "tools/list"
    params = @{}
  }

  $toolsData = Wait-ForMatchingMessage -Reader $reader -Pattern '"id":2'

  $jsonPayload = ($toolsData -replace "^data:\s*", "") | ConvertFrom-Json -Depth 20
  $toolNames = @($jsonPayload.result.tools | ForEach-Object { $_.name })
  foreach ($toolName in @(
    "agentium_create_session",
    "agentium_get_snapshot",
    "agentium_perform_action",
    "agentium_close_session"
  )) {
    if ($toolNames -notcontains $toolName) {
      throw "Expected MCP tool not found in SSE tools/list response: $toolName"
    }
  }
}
finally {
  if ($reader) { $reader.Dispose() }
  if ($stream) { $stream.Dispose() }
  if ($response) { $response.Dispose() }
  $client.Dispose()
}
