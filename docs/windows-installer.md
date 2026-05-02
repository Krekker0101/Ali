# Windows Installer Build

This project ships a reproducible Windows installer pipeline around the existing
`scripts/build_windows.ps1` and `app/ollama.iss` files.

## One Command Build

From the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_installer.ps1
```

For a faster CPU-only installer:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_installer.ps1 -CpuOnly
```

To install build dependencies first:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_installer.ps1 -Bootstrap -CpuOnly
```

The output is:

```text
dist\AliSetup.exe
```

The script validates the installer exists, has non-zero size, and prints its
SHA256 hash.

## Professional Web Installer

To build a single EXE bootstrapper that embeds the current source tree and
downloads/builds everything it needs during installation:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1
```

The output is:

```text
dist\AliWebSetup.exe
dist\AliWebSetup.manifest.json
```

`AliWebSetup.exe` opens a visible `Ali Setup` progress window, elevates once
through Windows UAC when needed, extracts the embedded source, installs missing
build tools with `winget`, builds `AliSetup.exe`, installs Ali silently, starts
the server, runs the post-install smoke verification, and opens the IDE.

Models are not downloaded during installation by default. The user can choose a
model later in the IDE/CLI; no model download starts just because the app was
installed.

To build a setup that preloads a model anyway:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1 -PreloadModel -Model qwen2.5-coder:1.5b
```

The setup cannot bypass Windows UAC or SmartScreen. For a public release, sign
`dist\AliWebSetup.exe` with a trusted code-signing certificate after building.

## Full Online Install From Source

For a fully automated install on a fresh Windows machine, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1 -CpuOnly
```

This script:

1. Installs missing build tools through `winget`.
2. Builds `dist\AliSetup.exe`.
3. Runs the installer silently.
4. Starts the local server.
5. Verifies `/api/version`, `/ide`, `/api/v1/ide/health`, and IDE settings.
6. Opens `http://127.0.0.1:11434/ide`.

Model downloads are deferred by default. To explicitly preload a model during
installation:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1 -CpuOnly -PreloadModel -Model qwen2.5-coder:1.5b
```

To preload another model:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1 -CpuOnly -PreloadModel -Model llama3.2
```

To use a prebuilt installer URL instead of building from source:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\full_install_windows.ps1 -InstallerUrl https://example.com/AliSetup.exe
```

To run only the installed-release smoke test:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\verify_release.ps1 -StartServer -Model qwen2.5-coder:1.5b -RequireModel
```

The smoke report is written to:

```text
%TEMP%\ali-release-verify.json
```

## Required Tools

The preflight checks for:

- Git
- Go and gofmt
- CMake
- Node.js and npm
- Inno Setup compiler (`ISCC.exe`)
- Visual Studio C++ Build Tools

Run only the preflight:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_installer.ps1 -PreflightOnly
```

## Installed IDE Launcher

The installer now includes `open-ide.ps1` and creates an `Ali IDE` Start Menu
shortcut. The launcher:

1. Checks `http://127.0.0.1:11434/api/version`.
2. Starts `ollama.exe serve` hidden if the local server is not running.
3. Waits for readiness.
4. Opens `http://127.0.0.1:11434/ide`.

The launcher does not download models. Users can pull models through the CLI or
existing model APIs after installation.
