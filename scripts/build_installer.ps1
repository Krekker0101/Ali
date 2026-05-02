<#
.SYNOPSIS
    Build a complete Windows installer for Ali.

.DESCRIPTION
    This wrapper performs a strict preflight, optionally installs build
    dependencies with winget, calls the existing Windows build pipeline, and
    validates the produced installer.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\build_installer.ps1

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\build_installer.ps1 -CpuOnly

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\build_installer.ps1 -Bootstrap -CpuOnly
#>

[CmdletBinding()]
param(
    [switch]$Bootstrap,
    [switch]$CpuOnly,
    [switch]$PreflightOnly,
    [string]$Version = "",
    [string]$Arch = ""
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$BuildScript = Join-Path $RepoRoot "scripts\build_windows.ps1"
$InstallerPath = Join-Path $RepoRoot "dist\AliSetup.exe"
$BuildEnvScript = Join-Path $PSScriptRoot "windows_build_env.ps1"
if (Test-Path -LiteralPath $BuildEnvScript) {
    . $BuildEnvScript
}

function Write-Section {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Test-CommandAvailable {
    param([string]$Name)
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Update-CurrentPath {
    $machine = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $user = [Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = "$machine;$user;$env:Path"
}

function Find-InnoSetupCompiler {
    if ($env:INNO_SETUP_DIR) {
        $compiler = Join-Path $env:INNO_SETUP_DIR "ISCC.exe"
        if (Test-Path -LiteralPath $compiler) {
            return $compiler
        }
    }

    $candidatePaths = @(
        (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6"),
        (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup"),
        "C:\Program Files\Inno Setup 6",
        "C:\Program Files (x86)\Inno Setup 6",
        "C:\Program Files\Inno Setup",
        "C:\Program Files (x86)\Inno Setup"
    )
    foreach ($path in $candidatePaths) {
        $compiler = Join-Path $path "ISCC.exe"
        if (Test-Path -LiteralPath $compiler) {
            return $compiler
        }
    }

    try {
        $regRoots = @(
            "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
            "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall",
            "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall"
        )
        foreach ($root in $regRoots) {
            $installs = Get-ItemProperty "$root\*" -ErrorAction SilentlyContinue |
                Where-Object { $_.DisplayName -like "Inno Setup*" -or $_.PSChildName -like "Inno Setup*" }
            foreach ($install in $installs) {
                if ($install.InstallLocation) {
                    $compiler = Join-Path $install.InstallLocation "ISCC.exe"
                    if (Test-Path -LiteralPath $compiler) {
                        return $compiler
                    }
                }
            }
        }
    } catch {
    }

    if (Test-CommandAvailable iscc) {
        return (Get-Command iscc).Source
    }
    return ""
}

function Find-VSBuildTools {
    $vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
    if (Test-Path -LiteralPath $vswhere) {
        $install = & $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
        if ($LASTEXITCODE -eq 0 -and $install) {
            return $install
        }

        $install = & $vswhere -latest -products * -property installationPath
        if ($LASTEXITCODE -eq 0 -and $install) {
            $cl = Get-ChildItem -LiteralPath $install -Recurse -Filter cl.exe -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($cl) {
                return $install
            }
        }
    }

    $fallbacks = @(
        "${env:ProgramFiles(x86)}\Microsoft Visual Studio\2022\BuildTools",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\BuildTools",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Community",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Professional",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Enterprise"
    )
    foreach ($path in $fallbacks) {
        if (Test-Path -LiteralPath $path) {
            $cl = Get-ChildItem -LiteralPath $path -Recurse -Filter cl.exe -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($cl) {
                return $path
            }
        }
    }
    return ""
}

function Install-WithWinget {
    param(
        [string]$Id,
        [string]$Name,
        [string]$Override = ""
    )

    if (-not (Test-CommandAvailable winget)) {
        throw "winget is required for -Bootstrap. Install App Installer from Microsoft Store first."
    }

    if (-not $Override) {
        $installed = & winget list --id $Id --exact --source winget --accept-source-agreements 2>&1 | Out-String
        if ($LASTEXITCODE -eq 0 -and $installed -match [regex]::Escape($Id)) {
            Write-Host "$Name is already installed."
            return
        }
    }

    Write-Host "Installing $Name ($Id)..."
    $wingetArgs = @(
        "install",
        "--id", $Id,
        "--source", "winget",
        "--exact",
        "--silent",
        "--disable-interactivity",
        "--accept-package-agreements",
        "--accept-source-agreements"
    )
    if ($Override) {
        $wingetArgs += @("--override", $Override)
    }
    $output = & winget @wingetArgs 2>&1 | Out-String
    if ($output) {
        Write-Host ($output.Trim())
    }
    if ($LASTEXITCODE -eq 0) {
        return
    }
    if ($output -match "Found an existing package|already installed|No available upgrade|No newer package") {
        Write-Host "$Name is already installed."
    } else {
        throw "winget install failed for $Name"
    }
}

function Invoke-Bootstrap {
    Write-Section "Installing build dependencies"
    Install-WithWinget -Id "GoLang.Go" -Name "Go"
    Install-WithWinget -Id "OpenJS.NodeJS.LTS" -Name "Node.js LTS"
    Install-WithWinget -Id "Kitware.CMake" -Name "CMake"
    Install-WithWinget -Id "Git.Git" -Name "Git"
    Install-WithWinget -Id "JRSoftware.InnoSetup" -Name "Inno Setup"

    $vs = Find-VSBuildTools
    if (-not $vs) {
        Install-WithWinget `
            -Id "Microsoft.VisualStudio.2022.BuildTools" `
            -Name "Visual Studio 2022 Build Tools" `
            -Override "--wait --quiet --norestart --installWhileDownloading --add Microsoft.VisualStudio.Component.VC.Tools.x86.x64"
        $vs = Find-VSBuildTools
        if (-not $vs) {
            throw "Visual Studio C++ Build Tools were not found after install attempt"
        }
    }

    if (-not (Import-AliVSDeveloperEnvironment -InstallRoot $vs)) {
        Write-Host "Visual Studio C++ environment is incomplete. Installing required components..."
        if (-not (Ensure-AliVSBuildEnvironment -InstallRoot $vs -InstallWindowsSdk)) {
            throw "Visual Studio C++ developer environment is not ready after repair"
        }
    }

    Write-Host ""
    Write-Host "Close and reopen PowerShell if newly installed tools are not visible in PATH." -ForegroundColor Yellow
}

function Invoke-Preflight {
    Write-Section "Preflight"
    $missing = New-Object System.Collections.Generic.List[string]

    foreach ($cmd in @("git", "go", "gofmt", "cmake", "node", "npm")) {
        if (Test-CommandAvailable $cmd) {
            $source = (Get-Command $cmd).Source
            Write-Host "OK   $cmd -> $source"
        } else {
            Write-Host "MISS $cmd" -ForegroundColor Red
            $missing.Add($cmd)
        }
    }

    $inno = Find-InnoSetupCompiler
    if ($inno) {
        Write-Host "OK   Inno Setup -> $inno"
    } else {
        Write-Host "MISS Inno Setup (ISCC.exe)" -ForegroundColor Red
        $missing.Add("Inno Setup")
    }

    $vs = Find-VSBuildTools
    if ($vs) {
        Write-Host "OK   Visual Studio C++ Build Tools -> $vs"
        if (Import-AliVSDeveloperEnvironment -InstallRoot $vs) {
            Write-Host "OK   Visual Studio C++ environment -> cl/nmake/Windows SDK"
        } else {
            Write-Host "MISS Visual Studio C++ environment (cl.exe, nmake.exe, or Windows SDK)" -ForegroundColor Red
            $missing.Add("Visual Studio C++ environment")
        }
    } else {
        Write-Host "MISS Visual Studio C++ Build Tools" -ForegroundColor Red
        $missing.Add("Visual Studio C++ Build Tools")
    }

    if ($missing.Count -gt 0) {
        $list = $missing -join ", "
        throw "Missing build dependencies: $list. Re-run with -Bootstrap or install them manually."
    }
}

function Invoke-InstallerBuild {
    Write-Section "Building installer"
    $vs = Find-VSBuildTools
    if ($vs -and -not (Import-AliVSDeveloperEnvironment -InstallRoot $vs)) {
        throw "Visual Studio C++ developer environment is not ready. Re-run with -Bootstrap."
    }

    $inno = Find-InnoSetupCompiler
    if ($inno) {
        $env:INNO_SETUP_DIR = Split-Path -Parent $inno
    }

    Push-Location $RepoRoot
    try {
        if ($Version) {
            $env:VERSION = $Version
        }
        if ($Arch) {
            $env:ARCH = $Arch
        }

        $steps = if ($CpuOnly) {
            @("clean", "cpu", "ollama", "app", "deps", "sign", "installer", "zip")
        } else {
            @("clean", "cpu", "cuda12", "cuda13", "rocm6", "vulkan", "mlxCuda13", "ollama", "app", "deps", "sign", "installer", "zip")
        }

        & powershell -ExecutionPolicy Bypass -File $BuildScript @steps
        if ($LASTEXITCODE -ne 0) {
            throw "build_windows.ps1 failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
}

function Test-InstallerArtifact {
    Write-Section "Validating installer artifact"
    if (-not (Test-Path -LiteralPath $InstallerPath)) {
        throw "Installer was not produced: $InstallerPath"
    }

    $item = Get-Item -LiteralPath $InstallerPath
    if ($item.Length -le 0) {
        throw "Installer is empty: $InstallerPath"
    }

    $hash = Get-FileHash -LiteralPath $InstallerPath -Algorithm SHA256
    Write-Host "Installer: $($item.FullName)"
    Write-Host "Size:      $([Math]::Round($item.Length / 1MB, 2)) MB"
    Write-Host "SHA256:    $($hash.Hash)"
}

if (-not (Test-Path -LiteralPath $BuildScript)) {
    throw "Windows build script not found: $BuildScript"
}

if ($Bootstrap) {
    Invoke-Bootstrap
}

Update-CurrentPath
Invoke-Preflight

if ($PreflightOnly) {
    Write-Host ""
    Write-Host "Preflight passed." -ForegroundColor Green
    exit 0
}

Invoke-InstallerBuild
Test-InstallerArtifact

Write-Host ""
Write-Host "Done. Run dist\AliSetup.exe to install Ali and open 'Ali IDE' from the Start Menu." -ForegroundColor Green
