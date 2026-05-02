<#
.SYNOPSIS
    Verify Windows build environment is properly configured.

.DESCRIPTION
    Checks all required build tools and provides detailed diagnostic information
    if any tools are missing or misconfigured. Safe to run multiple times.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\verify_windows_setup.ps1
#>

[CmdletBinding()]
param()

$ErrorActionPreference = "Continue"
$ProgressPreference = "SilentlyContinue"

$script:allValid = $true
$script:toolStatus = @()

function Test-CommandAvailable {
    param([string]$Name)
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Check-Tool {
    param(
        [string]$Name,
        [string]$Command,
        [string]$Description = ""
    )
    
    $installed = Test-CommandAvailable $Command
    $status = if ($installed) { "[OK]" } else { "[X]" }
    
    $entry = @{
        Name = $Name
        Status = $status
        Command = $Command
        Description = $Description
        Installed = $installed
    }
    
    $script:toolStatus += $entry
    
    if (-not $installed) {
        $script:allValid = $false
    }
    
    Write-Host "$status  $Name" -ForegroundColor $(if ($installed) { 'Green' } else { 'Red' })
}

function Check-File {
    param(
        [string]$Name,
        [string]$Path,
        [string]$Description = ""
    )
    
    $exists = Test-Path -LiteralPath $Path
    $status = if ($exists) { "[OK]" } else { "[X]" }
    
    if (-not $exists) {
        $script:allValid = $false
    }
    
    Write-Host "$status  $Name at $Path" -ForegroundColor $(if ($exists) { 'Green' } else { 'Red' })
}

function Find-InnoSetupCompiler {
    $candidates = @(
        (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6"),
        (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup"),
        "C:\Program Files\Inno Setup 6",
        "C:\Program Files (x86)\Inno Setup 6",
        "C:\Program Files\Inno Setup",
        "C:\Program Files (x86)\Inno Setup"
    )
    
    foreach ($path in $candidates) {
        $compiler = Join-Path $path "ISCC.exe"
        if (Test-Path -LiteralPath $compiler) {
            return $compiler
        }
    }
    
    return $null
}

function Get-ToolVersion {
    param([string]$Command)
    try {
        $version = & $Command --version 2>$null
        return $version
    } catch {
        return "Unknown"
    }
}

Write-Host ""
Write-Host "=== Windows Build Environment Verification ===" -ForegroundColor Cyan
Write-Host ""

Write-Host "Required Build Tools:" -ForegroundColor Yellow
Check-Tool -Name "Git" -Command "git"
Check-Tool -Name "Go" -Command "go"
Check-Tool -Name "Node.js" -Command "node"
Check-Tool -Name "npm" -Command "npm"
Check-Tool -Name "CMake" -Command "cmake"
Check-Tool -Name "C# Compiler (csc)" -Command "csc" -Description ".NET Framework"

Write-Host ""
Write-Host "Installer Tools:" -ForegroundColor Yellow
$innoSetup = Find-InnoSetupCompiler
if ($innoSetup) {
    Write-Host "[OK]  Inno Setup at $innoSetup" -ForegroundColor Green
} else {
    Write-Host "[X]  Inno Setup" -ForegroundColor Red
    $script:allValid = $false
}

Write-Host ""
Write-Host "Visual Studio Build Tools:" -ForegroundColor Yellow
$vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path -LiteralPath $vswhere) {
    try {
        $vsInstall = & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath 2>$null
        if ($vsInstall) {
            Write-Host "[OK]  Visual Studio Build Tools at $vsInstall" -ForegroundColor Green
        } else {
            Write-Host "[!]  Visual Studio found but VC++ tools not detected" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "[OK]  vswhere.exe (VS may be installed)" -ForegroundColor Green
    }
} else {
    Write-Host "[X]  Visual Studio Build Tools" -ForegroundColor Red
    $script:allValid = $false
}

Write-Host ""
Write-Host "Source Repository:" -ForegroundColor Yellow
Check-File -Name "app/ollama.iss" -Path "app\ollama.iss"
Check-File -Name "go.mod" -Path "go.mod"

Write-Host ""
Write-Host "Summary:" -ForegroundColor Cyan
if ($script:allValid) {
    Write-Host "[OK]  All required tools are installed and available." -ForegroundColor Green
    exit 0
} else {
    Write-Host "[X]  Some required tools are missing. Please install them using:" -ForegroundColor Red
    Write-Host "     powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1"
    exit 1
}
