param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$TargetUrl = "https://bot.sannysoft.com/",
  [int]$DelayAfterNavigateMs = 8000
)

$ErrorActionPreference = "Stop"

pwsh -NoProfile -File .\scripts\smoke-http.ps1 `
  -BaseUrl $BaseUrl `
  -TargetUrl $TargetUrl `
  -DelayAfterNavigateMs $DelayAfterNavigateMs `
  -RequiredText @(
    "WebDriver (New) missing (passed)",
    "WebDriver Advanced passed"
  ) `
  -ForbiddenText @(
    "WebDriver (New) present (failed)"
  )
