# Ali Release Readiness

This document captures the release path for the AI-powered IDE build.

## Release Artifacts

- `dist\AliSetup.exe`: offline app installer produced by the existing Windows
  build pipeline.
- `dist\AliWebSetup.exe`: web/bootstrap installer that embeds the current source
  and installs missing build prerequisites during setup.
- `dist\AliWebSetup.manifest.json`: release metadata with size, SHA256 hash,
  model preload status, and installer behavior.

## Stable Release Checklist

1. Build the web installer:

   ```powershell
   powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1
   ```

2. Test `dist\AliWebSetup.exe` in a clean Windows VM.

3. Confirm the smoke report passes:

   ```text
   %TEMP%\ali-release-verify.json
   ```

4. Sign the final EXE with a trusted certificate before public distribution.

5. Publish the SHA256 from `dist\AliWebSetup.manifest.json` next to the release.

## Runtime Guarantees

- Existing backend APIs remain unchanged.
- IDE filesystem APIs are under `/api/v1/ide/*`.
- IDE filesystem APIs are loopback-only unless `OLLAMA_IDE_ALLOW_REMOTE=true`.
- Agent file tools prepare diffs; they do not mutate files until the user applies
  changes.
- Delete operations require explicit confirmation.
- Non-secret IDE settings persist between launches. API keys are not written to
  the settings file.
- The web installer does not download AI models by default. Model downloads are
  deferred until explicit user action, or until a release is intentionally built
  with `-PreloadModel`.

## Required Manual Verification

Automated scripts can validate local behavior, but a public release still needs
manual checks on a clean machine:

- Windows UAC prompt appears once and is understandable.
- SmartScreen warning is acceptable or eliminated through code signing.
- `winget` package installs are not blocked by policy.
- If a preload build is used, the chosen local model downloads on the target
  network.
- Start Menu shortcuts launch both the app and `Ali IDE`.
