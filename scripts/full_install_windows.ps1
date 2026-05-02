<#
.SYNOPSIS
    Full online installer for Ali on Windows.

.DESCRIPTION
    Installs required build tools, builds AliSetup.exe from this source tree,
    runs the installer, starts the local server, and opens the built-in IDE.
    Local models are not downloaded during installation unless -PreloadModel is
    explicitly provided.

    If ALI_INSTALLER_URL or -InstallerUrl is provided, the script downloads and
    runs that installer instead of building from source.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1 -CpuOnly

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1 -CpuOnly -PreloadModel -Model qwen2.5-coder:1.5b

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1 -InstallerUrl https://example.com/AliSetup.exe
#>

[CmdletBinding()]
param(
    [string]$SourceDir = "",
    [string]$InstallerUrl = "",
    [string]$InstallDir = "",
    [string]$Model = "",
    [int]$RetryCount = 3,
    [switch]$PreloadModel,
    [switch]$SkipModel,
    [switch]$SkipVerify,
    [switch]$CpuOnly,
    [switch]$NoOpenIDE,
    [switch]$NoElevate
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$LogFile = Join-Path $env:TEMP "ali-full-install.log"
$BuildEnvScript = Join-Path $PSScriptRoot "windows_build_env.ps1"
if (Test-Path -LiteralPath $BuildEnvScript) {
    . $BuildEnvScript
}

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
    SafeAddContent -Message "==> $Message"
}

function Write-Info {
    param([string]$Message)
    Write-Host $Message
    SafeAddContent -Message $Message
}

function SafeAddContent {
    param([string]$Message)
    try {
        Add-Content -LiteralPath $LogFile -Value $Message -ErrorAction SilentlyContinue
    } catch {
        # If file is locked or other error occurs, try alternate approach
        try {
            [System.IO.File]::AppendAllText($LogFile, "$Message`n", [System.Text.Encoding]::UTF8)
        } catch {
            # Last resort - write to a backup log
            $backupLog = "$LogFile.backup"
            try {
                Add-Content -LiteralPath $backupLog -Value $Message -ErrorAction SilentlyContinue
            } catch {
                # Silently fail - we don't want logging errors to stop the installation
            }
        }
    }
}

function Invoke-WithRetry {
    param(
        [scriptblock]$Action,
        [string]$Name,
        [int]$Attempts = $RetryCount
    )

    if ($Attempts -lt 1) {
        $Attempts = 1
    }

    for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
        try {
            & $Action
            return
        } catch {
            if ($attempt -ge $Attempts) {
                throw
            }
            $delay = [Math]::Min(30, 3 * $attempt)
            Write-Info "$Name failed on attempt $attempt/${Attempts}: $($_.Exception.Message)"
            Write-Info "Retrying in $delay seconds..."
            Start-Sleep -Seconds $delay
        }
    }
}

function Test-IsAdmin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Get-RelaunchArguments {
    $relaunchArgs = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "`"$PSCommandPath`"")
    foreach ($entry in $PSBoundParameters.GetEnumerator()) {
        if ($entry.Key -eq "NoElevate") {
            continue
        }
        if ($entry.Value -is [System.Management.Automation.SwitchParameter]) {
            if ($entry.Value.IsPresent) {
                $relaunchArgs += "-$($entry.Key)"
            }
        } elseif ($null -ne $entry.Value -and "$($entry.Value)" -ne "") {
            $relaunchArgs += "-$($entry.Key)"
            $relaunchArgs += "`"$($entry.Value)`""
        }
    }
    $relaunchArgs += "-NoElevate"
    return $relaunchArgs
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

function Install-WithWinget {
    param(
        [string]$Id,
        [string]$Name,
        [string]$Override = ""
    )

    if (-not (Test-CommandAvailable winget)) {
        throw "winget is required. Install App Installer from Microsoft Store first."
    }

    if (-not $Override) {
        $installed = & winget list --id $Id --exact --source winget --accept-source-agreements 2>&1 | Out-String
        if ($LASTEXITCODE -eq 0 -and $installed -match [regex]::Escape($Id)) {
            Write-Info "$Name is already installed. Skipping."
            return
        }
    }

    Write-Info "Installing $Name..."
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

    Invoke-WithRetry -Name "winget install $Name" -Action {
        $output = & winget @wingetArgs 2>&1 | Out-String
        if ($output) {
            Write-Info ($output.Trim())
        }
        
        # Exit code 0 = success, -1978335694 or other codes = already installed/nothing to do
        if ($LASTEXITCODE -eq 0) {
            Write-Info "$Name installation completed."
        } elseif ($LASTEXITCODE -eq -1978335694 -or $output -match "Found an existing package|already installed") {
            Write-Info "$Name is already installed. Skipping."
        } else {
            throw "winget install failed for $Name ($Id) with exit code $LASTEXITCODE"
        }
    }
    Update-CurrentPath
}

function Find-InnoSetupCompiler {
    if ($env:INNO_SETUP_DIR) {
        $compiler = Join-Path $env:INNO_SETUP_DIR "ISCC.exe"
        if (Test-Path -LiteralPath $compiler) {
            return $compiler
        }
    }

    $candidates = @(
        (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6"),
        (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup"),
        "C:\Program Files\Inno Setup 6",
        "C:\Program Files (x86)\Inno Setup 6",
        "C:\Program Files\Inno Setup",
        "C:\Program Files (x86)\Inno Setup",
        "C:\Program Files\Inno Setup 5",
        "C:\Program Files (x86)\Inno Setup 5"
    )
    
    foreach ($path in $candidates) {
        if (Test-Path -LiteralPath $path) {
            $compiler = Join-Path $path "ISCC.exe"
            if (Test-Path -LiteralPath $compiler) {
                Write-Info "Found Inno Setup at: $path"
                return $compiler
            }
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
                        Write-Info "Found Inno Setup in registry at: $($install.InstallLocation)"
                        return $compiler
                    }
                }
            }
        }
    } catch {
        # Silently continue if registry search fails
    }
    
    # Try to find via command
    if (Test-CommandAvailable iscc) {
        Write-Info "Found Inno Setup via PATH"
        return "iscc"
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

function Ensure-BuildDependencies {
    Write-Step "Installing required build tools"

    if (-not (Test-CommandAvailable git)) {
        Install-WithWinget -Id "Git.Git" -Name "Git"
    } else {
        Write-Info "Git is already installed."
    }
    
    if (-not (Test-CommandAvailable go)) {
        Install-WithWinget -Id "GoLang.Go" -Name "Go"
    } else {
        Write-Info "Go is already installed."
    }
    
    if (-not (Test-CommandAvailable node) -or -not (Test-CommandAvailable npm)) {
        Install-WithWinget -Id "OpenJS.NodeJS.LTS" -Name "Node.js LTS"
    } else {
        Write-Info "Node.js is already installed."
    }
    
    if (-not (Test-CommandAvailable cmake)) {
        Install-WithWinget -Id "Kitware.CMake" -Name "CMake"
    } else {
        Write-Info "CMake is already installed."
    }
    
    $innoSetup = Find-InnoSetupCompiler
    if (-not $innoSetup) {
        Write-Info "Inno Setup not found. Installing..."
        Install-WithWinget -Id "JRSoftware.InnoSetup" -Name "Inno Setup"
        # Re-check after installation
        $innoSetup = Find-InnoSetupCompiler
        if (-not $innoSetup) {
            throw "Inno Setup installation failed - compiler not found after install attempt"
        }
    } else {
        Write-Info "Inno Setup is already installed at: $innoSetup"
    }
    $env:INNO_SETUP_DIR = Split-Path -Parent $innoSetup
    
    $vsTools = Find-VSBuildTools
    if (-not $vsTools) {
        Write-Info "Visual Studio Build Tools not found. Installing..."
        Install-WithWinget `
            -Id "Microsoft.VisualStudio.2022.BuildTools" `
            -Name "Visual Studio 2022 Build Tools" `
            -Override "--wait --quiet --norestart --installWhileDownloading --add Microsoft.VisualStudio.Component.VC.Tools.x86.x64"
        $vsTools = Find-VSBuildTools
        if (-not $vsTools) {
            throw "Visual Studio Build Tools installation failed - C++ compiler was not found after install attempt"
        }
    } else {
        Write-Info "Visual Studio Build Tools are already installed at: $vsTools"
    }

    Update-CurrentPath

    if (-not (Import-AliVSDeveloperEnvironment -InstallRoot $vsTools)) {
        Write-Info "Visual Studio Build Tools are present but the C++/Windows SDK environment is incomplete. Installing required components..."
        if (-not (Ensure-AliVSBuildEnvironment -InstallRoot $vsTools -InstallWindowsSdk)) {
            Update-CurrentPath
            throw "Visual Studio C++ developer environment is not ready after component installation. Check network access to Microsoft download domains and retry."
        }
    }

    Write-Info "Visual Studio C++ build environment is ready."
}

function Download-File {
    param(
        [string]$Url,
        [string]$OutFile
    )

    Write-Info "Downloading $Url"
    Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $OutFile
    if (-not (Test-Path -LiteralPath $OutFile)) {
        throw "Download failed: $Url"
    }
}

function Build-InstallerFromSource {
    param([string]$Root)

    Write-Step "Building Ali installer from source"

    $builder = Join-Path $Root "scripts\build_installer.ps1"
    if (-not (Test-Path -LiteralPath $builder)) {
        throw "build_installer.ps1 not found in source tree: $Root"
    }

    $buildArgs = @("-ExecutionPolicy", "Bypass", "-File", $builder)
    if ($CpuOnly) {
        $buildArgs += "-CpuOnly"
    }

    Push-Location $Root
    try {
        & powershell @buildArgs
        if ($LASTEXITCODE -ne 0) {
            throw "Installer build failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }

    $installer = Join-Path $Root "dist\AliSetup.exe"
    if (-not (Test-Path -LiteralPath $installer)) {
        throw "Installer was not produced: $installer"
    }
    return $installer
}

function Install-Ali {
    param([string]$Installer)

    Write-Step "Installing Ali"

    $installerArgs = "/VERYSILENT /NORESTART /SUPPRESSMSGBOXES"
    if ($InstallDir) {
        $installerArgs += " /DIR=`"$InstallDir`""
    }

    $proc = Start-Process -FilePath $Installer -ArgumentList $installerArgs -PassThru
    $proc.WaitForExit()
    if ($proc.ExitCode -ne 0) {
        throw "Ali installer failed with exit code $($proc.ExitCode)"
    }
}

function Get-AliInstallDir {
    if ($InstallDir) {
        return $InstallDir
    }
    return Join-Path $env:LOCALAPPDATA "Programs\Ali"
}

function Test-AliServer {
    try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri "http://127.0.0.1:11434/api/version" -TimeoutSec 2
        return $response.StatusCode -ge 200 -and $response.StatusCode -lt 300
    } catch {
        return $false
    }
}

function Start-AliServer {
    Write-Step "Starting local Ali server"

    $dir = Get-AliInstallDir
    $ollama = Join-Path $dir "ollama.exe"
    if (-not (Test-Path -LiteralPath $ollama)) {
        throw "Installed server executable not found: $ollama"
    }

    if (-not (Test-AliServer)) {
        Start-Process -FilePath $ollama -ArgumentList "serve" -WorkingDirectory $dir -WindowStyle Hidden | Out-Null
    }

    $deadline = (Get-Date).AddSeconds(45)
    do {
        Start-Sleep -Milliseconds 750
        if (Test-AliServer) {
            Write-Info "Ali server is ready."
            return $ollama
        }
    } while ((Get-Date) -lt $deadline)

    throw "Ali server did not become ready at http://127.0.0.1:11434"
}

function Pull-LocalModel {
    param([string]$OllamaExe)

    if ($SkipModel -or -not $PreloadModel -or -not $Model) {
        Write-Info "Skipping model download. A model will only be downloaded after the user explicitly requests one."
        return
    }

    Write-Step "Downloading local AI model: $Model"
    Invoke-WithRetry -Name "model download $Model" -Action {
        & $OllamaExe pull $Model
        if ($LASTEXITCODE -ne 0) {
            throw "Model download failed: $Model"
        }
    }
}

function Invoke-ReleaseVerification {
    if ($SkipVerify) {
        Write-Info "Skipping release verification."
        return
    }

    Write-Step "Verifying installed release"
    $sourceRoot = if ($SourceDir) { (Resolve-Path $SourceDir).Path } else { (Resolve-Path (Join-Path $PSScriptRoot "..")).Path }
    $verify = Join-Path $sourceRoot "scripts\verify_release.ps1"
    if (-not (Test-Path -LiteralPath $verify)) {
        throw "Release verification script not found: $verify"
    }

    $verifyArgs = @(
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", $verify,
        "-InstallDir", (Get-AliInstallDir),
        "-StartServer"
    )
    if ($PreloadModel -and -not $SkipModel -and $Model) {
        $verifyArgs += @("-Model", $Model, "-RequireModel")
    }

    & powershell.exe @verifyArgs
    if ($LASTEXITCODE -ne 0) {
        throw "Release verification failed with exit code $LASTEXITCODE"
    }
}

function Open-AliIDE {
    if ($NoOpenIDE) {
        return
    }

    Write-Step "Opening Ali IDE"
    $dir = Get-AliInstallDir
    $launcher = Join-Path $dir "open-ide.ps1"
    if (Test-Path -LiteralPath $launcher) {
        Start-Process -FilePath "powershell.exe" -ArgumentList @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "`"$launcher`"") | Out-Null
    } else {
        Start-Process "http://127.0.0.1:11434/ide"
    }
}

try {
    Set-Content -LiteralPath $LogFile -Value "Ali full install started: $(Get-Date -Format o)"

    if ($PSCommandPath -and -not $NoElevate -and -not (Test-IsAdmin)) {
        Write-Host "Requesting administrator privileges for tool installation..." -ForegroundColor Yellow
        Start-Process -FilePath "powershell.exe" -Verb RunAs -ArgumentList (Get-RelaunchArguments)
        exit 0
    }

    if (-not $InstallerUrl -and $env:ALI_INSTALLER_URL) {
        $InstallerUrl = $env:ALI_INSTALLER_URL
    }

    $installer = ""
    if ($InstallerUrl) {
        Write-Step "Downloading Ali installer"
        $installer = Join-Path $env:TEMP "AliSetup.exe"
        Download-File -Url $InstallerUrl -OutFile $installer
    } else {
        if (-not $SourceDir) {
            $SourceDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
        } else {
            $SourceDir = (Resolve-Path $SourceDir).Path
        }

        Ensure-BuildDependencies
        $installer = Build-InstallerFromSource -Root $SourceDir
    }

    Install-Ali -Installer $installer
    $ollama = Start-AliServer
    Pull-LocalModel -OllamaExe $ollama
    Invoke-ReleaseVerification
    Open-AliIDE

    Write-Host ""
    Write-Host "Ali is installed and ready. Log: $LogFile" -ForegroundColor Green
} catch {
    Write-Host ""
    Write-Host "Ali full install failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Log: $LogFile" -ForegroundColor Yellow
    SafeAddContent -Message "FAILED: $($_.Exception.Message)"
    exit 1
}
