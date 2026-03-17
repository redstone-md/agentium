$ErrorActionPreference = "Stop"

param(
  [string]$BaseUrl = "http://127.0.0.1:8080"
)

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

  $lines = New-Object System.Collections.Generic.List[string]
  $deadline = [DateTime]::UtcNow.AddSeconds(10)
  while ([DateTime]::UtcNow -lt $deadline) {
    $line = $reader.ReadLine()
    if ($null -eq $line) {
      Start-Sleep -Milliseconds 100
      continue
    }

    if ($line -ne "") {
      $lines.Add($line)
      Write-Host $line
      continue
    }

    if ($lines.Count -gt 0) {
      break
    }
  }

  if ($lines.Count -eq 0) {
    throw "No SSE event received from $BaseUrl/mcp"
  }

  $eventLine = $lines | Where-Object { $_ -like "event:*" } | Select-Object -First 1
  $dataLine = $lines | Where-Object { $_ -like "data:*" } | Select-Object -First 1

  if ($eventLine -ne "event: endpoint") {
    throw "Expected first SSE event to be 'endpoint', got '$eventLine'"
  }

  if (-not $dataLine -or $dataLine -notmatch "sessionid=") {
    throw "Expected endpoint event data to contain sessionid, got '$dataLine'"
  }
}
finally {
  if ($reader) { $reader.Dispose() }
  if ($stream) { $stream.Dispose() }
  if ($response) { $response.Dispose() }
  $client.Dispose()
}
