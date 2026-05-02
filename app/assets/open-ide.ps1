param(
    [string]$HostUrl = "http://127.0.0.1:11434",
    [int]$TimeoutSeconds = 25
)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms

function Test-AliServer {
    param([string]$BaseUrl)

    try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/api/version" -TimeoutSec 2
        return $response.StatusCode -ge 200 -and $response.StatusCode -lt 300
    } catch {
        return $false
    }
}

$appDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$serverExe = Join-Path $appDir "ollama.exe"
$ideUrl = "$HostUrl/ide"

if (-not (Test-Path -LiteralPath $serverExe)) {
    [System.Windows.Forms.MessageBox]::Show(
        "Ali server executable was not found:`n$serverExe",
        "Ali IDE",
        [System.Windows.Forms.MessageBoxButtons]::OK,
        [System.Windows.Forms.MessageBoxIcon]::Error
    ) | Out-Null
    exit 1
}

if (-not (Test-AliServer -BaseUrl $HostUrl)) {
    Start-Process -FilePath $serverExe -ArgumentList "serve" -WorkingDirectory $appDir -WindowStyle Hidden | Out-Null

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    do {
        Start-Sleep -Milliseconds 500
        if (Test-AliServer -BaseUrl $HostUrl) {
            break
        }
    } while ((Get-Date) -lt $deadline)
}

if (Test-AliServer -BaseUrl $HostUrl) {
    Start-Process $ideUrl
    exit 0
}

[System.Windows.Forms.MessageBox]::Show(
    "Ali server did not become ready at $HostUrl within $TimeoutSeconds seconds.",
    "Ali IDE",
    [System.Windows.Forms.MessageBoxButtons]::OK,
    [System.Windows.Forms.MessageBoxIcon]::Warning
) | Out-Null
exit 1
