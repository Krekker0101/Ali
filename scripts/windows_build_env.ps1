<#
.SYNOPSIS
    Shared Windows build environment helpers for installer scripts.

.DESCRIPTION
    Detects Visual Studio Build Tools, imports a C++ developer environment into
    the current PowerShell process, and can repair an incomplete Build Tools
    installation by adding the VC toolchain, CMake support, and Windows SDK.
#>

function Find-AliVSInstallRoot {
    $vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
    if (Test-Path -LiteralPath $vswhere) {
        foreach ($query in @(
            @("-latest", "-products", "*", "-requires", "Microsoft.VisualStudio.Component.VC.Tools.x86.x64", "-property", "installationPath"),
            @("-latest", "-products", "*", "-property", "installationPath")
        )) {
            $install = & $vswhere @query 2>$null | Select-Object -First 1
            if ($LASTEXITCODE -eq 0 -and $install -and (Test-Path -LiteralPath $install)) {
                return $install
            }
        }
    }

    foreach ($path in @(
        "${env:ProgramFiles(x86)}\Microsoft Visual Studio\2022\BuildTools",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\BuildTools",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Community",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Professional",
        "${env:ProgramFiles}\Microsoft Visual Studio\2022\Enterprise"
    )) {
        if ($path -and (Test-Path -LiteralPath $path)) {
            return $path
        }
    }

    return ""
}

function Find-AliMSVCToolsDir {
    param([string]$InstallRoot = "")

    if (-not $InstallRoot) {
        $InstallRoot = Find-AliVSInstallRoot
    }
    if (-not $InstallRoot) {
        return ""
    }

    $toolsRoot = Join-Path $InstallRoot "VC\Tools\MSVC"
    if (-not (Test-Path -LiteralPath $toolsRoot)) {
        return ""
    }

    $tools = Get-ChildItem -LiteralPath $toolsRoot -Directory -ErrorAction SilentlyContinue |
        Sort-Object Name -Descending |
        Where-Object { Test-Path -LiteralPath (Join-Path $_.FullName "bin\Hostx64\x64\cl.exe") } |
        Select-Object -First 1

    if ($tools) {
        return $tools.FullName
    }

    return ""
}

function Find-AliWindowsSdk {
    foreach ($root in @(
        "${env:ProgramFiles(x86)}\Windows Kits\10",
        "${env:ProgramFiles(x86)}\Windows Kits\11"
    )) {
        if (-not $root -or -not (Test-Path -LiteralPath $root)) {
            continue
        }

        $includeRoot = Join-Path $root "Include"
        if (-not (Test-Path -LiteralPath $includeRoot)) {
            continue
        }

        $version = Get-ChildItem -LiteralPath $includeRoot -Directory -ErrorAction SilentlyContinue |
            Sort-Object Name -Descending |
            Where-Object {
                (Test-Path -LiteralPath (Join-Path $_.FullName "um\windows.h")) -and
                (Test-Path -LiteralPath (Join-Path $root "Lib\$($_.Name)\um\x64\kernel32.lib")) -and
                (Test-Path -LiteralPath (Join-Path $root "Lib\$($_.Name)\ucrt\x64\ucrt.lib"))
            } |
            Select-Object -First 1

        if ($version) {
            return [pscustomobject]@{
                Root = $root
                Version = $version.Name
            }
        }
    }

    return $null
}

function Test-AliVSBuildEnvironment {
    $cl = Get-Command cl.exe -ErrorAction SilentlyContinue
    $nmake = Get-Command nmake.exe -ErrorAction SilentlyContinue
    $sdk = Find-AliWindowsSdk
    return [bool]($cl -and $nmake -and $sdk)
}

function Write-AliBuildEnvMessage {
    param([string]$Message)

    if (Get-Command Write-Info -ErrorAction SilentlyContinue) {
        Write-Info $Message
    } else {
        Write-Host $Message
    }
}

function Import-AliEnvironmentFromBatch {
    param(
        [string]$BatchPath,
        [string]$BatchArgs = ""
    )

    if (-not (Test-Path -LiteralPath $BatchPath)) {
        return $false
    }

    $tempFile = [System.IO.Path]::GetTempFileName()
    try {
        $cmd = "`"$BatchPath`" $BatchArgs >nul && set > `"$tempFile`""
        $oldErrorActionPreference = $ErrorActionPreference
        $ErrorActionPreference = "Continue"
        try {
            & cmd.exe /d /s /c $cmd 2>&1 | Out-Null
            $exitCode = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $oldErrorActionPreference
        }

        if ($exitCode -ne 0 -or -not (Test-Path -LiteralPath $tempFile)) {
            return $false
        }

        foreach ($line in Get-Content -LiteralPath $tempFile -ErrorAction SilentlyContinue) {
            $separator = $line.IndexOf("=")
            if ($separator -le 0) {
                continue
            }

            $name = $line.Substring(0, $separator)
            $value = $line.Substring($separator + 1)
            if ($name.StartsWith("=")) {
                continue
            }

            Set-Item -Path "Env:$name" -Value $value
        }
    } finally {
        Remove-Item -LiteralPath $tempFile -Force -ErrorAction SilentlyContinue
    }

    return (Test-AliVSBuildEnvironment)
}

function Import-AliVSDeveloperEnvironment {
    param([string]$InstallRoot = "")

    if (-not $InstallRoot) {
        $InstallRoot = Find-AliVSInstallRoot
    }
    if (-not $InstallRoot) {
        return $false
    }

    if (Test-AliVSBuildEnvironment) {
        return $true
    }

    $vcvars = Join-Path $InstallRoot "VC\Auxiliary\Build\vcvars64.bat"
    if (Import-AliEnvironmentFromBatch -BatchPath $vcvars) {
        return $true
    }

    $devCmd = Join-Path $InstallRoot "Common7\Tools\VsDevCmd.bat"
    if (Import-AliEnvironmentFromBatch -BatchPath $devCmd -BatchArgs "-no_logo -arch=amd64 -host_arch=amd64") {
        return $true
    }

    $msvcDir = Find-AliMSVCToolsDir -InstallRoot $InstallRoot
    if (-not $msvcDir) {
        return $false
    }

    $bin = Join-Path $msvcDir "bin\Hostx64\x64"
    if (Test-Path -LiteralPath $bin) {
        $env:Path = "$bin;$env:Path"
    }

    $env:VSINSTALLDIR = "$InstallRoot\"
    $env:VCINSTALLDIR = Join-Path $InstallRoot "VC\"
    $env:VCToolsInstallDir = "$msvcDir\"
    $env:VisualStudioVersion = "17.0"
    $env:Platform = "x64"

    $sdk = Find-AliWindowsSdk
    if ($sdk) {
        $sdkBin = Join-Path $sdk.Root "bin\$($sdk.Version)\x64"
        if (Test-Path -LiteralPath $sdkBin) {
            $env:Path = "$sdkBin;$env:Path"
        }

        $env:WindowsSdkDir = "$($sdk.Root)\"
        $env:WindowsSDKVersion = "$($sdk.Version)\"
        $env:UCRTVersion = $sdk.Version

        $include = @(
            (Join-Path $msvcDir "include"),
            (Join-Path $sdk.Root "Include\$($sdk.Version)\ucrt"),
            (Join-Path $sdk.Root "Include\$($sdk.Version)\um"),
            (Join-Path $sdk.Root "Include\$($sdk.Version)\shared"),
            (Join-Path $sdk.Root "Include\$($sdk.Version)\winrt"),
            (Join-Path $sdk.Root "Include\$($sdk.Version)\cppwinrt"),
            $env:INCLUDE
        ) | Where-Object { $_ -and (Test-Path -LiteralPath $_ -ErrorAction SilentlyContinue) -or ($_ -eq $env:INCLUDE -and $_) }

        $lib = @(
            (Join-Path $msvcDir "lib\x64"),
            (Join-Path $sdk.Root "Lib\$($sdk.Version)\ucrt\x64"),
            (Join-Path $sdk.Root "Lib\$($sdk.Version)\um\x64"),
            $env:LIB
        ) | Where-Object { $_ -and (Test-Path -LiteralPath $_ -ErrorAction SilentlyContinue) -or ($_ -eq $env:LIB -and $_) }

        $env:INCLUDE = ($include -join ";")
        $env:LIB = ($lib -join ";")
        $env:LIBPATH = ($lib -join ";")
    }

    return (Test-AliVSBuildEnvironment)
}

function Invoke-AliVSBuildToolsRepair {
    param([string]$InstallRoot = "")

    if (-not $InstallRoot) {
        $InstallRoot = Find-AliVSInstallRoot
    }
    if (-not $InstallRoot) {
        throw "Visual Studio Build Tools installation was not found."
    }

    $setup = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\setup.exe"
    if (-not (Test-Path -LiteralPath $setup)) {
        throw "Visual Studio Installer was not found: $setup"
    }

    $args = @(
        "modify",
        "--installPath", $InstallRoot,
        "--quiet",
        "--wait",
        "--norestart",
        "--installWhileDownloading",
        "--add", "Microsoft.VisualStudio.Component.VC.Tools.x86.x64",
        "--add", "Microsoft.VisualStudio.Component.Windows10SDK.19041",
        "--remove", "Microsoft.VisualStudio.Workload.VCTools",
        "--remove", "Microsoft.VisualStudio.Component.VC.ASAN",
        "--remove", "Microsoft.VisualStudio.Component.TestTools.BuildTools",
        "--remove", "Microsoft.VisualStudio.Component.TextTemplating",
        "--remove", "Microsoft.VisualStudio.Component.Vcpkg",
        "--remove", "Microsoft.VisualStudio.Component.VC.CMake.Project",
        "--remove", "Microsoft.VisualStudio.Component.VC.CoreIde",
        "--remove", "Microsoft.VisualStudio.ComponentGroup.NativeDesktop.Core",
        "--remove", "Microsoft.VisualStudio.Component.Windows11SDK.26100"
    )

    Write-AliBuildEnvMessage "Repairing Visual Studio Build Tools with minimal C++ components..."
    $output = & $setup @args 2>&1 | Out-String
    if ($output.Trim()) {
        Write-AliBuildEnvMessage ($output.Trim())
    }
    if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 3010) {
        throw "Visual Studio Build Tools repair failed with exit code $LASTEXITCODE"
    }
}

function Install-AliWindowsSdkWithWinget {
    if (Find-AliWindowsSdk) {
        return $true
    }

    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        Write-AliBuildEnvMessage "winget was not found, cannot install standalone Windows SDK."
        return $false
    }

    $sdkIds = @(
        "Microsoft.WindowsSDK.10.0.26100",
        "Microsoft.WindowsSDK.10.0.22621",
        "Microsoft.WindowsSDK.10.0.19041"
    )

    foreach ($id in $sdkIds) {
        for ($attempt = 1; $attempt -le 3; $attempt++) {
            Write-AliBuildEnvMessage "Installing standalone Windows SDK ($id), attempt $attempt/3..."
            $args = @(
                "install",
                "--id", $id,
                "--source", "winget",
                "--exact",
                "--silent",
                "--disable-interactivity",
                "--accept-package-agreements",
                "--accept-source-agreements"
            )

            $output = & winget @args 2>&1 | Out-String
            if ($output.Trim()) {
                Write-AliBuildEnvMessage ($output.Trim())
            }

            if ($LASTEXITCODE -eq 0 -or $output -match "Found an existing package|already installed|No available upgrade|No newer package") {
                Start-Sleep -Seconds 2
                if (Find-AliWindowsSdk) {
                    return $true
                }
            }

            if ($attempt -lt 3) {
                Start-Sleep -Seconds ([Math]::Min(20, 3 * $attempt))
            }
        }
    }

    return [bool](Find-AliWindowsSdk)
}

function Ensure-AliVSBuildEnvironment {
    param(
        [string]$InstallRoot = "",
        [switch]$InstallWindowsSdk
    )

    if (-not $InstallRoot) {
        $InstallRoot = Find-AliVSInstallRoot
    }

    if (Import-AliVSDeveloperEnvironment -InstallRoot $InstallRoot) {
        return $true
    }

    if ($InstallWindowsSdk -and -not (Find-AliWindowsSdk)) {
        Install-AliWindowsSdkWithWinget | Out-Null
        if (Import-AliVSDeveloperEnvironment -InstallRoot $InstallRoot) {
            return $true
        }
    }

    $msvcDir = Find-AliMSVCToolsDir -InstallRoot $InstallRoot
    if ($msvcDir -and -not (Find-AliWindowsSdk)) {
        Write-AliBuildEnvMessage "MSVC tools are installed, but Windows SDK is still missing."
        return $false
    }

    if ($InstallRoot) {
        Invoke-AliVSBuildToolsRepair -InstallRoot $InstallRoot
        Start-Sleep -Seconds 2
        if (Import-AliVSDeveloperEnvironment -InstallRoot $InstallRoot) {
            return $true
        }
    }

    if ($InstallWindowsSdk -and -not (Find-AliWindowsSdk)) {
        Install-AliWindowsSdkWithWinget | Out-Null
        if (Import-AliVSDeveloperEnvironment -InstallRoot $InstallRoot) {
            return $true
        }
    }

    return $false
}
