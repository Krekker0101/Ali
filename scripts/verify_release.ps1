<#
.SYNOPSIS
    Verify an installed Ali release on Windows.

.DESCRIPTION
    Performs a post-install smoke test against the local executable and HTTP
    APIs. It can start the server if it is not already running and writes a JSON
    report to %TEMP%\ali-release-verify.json by default.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\verify_release.ps1 -StartServer -Model qwen2.5-coder:1.5b -RequireModel
#>

[CmdletBinding()]
param(
    [string]$InstallDir = "",
    [string]$BaseUrl = "http://127.0.0.1:11434",
    [string]$Model = "",
    [string]$ReportPath = "",
    [int]$TimeoutSec = 90,
    [switch]$StartServer,
    [switch]$RequireModel
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

if (-not $ReportPath) {
    $ReportPath = Join-Path $env:TEMP "ali-release-verify.json"
}

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Find-AliInstallDir {
    if ($InstallDir) {
        return [System.IO.Path]::GetFullPath($InstallDir)
    }

    $candidates = @(
        (Join-Path $env:LOCALAPPDATA "Programs\Ali"),
        (Join-Path $env:LOCALAPPDATA "Programs\Ollama"),
        (Join-Path $env:ProgramFiles "Ali"),
        (Join-Path $env:ProgramFiles "Ollama")
    )

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath (Join-Path $candidate "ollama.exe")) {
            return $candidate
        }
    }
    return Join-Path $env:LOCALAPPDATA "Programs\Ali"
}

function Invoke-JsonEndpoint {
    param([string]$Path)

    $uri = $BaseUrl.TrimEnd("/") + $Path
    $response = Invoke-WebRequest -UseBasicParsing -Uri $uri -TimeoutSec 10
    if ($response.StatusCode -lt 200 -or $response.StatusCode -ge 300) {
        throw "$uri returned HTTP $($response.StatusCode)"
    }
    if ($response.Content) {
        return $response.Content | ConvertFrom-Json
    }
    return $null
}

function Test-EndpointReady {
    try {
        Invoke-JsonEndpoint -Path "/api/version" | Out-Null
        return $true
    } catch {
        return $false
    }
}

function Wait-ServerReady {
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    do {
        if (Test-EndpointReady) {
            return
        }
        Start-Sleep -Milliseconds 750
    } while ((Get-Date) -lt $deadline)

    throw "Server did not become ready at $BaseUrl within $TimeoutSec seconds"
}

function Test-ModelInstalled {
    param([string]$Name)

    if (-not $Name) {
        return $true
    }

    $tags = Invoke-JsonEndpoint -Path "/api/tags"
    foreach ($modelInfo in @($tags.models)) {
        if ($modelInfo.name -eq $Name -or $modelInfo.model -eq $Name) {
            return $true
        }
    }
    return $false
}

$checks = [System.Collections.Generic.List[object]]::new()

function Add-Check {
    param(
        [string]$Name,
        [bool]$Passed,
        [string]$Details = ""
    )

    $checks.Add([ordered]@{
        name = $Name
        passed = $Passed
        details = $Details
    })
}

$installRoot = Find-AliInstallDir
$ollamaExe = Join-Path $installRoot "ollama.exe"

try {
    Write-Step "Checking installed files"
    if (-not (Test-Path -LiteralPath $ollamaExe)) {
        throw "ollama.exe not found: $ollamaExe"
    }
    Add-Check -Name "ollama_exe" -Passed $true -Details $ollamaExe

    if ($StartServer -and -not (Test-EndpointReady)) {
        Write-Step "Starting local server"
        Start-Process -FilePath $ollamaExe -ArgumentList "serve" -WorkingDirectory $installRoot -WindowStyle Hidden | Out-Null
    }

    Write-Step "Waiting for server"
    Wait-ServerReady
    Add-Check -Name "server_ready" -Passed $true -Details $BaseUrl

    Write-Step "Checking core APIs"
    $version = Invoke-JsonEndpoint -Path "/api/version"
    Add-Check -Name "api_version" -Passed $true -Details ($version | ConvertTo-Json -Compress)

    $ideHtml = Invoke-WebRequest -UseBasicParsing -Uri ($BaseUrl.TrimEnd("/") + "/ide") -TimeoutSec 10
    Add-Check -Name "ide_page" -Passed ($ideHtml.StatusCode -eq 200) -Details "HTTP $($ideHtml.StatusCode)"

    $health = Invoke-JsonEndpoint -Path "/api/v1/ide/health"
    Add-Check -Name "ide_health" -Passed ($health.status -eq "ok") -Details ($health | ConvertTo-Json -Compress -Depth 8)

    $settings = Invoke-JsonEndpoint -Path "/api/v1/ide/settings"
    Add-Check -Name "ide_settings" -Passed ($null -ne $settings.ai -and $null -ne $settings.theme) -Details ($settings | ConvertTo-Json -Compress -Depth 6)

    if ($Model) {
        Write-Step "Checking local model"
        $modelReady = Test-ModelInstalled -Name $Model
        Add-Check -Name "model_installed" -Passed $modelReady -Details $Model
        if ($RequireModel -and -not $modelReady) {
            throw "Model is not installed: $Model"
        }
    }

    $report = [ordered]@{
        status = "passed"
        created_at = (Get-Date).ToString("o")
        install_dir = $installRoot
        base_url = $BaseUrl
        checks = $checks
    }
    $report | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $ReportPath -Encoding UTF8
    Write-Host "Release verification passed. Report: $ReportPath" -ForegroundColor Green
} catch {
    Add-Check -Name "fatal" -Passed $false -Details $_.Exception.Message
    $report = [ordered]@{
        status = "failed"
        created_at = (Get-Date).ToString("o")
        install_dir = $installRoot
        base_url = $BaseUrl
        checks = $checks
        error = $_.Exception.Message
    }
    $report | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $ReportPath -Encoding UTF8
    Write-Host "Release verification failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Report: $ReportPath" -ForegroundColor Yellow
    exit 1
}
