<#
.SYNOPSIS
    Build AliWebSetup.exe, a self-contained online bootstrap installer.

.DESCRIPTION
    Creates a Windows EXE with the current source tree embedded as source.zip.
    When launched, the EXE silently extracts the source, elevates once for tool
    installation, installs build prerequisites with winget, builds AliSetup.exe,
    installs Ali, starts the local server, and opens the IDE. It does not
    download an AI model during installation unless -PreloadModel is explicitly
    provided.

    The generated EXE does not bundle third-party build tools or models. It
    downloads build tools during installation through the existing Windows
    package manager. Model downloads are deferred to explicit user action.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1 -PreloadModel -Model qwen2.5-coder:1.5b

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\build_web_installer_exe.ps1 -GpuBuild
#>

[CmdletBinding()]
param(
    [string]$OutputPath = "",
    [string]$Version = "1.0.0.0",
    [string]$Model = "",
    [switch]$PreloadModel,
    [switch]$SkipModel,
    [switch]$NoOpenIDE,
    [switch]$GpuBuild,
    [switch]$KeepStaging
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$DistDir = Join-Path $RepoRoot "dist"
if (-not $OutputPath) {
    $OutputPath = Join-Path $DistDir "AliWebSetup.exe"
}
$OutputPath = [System.IO.Path]::GetFullPath($OutputPath)

$StagingRoot = Join-Path $DistDir "web-installer"
$SourceStage = Join-Path $StagingRoot "source"
$PayloadDir = Join-Path $StagingRoot "payload"
$SourceZip = Join-Path $PayloadDir "source.zip"
$BootstrapCmd = Join-Path $PayloadDir "bootstrap.cmd"
$BootstrapPs1 = Join-Path $PayloadDir "bootstrap.ps1"
$BootstrapCs = Join-Path $PayloadDir "AliWebSetup.cs"
$ReleaseManifest = [System.IO.Path]::ChangeExtension($OutputPath, ".manifest.json")

function Write-Section {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Assert-UnderPath {
    param(
        [string]$Path,
        [string]$Parent
    )

    $fullPath = [System.IO.Path]::GetFullPath($Path).TrimEnd('\')
    $fullParent = [System.IO.Path]::GetFullPath($Parent).TrimEnd('\')
    if (-not $fullPath.StartsWith($fullParent, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to operate outside expected directory. Path: $fullPath Parent: $fullParent"
    }
}

function New-BootstrapScript {
    param([string]$Path)

    $cpuOnlyValue = if ($GpuBuild) { "False" } else { "True" }
    $skipModelValue = if ($SkipModel) { "True" } else { "False" }
    $preloadModelValue = if ($PreloadModel) { "True" } else { "False" }
    $noOpenIDEValue = if ($NoOpenIDE) { "True" } else { "False" }
    $modelValue = $Model.Replace("'", "''")

    $content = @'
[CmdletBinding()]
param(
    [string]$PayloadZip = "",
    [string]$WorkRoot = "",
    [switch]$Elevated
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Model = '@@MODEL@@'
$SkipModel = [System.Convert]::ToBoolean('@@SKIP_MODEL@@')
$PreloadModel = [System.Convert]::ToBoolean('@@PRELOAD_MODEL@@')
$NoOpenIDE = [System.Convert]::ToBoolean('@@NO_OPEN_IDE@@')
$CpuOnly = [System.Convert]::ToBoolean('@@CPU_ONLY@@')

function Test-IsAdmin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Add-QuotedArg {
    param(
        [System.Collections.Generic.List[string]]$ArgumentList,
        [string]$Value
    )
    $ArgumentList.Add("`"$Value`"")
}

function Assert-UnderPath {
    param(
        [string]$Path,
        [string]$Parent
    )

    $fullPath = [System.IO.Path]::GetFullPath($Path).TrimEnd('\')
    $fullParent = [System.IO.Path]::GetFullPath($Parent).TrimEnd('\')
    if (-not $fullPath.StartsWith($fullParent, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to operate outside expected directory. Path: $fullPath Parent: $fullParent"
    }
}

if (-not $WorkRoot) {
    $WorkRoot = Join-Path $env:TEMP ("AliWebSetup-" + [guid]::NewGuid().ToString("N"))
}
$WorkRoot = [System.IO.Path]::GetFullPath($WorkRoot)
New-Item -ItemType Directory -Force -Path $WorkRoot | Out-Null

$LogFile = Join-Path $WorkRoot "install.log"
Set-Content -LiteralPath $LogFile -Value "Ali web setup started: $(Get-Date -Format o)" -ErrorAction SilentlyContinue

function Write-BootstrapLog {
    param([string]$Message)
    try {
        Add-Content -LiteralPath $LogFile -Value $Message -ErrorAction SilentlyContinue
    } catch {
    }
}

try {
    Write-Host "Ali web setup started. Log: $LogFile"
    Write-BootstrapLog "Ali web setup started. Log: $LogFile"

    if (-not $PayloadZip) {
        $PayloadZip = Join-Path $PSScriptRoot "source.zip"
    }
    if (-not (Test-Path -LiteralPath $PayloadZip)) {
        throw "Payload source.zip not found: $PayloadZip"
    }

    $LocalZip = Join-Path $WorkRoot "source.zip"
    if ([System.IO.Path]::GetFullPath($PayloadZip) -ne [System.IO.Path]::GetFullPath($LocalZip)) {
        Copy-Item -LiteralPath $PayloadZip -Destination $LocalZip -Force
    }

    $LocalBootstrap = Join-Path $WorkRoot "bootstrap.ps1"
    if ($PSCommandPath -and ([System.IO.Path]::GetFullPath($PSCommandPath) -ne [System.IO.Path]::GetFullPath($LocalBootstrap))) {
        Copy-Item -LiteralPath $PSCommandPath -Destination $LocalBootstrap -Force
    } elseif (-not (Test-Path -LiteralPath $LocalBootstrap)) {
        throw "Unable to prepare bootstrap script in $WorkRoot"
    }

    if (-not $Elevated -and -not (Test-IsAdmin)) {
        Write-Host "Requesting administrator privileges..."
        $relaunchArgs = [System.Collections.Generic.List[string]]::new()
        $relaunchArgs.Add("-NoProfile")
        $relaunchArgs.Add("-ExecutionPolicy")
        $relaunchArgs.Add("Bypass")
        $relaunchArgs.Add("-File")
        Add-QuotedArg -ArgumentList $relaunchArgs -Value $LocalBootstrap
        $relaunchArgs.Add("-PayloadZip")
        Add-QuotedArg -ArgumentList $relaunchArgs -Value $LocalZip
        $relaunchArgs.Add("-WorkRoot")
        Add-QuotedArg -ArgumentList $relaunchArgs -Value $WorkRoot
        $relaunchArgs.Add("-Elevated")

        $proc = Start-Process -FilePath "powershell.exe" -Verb RunAs -WindowStyle Hidden -ArgumentList $relaunchArgs.ToArray() -PassThru
        $proc.WaitForExit()
        exit $proc.ExitCode
    }

    $SourceDir = Join-Path $WorkRoot "source"
    if (Test-Path -LiteralPath $SourceDir) {
        Assert-UnderPath -Path $SourceDir -Parent $WorkRoot
        Remove-Item -LiteralPath $SourceDir -Recurse -Force
    }

    Write-Host "Extracting embedded source..."
    Expand-Archive -LiteralPath $LocalZip -DestinationPath $SourceDir -Force

    $InstallScript = Join-Path $SourceDir "scripts\full_install_windows.ps1"
    if (-not (Test-Path -LiteralPath $InstallScript)) {
        throw "Full installer script not found: $InstallScript"
    }

    Write-Host "Installing Ali from embedded source..."
    $installArgs = @(
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", $InstallScript,
        "-SourceDir", $SourceDir,
        "-NoElevate"
    )
    if ($CpuOnly) {
        $installArgs += "-CpuOnly"
    }
    if ($SkipModel) {
        $installArgs += "-SkipModel"
    } elseif ($PreloadModel -and $Model) {
        $installArgs += @("-PreloadModel", "-Model", $Model)
    }
    if ($NoOpenIDE) {
        $installArgs += "-NoOpenIDE"
    }

    & powershell.exe @installArgs
    if ($LASTEXITCODE -ne 0) {
        throw "Ali installation failed with exit code $LASTEXITCODE"
    }

    Write-Host "Ali web setup completed successfully."
} catch {
    Write-BootstrapLog "ERROR: $($_.Exception.Message)"
    Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
    throw
}
'@

    $content = $content.Replace("@@MODEL@@", $modelValue)
    $content = $content.Replace("@@SKIP_MODEL@@", $skipModelValue)
    $content = $content.Replace("@@PRELOAD_MODEL@@", $preloadModelValue)
    $content = $content.Replace("@@NO_OPEN_IDE@@", $noOpenIDEValue)
    $content = $content.Replace("@@CPU_ONLY@@", $cpuOnlyValue)
    Set-Content -LiteralPath $Path -Value $content -Encoding ASCII
}

function New-BootstrapExecutableSource {
    param([string]$Path)

    $content = @'
using System;
using System.Diagnostics;
using System.Drawing;
using System.Drawing.Drawing2D;
using System.IO;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Security.Principal;
using System.Text;
using System.Threading;
using System.Windows.Forms;

[assembly: AssemblyTitle("Ali Web Setup")]
[assembly: AssemblyDescription("Ali online installer and IDE bootstrapper")]
[assembly: AssemblyCompany("Ali")]
[assembly: AssemblyProduct("Ali")]
[assembly: AssemblyCopyright("Copyright (c) Ali")]
[assembly: AssemblyVersion("@@VERSION@@")]
[assembly: AssemblyFileVersion("@@VERSION@@")]

internal static class AliWebSetup
{
    [STAThread]
    private static int Main(string[] args)
    {
        try
        {
            if (HasArg(args, "--self-test"))
            {
                return SelfTest();
            }

            if (!HasArg(args, "--elevated") && !IsAdministrator())
            {
                return RelaunchElevated();
            }

            Application.EnableVisualStyles();
            Application.SetCompatibleTextRenderingDefault(false);
            SetupForm form = new SetupForm();
            Application.Run(form);
            return form.ExitCode;
        }
        catch (Exception ex)
        {
            MessageBox.Show(
                "Ali installation could not start." + Environment.NewLine + ex.Message,
                "Ali Web Setup",
                MessageBoxButtons.OK,
                MessageBoxIcon.Error);
            return 1;
        }
    }

    private static bool HasArg(string[] args, string name)
    {
        foreach (string arg in args)
        {
            if (string.Equals(arg, name, StringComparison.OrdinalIgnoreCase))
            {
                return true;
            }
        }
        return false;
    }

    private static bool IsAdministrator()
    {
        WindowsIdentity identity = WindowsIdentity.GetCurrent();
        WindowsPrincipal principal = new WindowsPrincipal(identity);
        return principal.IsInRole(WindowsBuiltInRole.Administrator);
    }

    private static int RelaunchElevated()
    {
        try
        {
            ProcessStartInfo info = new ProcessStartInfo();
            info.FileName = Application.ExecutablePath;
            info.Arguments = "--elevated";
            info.Verb = "runas";
            info.UseShellExecute = true;
            using (Process process = Process.Start(info))
            {
                return 0;
            }
        }
        catch (Exception ex)
        {
            MessageBox.Show(
                "Administrator approval is required to install build tools." + Environment.NewLine + ex.Message,
                "Ali Web Setup",
                MessageBoxButtons.OK,
                MessageBoxIcon.Error);
            return 1;
        }
    }

    private static int SelfTest()
    {
        string workRoot = Path.Combine(Path.GetTempPath(), "AliWebSetupSelfTest-" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(workRoot);
        ExtractResource("bootstrap.ps1", Path.Combine(workRoot, "bootstrap.ps1"));
        ExtractResource("source.zip", Path.Combine(workRoot, "source.zip"));
        ExtractResource("app.ico", Path.Combine(workRoot, "app.ico"));
        return File.Exists(Path.Combine(workRoot, "bootstrap.ps1")) && File.Exists(Path.Combine(workRoot, "source.zip")) && File.Exists(Path.Combine(workRoot, "app.ico")) ? 0 : 1;
    }

    internal static void ExtractResource(string resourceName, string outputPath)
    {
        Assembly assembly = Assembly.GetExecutingAssembly();
        using (Stream input = assembly.GetManifestResourceStream(resourceName))
        {
            if (input == null)
            {
                throw new InvalidOperationException("Embedded resource was not found: " + resourceName);
            }

            using (FileStream output = File.Create(outputPath))
            {
                input.CopyTo(output);
            }
        }
    }

    internal static Icon LoadAppIcon()
    {
        Assembly assembly = Assembly.GetExecutingAssembly();
        using (Stream input = assembly.GetManifestResourceStream("app.ico"))
        {
            if (input == null)
            {
                return null;
            }

            return new Icon(input);
        }
    }

    internal static int RunPowerShell(string bootstrapPath, string sourceZipPath, string workRoot, Action<string> appendLog)
    {
        ProcessStartInfo info = new ProcessStartInfo();
        info.FileName = "powershell.exe";
        info.Arguments =
            "-NoProfile -ExecutionPolicy Bypass -File " + Quote(bootstrapPath) +
            " -PayloadZip " + Quote(sourceZipPath) +
            " -WorkRoot " + Quote(workRoot) +
            " -Elevated";
        info.UseShellExecute = false;
        info.CreateNoWindow = true;
        info.WindowStyle = ProcessWindowStyle.Hidden;
        info.RedirectStandardOutput = true;
        info.RedirectStandardError = true;

        using (Process process = Process.Start(info))
        {
            if (process == null)
            {
                throw new InvalidOperationException("Unable to start PowerShell.");
            }

            process.OutputDataReceived += delegate(object sender, DataReceivedEventArgs eventArgs)
            {
                if (!string.IsNullOrEmpty(eventArgs.Data))
                {
                    appendLog(eventArgs.Data);
                }
            };
            process.ErrorDataReceived += delegate(object sender, DataReceivedEventArgs eventArgs)
            {
                if (!string.IsNullOrEmpty(eventArgs.Data))
                {
                    appendLog(eventArgs.Data);
                }
            };
            process.BeginOutputReadLine();
            process.BeginErrorReadLine();
            process.WaitForExit();
            return process.ExitCode;
        }
    }

    private static string Quote(string value)
    {
        return "\"" + value.Replace("\"", "\\\"") + "\"";
    }
}

internal sealed class SetupForm : Form
{
    private readonly Color backgroundTop = Color.FromArgb(15, 18, 27);
    private readonly Color backgroundBottom = Color.FromArgb(4, 7, 14);
    private readonly Color glassFill = Color.FromArgb(46, 255, 255, 255);
    private readonly Color glassStroke = Color.FromArgb(82, 255, 255, 255);
    private readonly Color textPrimary = Color.FromArgb(245, 247, 255);
    private readonly Color textSecondary = Color.FromArgb(168, 176, 196);
    private readonly Color accent = Color.FromArgb(87, 153, 255);
    private readonly Color accent2 = Color.FromArgb(122, 92, 255);

    private readonly GlassPanel heroPanel;
    private readonly GlassPanel logPanel;
    private readonly Label titleLabel;
    private readonly Label subtitleLabel;
    private readonly Label statusLabel;
    private readonly Label logTitleLabel;
    private readonly Label badgeLabel;
    private readonly Label phaseLabel;
    private readonly ProgressRing progressRing;
    private readonly ProgressStrip progressStrip;
    private readonly TextBox logBox;
    private readonly Button closeButton;
    private readonly string workRoot;
    private readonly string installLog;
    private int progressValue;

    public int ExitCode { get; private set; }

    public SetupForm()
    {
        ExitCode = 1;
        workRoot = Path.Combine(Path.GetTempPath(), "AliWebSetup-" + Guid.NewGuid().ToString("N"));
        installLog = Path.Combine(workRoot, "install.log");

        Text = "Ali Setup";
        Width = 860;
        Height = 560;
        MinimumSize = new Size(700, 480);
        StartPosition = FormStartPosition.CenterScreen;
        Font = new Font("Segoe UI", 9F);
        BackColor = backgroundBottom;
        ForeColor = textPrimary;
        DoubleBuffered = true;
        Icon = AliWebSetup.LoadAppIcon();

        heroPanel = new GlassPanel();
        heroPanel.BackColor = Color.Transparent;
        heroPanel.FillColor = glassFill;
        heroPanel.StrokeColor = glassStroke;
        heroPanel.Radius = 22;

        logPanel = new GlassPanel();
        logPanel.BackColor = Color.Transparent;
        logPanel.FillColor = Color.FromArgb(34, 255, 255, 255);
        logPanel.StrokeColor = Color.FromArgb(58, 255, 255, 255);
        logPanel.Radius = 18;

        titleLabel = new Label();
        titleLabel.Text = "Ali Setup";
        titleLabel.Font = new Font("Segoe UI Semibold", 24F, FontStyle.Bold);
        titleLabel.ForeColor = textPrimary;
        titleLabel.BackColor = Color.Transparent;
        titleLabel.AutoSize = true;

        subtitleLabel = new Label();
        subtitleLabel.Text = "Installing the AI-powered IDE platform";
        subtitleLabel.Font = new Font("Segoe UI", 10.5F);
        subtitleLabel.ForeColor = textSecondary;
        subtitleLabel.BackColor = Color.Transparent;
        subtitleLabel.AutoSize = true;

        badgeLabel = new Label();
        badgeLabel.Text = "SECURE SETUP";
        badgeLabel.Font = new Font("Segoe UI Semibold", 8.5F, FontStyle.Bold);
        badgeLabel.ForeColor = Color.FromArgb(223, 232, 255);
        badgeLabel.BackColor = Color.FromArgb(42, 87, 153, 255);
        badgeLabel.TextAlign = ContentAlignment.MiddleCenter;

        statusLabel = new Label();
        statusLabel.Text = "Preparing installer...";
        statusLabel.AutoSize = false;
        statusLabel.Font = new Font("Segoe UI Semibold", 10.5F, FontStyle.Bold);
        statusLabel.ForeColor = textPrimary;
        statusLabel.BackColor = Color.Transparent;

        phaseLabel = new Label();
        phaseLabel.Text = "Initializing";
        phaseLabel.AutoSize = false;
        phaseLabel.Font = new Font("Segoe UI", 9F);
        phaseLabel.ForeColor = textSecondary;
        phaseLabel.BackColor = Color.Transparent;

        progressRing = new ProgressRing();
        progressRing.BackColor = Color.Transparent;
        progressRing.Accent = accent;
        progressRing.Accent2 = accent2;

        progressStrip = new ProgressStrip();
        progressStrip.BackColor = Color.Transparent;
        progressStrip.Accent = accent;
        progressStrip.Accent2 = accent2;

        logTitleLabel = new Label();
        logTitleLabel.Text = "Installation log";
        logTitleLabel.Font = new Font("Segoe UI Semibold", 10F, FontStyle.Bold);
        logTitleLabel.ForeColor = textPrimary;
        logTitleLabel.BackColor = Color.Transparent;
        logTitleLabel.AutoSize = true;

        logBox = new TextBox();
        logBox.Multiline = true;
        logBox.ReadOnly = true;
        logBox.ScrollBars = ScrollBars.Vertical;
        logBox.BorderStyle = BorderStyle.None;
        logBox.BackColor = Color.FromArgb(13, 17, 29);
        logBox.ForeColor = Color.FromArgb(206, 215, 236);
        logBox.Font = new Font("Consolas", 9F);

        closeButton = new Button();
        closeButton.Text = "Close";
        closeButton.Enabled = false;
        closeButton.FlatStyle = FlatStyle.Flat;
        closeButton.FlatAppearance.BorderSize = 0;
        closeButton.BackColor = Color.FromArgb(62, 98, 168, 255);
        closeButton.ForeColor = Color.White;
        closeButton.Font = new Font("Segoe UI Semibold", 9F, FontStyle.Bold);
        closeButton.Cursor = Cursors.Hand;
        closeButton.Click += delegate { Close(); };

        Controls.Add(heroPanel);
        Controls.Add(logPanel);
        Controls.Add(closeButton);
        heroPanel.Controls.Add(titleLabel);
        heroPanel.Controls.Add(subtitleLabel);
        heroPanel.Controls.Add(badgeLabel);
        heroPanel.Controls.Add(statusLabel);
        heroPanel.Controls.Add(phaseLabel);
        heroPanel.Controls.Add(progressRing);
        heroPanel.Controls.Add(progressStrip);
        logPanel.Controls.Add(logTitleLabel);
        logPanel.Controls.Add(logBox);

        SetProgress(2, "Initializing");
        LayoutControls();
    }

    protected override void OnPaint(PaintEventArgs e)
    {
        base.OnPaint(e);

        using (LinearGradientBrush brush = new LinearGradientBrush(ClientRectangle, backgroundTop, backgroundBottom, LinearGradientMode.Vertical))
        {
            e.Graphics.FillRectangle(brush, ClientRectangle);
        }

        e.Graphics.SmoothingMode = SmoothingMode.AntiAlias;
        using (SolidBrush glow = new SolidBrush(Color.FromArgb(50, accent)))
        {
            e.Graphics.FillEllipse(glow, Width - 270, -120, 390, 310);
        }
        using (SolidBrush glow = new SolidBrush(Color.FromArgb(38, accent2)))
        {
            e.Graphics.FillEllipse(glow, -130, Height - 250, 350, 310);
        }
    }

    protected override void OnResize(EventArgs e)
    {
        base.OnResize(e);
        LayoutControls();
        Invalidate();
    }

    private void LayoutControls()
    {
        int margin = 28;
        int heroHeight = 168;
        int buttonHeight = 38;
        int buttonWidth = 108;
        int bottom = 24;
        int ringSize = 92;

        heroPanel.SetBounds(margin, margin, ClientSize.Width - (margin * 2), heroHeight);
        titleLabel.SetBounds(28, 22, heroPanel.Width - 190, 38);
        subtitleLabel.SetBounds(30, 64, heroPanel.Width - 220, 24);
        badgeLabel.SetBounds(heroPanel.Width - 164, 24, 116, 26);
        progressRing.SetBounds(heroPanel.Width - ringSize - 42, 58, ringSize, ringSize);
        statusLabel.SetBounds(30, 104, heroPanel.Width - ringSize - 104, 28);
        phaseLabel.SetBounds(30, 130, heroPanel.Width - ringSize - 104, 22);
        progressStrip.SetBounds(30, 154, heroPanel.Width - ringSize - 104, 8);

        int logTop = heroPanel.Bottom + 18;
        int logBottom = ClientSize.Height - bottom - buttonHeight - 14;
        logPanel.SetBounds(margin, logTop, ClientSize.Width - (margin * 2), Math.Max(190, logBottom - logTop));
        logTitleLabel.SetBounds(22, 18, logPanel.Width - 44, 22);
        logBox.SetBounds(22, 50, logPanel.Width - 44, logPanel.Height - 72);

        closeButton.SetBounds(ClientSize.Width - margin - buttonWidth, ClientSize.Height - bottom - buttonHeight, buttonWidth, buttonHeight);
    }

    protected override void OnShown(EventArgs e)
    {
        base.OnShown(e);
        Thread worker = new Thread(RunInstallation);
        worker.IsBackground = true;
        worker.Start();
    }

    protected override void OnHandleCreated(EventArgs e)
    {
        base.OnHandleCreated(e);
        NativeChrome.TryUseImmersiveDarkMode(Handle);
    }

    private void RunInstallation()
    {
        try
        {
            Directory.CreateDirectory(workRoot);
            string bootstrapPath = Path.Combine(workRoot, "bootstrap.ps1");
            string sourceZipPath = Path.Combine(workRoot, "source.zip");

            SetProgress(6, "Preparing payload");
            SetStatus("Extracting embedded installer payload...");
            AliWebSetup.ExtractResource("bootstrap.ps1", bootstrapPath);
            AliWebSetup.ExtractResource("source.zip", sourceZipPath);
            SetProgress(12, "Payload extracted");

            AppendLog("Log file: " + installLog);
            AppendLog("Installing required components. This can take a while on a clean Windows machine.");
            SetProgress(15, "Starting installation");
            SetStatus("Installing Ali and required components...");

            int code = AliWebSetup.RunPowerShell(bootstrapPath, sourceZipPath, workRoot, AppendLog);
            ExitCode = code;
            if (code == 0)
            {
                SetProgress(100, "Complete");
                SetStatus("Ali is installed and ready.");
                AppendLog("Done.");
                EnableClose();
                MessageBox.Show("Ali is installed and ready.", "Ali Setup", MessageBoxButtons.OK, MessageBoxIcon.Information);
            }
            else
            {
                MarkProgressFailed();
                SetStatus("Installation failed. See the log path above.");
                AppendLog("Failed with exit code " + code + ".");
                EnableClose();
                MessageBox.Show("Ali installation failed. Log: " + installLog, "Ali Setup", MessageBoxButtons.OK, MessageBoxIcon.Error);
            }
        }
        catch (Exception ex)
        {
            ExitCode = 1;
            try
            {
                Directory.CreateDirectory(workRoot);
                File.AppendAllText(Path.Combine(workRoot, "launcher.log"), ex.ToString() + Environment.NewLine, Encoding.UTF8);
            }
            catch
            {
            }
            SetStatus("Installation could not start.");
            MarkProgressFailed();
            AppendLog(ex.ToString());
            EnableClose();
            MessageBox.Show("Ali installation could not start. Log: " + Path.Combine(workRoot, "launcher.log"), "Ali Setup", MessageBoxButtons.OK, MessageBoxIcon.Error);
        }
    }

    private void SetStatus(string text)
    {
        if (InvokeRequired)
        {
            BeginInvoke(new Action<string>(SetStatus), text);
            return;
        }
        statusLabel.Text = text;
    }

    private void SetProgress(int value, string phase)
    {
        if (InvokeRequired)
        {
            BeginInvoke(new Action<int, string>(SetProgress), value, phase);
            return;
        }

        value = Math.Max(progressValue, Math.Min(100, value));
        progressValue = value;
        phaseLabel.Text = phase;
        progressStrip.SetProgress(value);
        progressRing.SetProgress(value);
    }

    private void MarkProgressFailed()
    {
        if (InvokeRequired)
        {
            BeginInvoke(new Action(MarkProgressFailed));
            return;
        }

        progressStrip.MarkFailed();
        progressRing.MarkFailed();
        phaseLabel.Text = "Failed";
    }

    private void UpdateProgressFromLog(string text)
    {
        if (string.IsNullOrEmpty(text))
        {
            return;
        }

        string lower = text.ToLowerInvariant();
        if (lower.Contains("ali web setup started")) SetProgress(7, "Starting setup");
        else if (lower.Contains("extracting embedded source")) SetProgress(10, "Extracting source");
        else if (lower.Contains("installing ali from embedded source")) SetProgress(14, "Preparing source install");
        else if (lower.Contains("installing required build tools")) SetProgress(18, "Checking build tools");
        else if (lower.Contains("installing standalone windows sdk")) SetProgress(24, "Installing Windows SDK");
        else if (lower.Contains("visual studio c++ build environment is ready")) SetProgress(30, "Build environment ready");
        else if (lower.Contains("building ali installer from source")) SetProgress(35, "Building installer");
        else if (lower.Contains("running build step clean")) SetProgress(40, "Cleaning build output");
        else if (lower.Contains("running build step cpu")) SetProgress(48, "Building CPU runtime");
        else if (lower.Contains("building ollama cli")) SetProgress(58, "Building command line");
        else if (lower.Contains("running build step app") || lower.Contains("building react application")) SetProgress(66, "Building desktop app");
        else if (lower.Contains("running go generate")) SetProgress(72, "Generating assets");
        else if (lower.Contains("download msvc redistributables") || lower.Contains("running build step deps")) SetProgress(76, "Bundling runtime");
        else if (lower.Contains("building ollama installer") || lower.Contains("running build step installer")) SetProgress(82, "Packaging installer");
        else if (lower.Contains("validating installer artifact") || lower.StartsWith("installer:")) SetProgress(86, "Validating package");
        else if (lower.Contains("==> installing ali")) SetProgress(89, "Installing application");
        else if (lower.Contains("starting local ali server")) SetProgress(93, "Starting local server");
        else if (lower.Contains("verifying installed release")) SetProgress(96, "Verifying release");
        else if (lower.Contains("opening ali ide")) SetProgress(98, "Opening IDE");
        else if (lower.Contains("ali web setup completed successfully")) SetProgress(100, "Complete");
        else if (lower.Contains("error:") || lower.Contains("failed with exit code") || lower.Contains("ali full install failed")) MarkProgressFailed();
    }

    private void AppendLog(string text)
    {
        if (InvokeRequired)
        {
            BeginInvoke(new Action<string>(AppendLog), text);
            return;
        }
        try
        {
            File.AppendAllText(installLog, text + Environment.NewLine, Encoding.UTF8);
        }
        catch
        {
        }
        UpdateProgressFromLog(text);
        logBox.AppendText(text + Environment.NewLine);
    }

    private void EnableClose()
    {
        if (InvokeRequired)
        {
            BeginInvoke(new Action(EnableClose));
            return;
        }
        closeButton.Enabled = true;
        closeButton.BackColor = Color.FromArgb(87, 153, 255);
    }
}

internal sealed class GlassPanel : Panel
{
    public int Radius { get; set; }
    public Color FillColor { get; set; }
    public Color StrokeColor { get; set; }

    public GlassPanel()
    {
        Radius = 18;
        FillColor = Color.FromArgb(42, 255, 255, 255);
        StrokeColor = Color.FromArgb(70, 255, 255, 255);
        DoubleBuffered = true;
        SetStyle(ControlStyles.AllPaintingInWmPaint | ControlStyles.OptimizedDoubleBuffer | ControlStyles.ResizeRedraw | ControlStyles.UserPaint, true);
    }

    protected override void OnPaint(PaintEventArgs e)
    {
        e.Graphics.SmoothingMode = SmoothingMode.AntiAlias;
        Rectangle rect = new Rectangle(0, 0, Width - 1, Height - 1);
        using (GraphicsPath path = RoundedRect(rect, Radius))
        using (LinearGradientBrush fill = new LinearGradientBrush(rect, FillColor, Color.FromArgb(Math.Max(18, FillColor.A - 16), FillColor), LinearGradientMode.Vertical))
        using (Pen stroke = new Pen(StrokeColor, 1F))
        {
            e.Graphics.FillPath(fill, path);
            e.Graphics.DrawPath(stroke, path);
        }
        base.OnPaint(e);
    }

    private static GraphicsPath RoundedRect(Rectangle bounds, int radius)
    {
        int diameter = radius * 2;
        GraphicsPath path = new GraphicsPath();
        path.AddArc(bounds.Left, bounds.Top, diameter, diameter, 180, 90);
        path.AddArc(bounds.Right - diameter, bounds.Top, diameter, diameter, 270, 90);
        path.AddArc(bounds.Right - diameter, bounds.Bottom - diameter, diameter, diameter, 0, 90);
        path.AddArc(bounds.Left, bounds.Bottom - diameter, diameter, diameter, 90, 90);
        path.CloseFigure();
        return path;
    }
}

internal sealed class ProgressStrip : Control
{
    private readonly System.Windows.Forms.Timer timer;
    private int phase;
    private float displayedValue;
    private float targetValue;
    private bool failed;

    public Color Accent { get; set; }
    public Color Accent2 { get; set; }

    public ProgressStrip()
    {
        Accent = Color.FromArgb(87, 153, 255);
        Accent2 = Color.FromArgb(122, 92, 255);
        DoubleBuffered = true;
        SetStyle(ControlStyles.AllPaintingInWmPaint | ControlStyles.OptimizedDoubleBuffer | ControlStyles.ResizeRedraw | ControlStyles.UserPaint, true);

        timer = new System.Windows.Forms.Timer();
        timer.Interval = 32;
        timer.Tick += delegate
        {
            phase = (phase + 7) % 240;
            displayedValue += (targetValue - displayedValue) * 0.18F;
            if (Math.Abs(targetValue - displayedValue) < 0.2F)
            {
                displayedValue = targetValue;
            }
            Invalidate();
        };
        timer.Start();
    }

    public void SetProgress(int value)
    {
        failed = false;
        targetValue = Math.Max(0, Math.Min(100, value));
        Invalidate();
    }

    public void MarkFailed()
    {
        failed = true;
        Invalidate();
    }

    protected override void OnPaint(PaintEventArgs e)
    {
        e.Graphics.SmoothingMode = SmoothingMode.AntiAlias;
        Rectangle rect = new Rectangle(0, 0, Width - 1, Height - 1);
        using (GraphicsPath trackPath = RoundedRect(rect, Height / 2))
        using (SolidBrush track = new SolidBrush(Color.FromArgb(44, 255, 255, 255)))
        {
            e.Graphics.FillPath(track, trackPath);
        }

        int fillWidth = Math.Max(Height, (int)Math.Round((Width - 1) * (displayedValue / 100F)));
        Rectangle fillRect = new Rectangle(0, 0, fillWidth, Height - 1);
        Color left = failed ? Color.FromArgb(255, 84, 104) : Accent2;
        Color right = failed ? Color.FromArgb(255, 122, 102) : Accent;

        using (GraphicsPath fillPath = RoundedRect(fillRect, Height / 2))
        using (LinearGradientBrush fill = new LinearGradientBrush(fillRect, left, right, LinearGradientMode.Horizontal))
        {
            e.Graphics.FillPath(fill, fillPath);
        }

        if (!failed && fillWidth > 32 && displayedValue < 100F)
        {
            int shineWidth = Math.Min(110, Math.Max(42, fillWidth / 3));
            int x = -shineWidth + (phase * (fillWidth + shineWidth) / 240);
            Rectangle shineRect = new Rectangle(x, 0, shineWidth, Height - 1);
            using (GraphicsPath clip = RoundedRect(fillRect, Height / 2))
            using (LinearGradientBrush shine = new LinearGradientBrush(shineRect, Color.FromArgb(0, 255, 255, 255), Color.FromArgb(120, 255, 255, 255), LinearGradientMode.Horizontal))
            {
                ColorBlend blend = new ColorBlend();
                blend.Positions = new float[] { 0F, 0.5F, 1F };
                blend.Colors = new Color[] { Color.FromArgb(0, 255, 255, 255), Color.FromArgb(130, 255, 255, 255), Color.FromArgb(0, 255, 255, 255) };
                shine.InterpolationColors = blend;
                GraphicsState state = e.Graphics.Save();
                e.Graphics.SetClip(clip);
                e.Graphics.FillRectangle(shine, shineRect);
                e.Graphics.Restore(state);
            }
        }
    }

    protected override void Dispose(bool disposing)
    {
        if (disposing)
        {
            timer.Dispose();
        }
        base.Dispose(disposing);
    }

    private static GraphicsPath RoundedRect(Rectangle bounds, int radius)
    {
        int diameter = Math.Max(2, radius * 2);
        GraphicsPath path = new GraphicsPath();
        path.AddArc(bounds.Left, bounds.Top, diameter, diameter, 180, 90);
        path.AddArc(bounds.Right - diameter, bounds.Top, diameter, diameter, 270, 90);
        path.AddArc(bounds.Right - diameter, bounds.Bottom - diameter, diameter, diameter, 0, 90);
        path.AddArc(bounds.Left, bounds.Bottom - diameter, diameter, diameter, 90, 90);
        path.CloseFigure();
        return path;
    }
}

internal sealed class ProgressRing : Control
{
    private readonly System.Windows.Forms.Timer timer;
    private float displayedValue;
    private float targetValue;
    private int angle;
    private bool failed;

    public Color Accent { get; set; }
    public Color Accent2 { get; set; }

    public ProgressRing()
    {
        Accent = Color.FromArgb(87, 153, 255);
        Accent2 = Color.FromArgb(122, 92, 255);
        DoubleBuffered = true;
        SetStyle(ControlStyles.AllPaintingInWmPaint | ControlStyles.OptimizedDoubleBuffer | ControlStyles.ResizeRedraw | ControlStyles.UserPaint, true);

        timer = new System.Windows.Forms.Timer();
        timer.Interval = 32;
        timer.Tick += delegate
        {
            angle = (angle + 5) % 360;
            displayedValue += (targetValue - displayedValue) * 0.16F;
            if (Math.Abs(targetValue - displayedValue) < 0.2F)
            {
                displayedValue = targetValue;
            }
            Invalidate();
        };
        timer.Start();
    }

    public void SetProgress(int value)
    {
        failed = false;
        targetValue = Math.Max(0, Math.Min(100, value));
        Invalidate();
    }

    public void MarkFailed()
    {
        failed = true;
        Invalidate();
    }

    protected override void OnPaint(PaintEventArgs e)
    {
        e.Graphics.SmoothingMode = SmoothingMode.AntiAlias;
        Rectangle rect = new Rectangle(7, 7, Width - 15, Height - 15);
        Color main = failed ? Color.FromArgb(255, 98, 112) : Accent;
        Color secondary = failed ? Color.FromArgb(255, 145, 112) : Accent2;

        using (Pen track = new Pen(Color.FromArgb(42, 255, 255, 255), 7F))
        {
            track.StartCap = LineCap.Round;
            track.EndCap = LineCap.Round;
            e.Graphics.DrawArc(track, rect, 0, 360);
        }

        float sweep = Math.Max(2F, 360F * displayedValue / 100F);
        using (LinearGradientBrush brush = new LinearGradientBrush(rect, secondary, main, LinearGradientMode.ForwardDiagonal))
        using (Pen arc = new Pen(brush, 7F))
        {
            arc.StartCap = LineCap.Round;
            arc.EndCap = LineCap.Round;
            e.Graphics.DrawArc(arc, rect, -90, sweep);
        }

        if (!failed && displayedValue < 100F)
        {
            double radians = (angle - 90) * Math.PI / 180D;
            int radius = rect.Width / 2;
            int cx = rect.Left + rect.Width / 2;
            int cy = rect.Top + rect.Height / 2;
            int dotX = cx + (int)(Math.Cos(radians) * radius) - 4;
            int dotY = cy + (int)(Math.Sin(radians) * radius) - 4;
            using (SolidBrush dot = new SolidBrush(Color.FromArgb(235, 255, 255, 255)))
            {
                e.Graphics.FillEllipse(dot, dotX, dotY, 8, 8);
            }
        }

        string text = failed ? "!" : ((int)Math.Round(displayedValue)).ToString() + "%";
        using (Font font = new Font("Segoe UI Semibold", failed ? 24F : 16F, FontStyle.Bold))
        using (SolidBrush brush = new SolidBrush(Color.FromArgb(246, 248, 255)))
        using (StringFormat format = new StringFormat())
        {
            format.Alignment = StringAlignment.Center;
            format.LineAlignment = StringAlignment.Center;
            e.Graphics.DrawString(text, font, brush, ClientRectangle, format);
        }
    }

    protected override void Dispose(bool disposing)
    {
        if (disposing)
        {
            timer.Dispose();
        }
        base.Dispose(disposing);
    }
}

internal static class NativeChrome
{
    [DllImport("dwmapi.dll")]
    private static extern int DwmSetWindowAttribute(IntPtr hwnd, int attr, ref int attrValue, int attrSize);

    public static void TryUseImmersiveDarkMode(IntPtr handle)
    {
        try
        {
            int enabled = 1;
            if (DwmSetWindowAttribute(handle, 20, ref enabled, sizeof(int)) != 0)
            {
                DwmSetWindowAttribute(handle, 19, ref enabled, sizeof(int));
            }
        }
        catch
        {
        }
    }
}
'@

    $content = $content.Replace("@@VERSION@@", $Version)
    Set-Content -LiteralPath $Path -Value $content -Encoding ASCII
}

function Find-CSharpCompiler {
    $candidates = @(
        (Join-Path $env:WINDIR "Microsoft.NET\Framework64\v4.0.30319\csc.exe"),
        (Join-Path $env:WINDIR "Microsoft.NET\Framework\v4.0.30319\csc.exe")
    )

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }

    $command = Get-Command csc.exe -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    return ""
}

Write-Section "Preparing staging"
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null
Assert-UnderPath -Path $StagingRoot -Parent $DistDir
if (Test-Path -LiteralPath $StagingRoot) {
    Remove-Item -LiteralPath $StagingRoot -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $SourceStage | Out-Null
New-Item -ItemType Directory -Force -Path $PayloadDir | Out-Null

Write-Section "Copying source tree"
$robocopyArgs = @(
    $RepoRoot,
    $SourceStage,
    "/E",
    "/NFL",
    "/NDL",
    "/NJH",
    "/NJS",
    "/NP",
    "/XD",
    ".git",
    "dist",
    "build",
    "node_modules",
    ".cache",
    "__pycache__",
    "/XF",
    "*.log",
    "*.tmp"
)
& robocopy @robocopyArgs | Out-Null
$robocopyExitCode = $LASTEXITCODE
if ($robocopyExitCode -ge 8) {
    throw "robocopy failed with exit code $robocopyExitCode"
}
$global:LASTEXITCODE = 0

Write-Section "Compressing embedded source"
if (Test-Path -LiteralPath $SourceZip) {
    Remove-Item -LiteralPath $SourceZip -Force
}
$sourceItems = Get-ChildItem -LiteralPath $SourceStage -Force
Compress-Archive -Path $sourceItems.FullName -DestinationPath $SourceZip -CompressionLevel Optimal -Force

Write-Section "Creating bootstrap payload"
Set-Content -LiteralPath $BootstrapCmd -Encoding ASCII -Value @'
@echo off
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0bootstrap.ps1"
exit /b %ERRORLEVEL%
'@
New-BootstrapScript -Path $BootstrapPs1
New-BootstrapExecutableSource -Path $BootstrapCs

Write-Section "Compiling AliWebSetup.exe"
$compiler = Find-CSharpCompiler
if (-not $compiler) {
    throw ".NET Framework C# compiler was not found. Install .NET Framework Developer Pack or Windows SDK."
}

if (Test-Path -LiteralPath $OutputPath) {
    Remove-Item -LiteralPath $OutputPath -Force
}

$compileArgs = @(
    "/nologo",
    "/target:winexe",
    "/platform:anycpu",
    "/optimize+",
    "/reference:System.Windows.Forms.dll",
    "/reference:System.Drawing.dll",
    "/out:$OutputPath",
    "/resource:$BootstrapPs1,bootstrap.ps1",
    "/resource:$SourceZip,source.zip",
    $BootstrapCs
)
$iconPath = Join-Path $RepoRoot "app\assets\app.ico"
if (Test-Path -LiteralPath $iconPath) {
    $compileArgs += "/win32icon:$iconPath"
    $compileArgs += "/resource:$iconPath,app.ico"
}
& $compiler @compileArgs
if ($LASTEXITCODE -ne 0) {
    throw "C# compiler failed with exit code $LASTEXITCODE"
}

if (-not (Test-Path -LiteralPath $OutputPath)) {
    throw "Installer was not produced: $OutputPath"
}

$item = Get-Item -LiteralPath $OutputPath
$hash = Get-FileHash -LiteralPath $OutputPath -Algorithm SHA256
$manifest = [ordered]@{
    name = "AliWebSetup"
    version = $Version
    created_at = (Get-Date).ToString("o")
    output = $item.FullName
    size = $item.Length
    sha256 = $hash.Hash
    embedded = @("bootstrap.ps1", "source.zip", "app.ico")
    default_model = if ($PreloadModel -and -not $SkipModel) { $Model } else { "" }
    preload_model = [bool]($PreloadModel -and -not $SkipModel -and $Model)
    cpu_only = -not $GpuBuild
    installer_behavior = @(
        "show modern dark Ali Setup progress window",
        "show percentage progress by installation stage",
        "show animated circular progress indicator",
        "show smooth determinate progress line",
        "self-elevate through Windows UAC when needed",
        "extract embedded source",
        "install missing build tools with winget",
        "build AliSetup.exe",
        "install Ali silently",
        "start local server",
        "do not download models by default",
        "download model only when built with -PreloadModel",
        "run post-install release verification",
        "open IDE unless disabled"
    )
}
$manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $ReleaseManifest -Encoding UTF8

Write-Host ""
Write-Host "Installer: $($item.FullName)" -ForegroundColor Green
Write-Host "Size:      $([Math]::Round($item.Length / 1MB, 2)) MB"
Write-Host "SHA256:    $($hash.Hash)"
Write-Host "Manifest:  $ReleaseManifest"
Write-Host "Log inside installed run: %TEMP%\AliWebSetup-*\install.log"

if (-not $KeepStaging) {
    Write-Section "Cleaning staging"
    Assert-UnderPath -Path $StagingRoot -Parent $DistDir
    Remove-Item -LiteralPath $StagingRoot -Recurse -Force
}
