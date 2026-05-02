<#
.SYNOPSIS
    Regenerate Windows app icons from the project image assets.

.DESCRIPTION
    Uses images\logo.png when present, otherwise docs\images\logo.png, to
    regenerate app.ico, tray.ico, tray_upgrade.ico, and setup.bmp.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\update_app_icons.ps1
#>

[CmdletBinding()]
param(
    [string]$SourceImage = "",
    [string]$SetupImage = ""
)

$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$AssetsDir = Join-Path $RepoRoot "app\assets"

function Resolve-AssetImage {
    param(
        [string]$Explicit,
        [string[]]$Candidates
    )

    if ($Explicit) {
        $path = if ([System.IO.Path]::IsPathRooted($Explicit)) { $Explicit } else { Join-Path $RepoRoot $Explicit }
        if (Test-Path -LiteralPath $path) {
            return (Resolve-Path $path).Path
        }
        throw "Image not found: $Explicit"
    }

    foreach ($candidate in $Candidates) {
        $path = Join-Path $RepoRoot $candidate
        if (Test-Path -LiteralPath $path) {
            return (Resolve-Path $path).Path
        }
    }
    throw "No suitable source image found."
}

function New-ResizedBitmap {
    param(
        [System.Drawing.Image]$Source,
        [int]$Width,
        [int]$Height,
        [switch]$Transparent
    )

    $bitmap = [System.Drawing.Bitmap]::new($Width, $Height, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
    try {
        $graphics.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
        $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
        $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        if ($Transparent) {
            $graphics.Clear([System.Drawing.Color]::Transparent)
        } else {
            $graphics.Clear([System.Drawing.Color]::FromArgb(245, 247, 250))
        }

        $scale = [Math]::Min($Width / $Source.Width, $Height / $Source.Height)
        $targetWidth = [int][Math]::Round($Source.Width * $scale)
        $targetHeight = [int][Math]::Round($Source.Height * $scale)
        $x = [int][Math]::Round(($Width - $targetWidth) / 2)
        $y = [int][Math]::Round(($Height - $targetHeight) / 2)
        $graphics.DrawImage($Source, $x, $y, $targetWidth, $targetHeight)
    } finally {
        $graphics.Dispose()
    }
    return $bitmap
}

function Get-PngBytes {
    param([System.Drawing.Bitmap]$Bitmap)

    $stream = [System.IO.MemoryStream]::new()
    try {
        $Bitmap.Save($stream, [System.Drawing.Imaging.ImageFormat]::Png)
        return $stream.ToArray()
    } finally {
        $stream.Dispose()
    }
}

function Write-Ico {
    param(
        [string]$Path,
        [string]$SourcePath,
        [int[]]$Sizes
    )

    $source = [System.Drawing.Image]::FromFile($SourcePath)
    try {
        $frames = @()
        foreach ($size in $Sizes) {
            $bitmap = New-ResizedBitmap -Source $source -Width $size -Height $size -Transparent
            try {
                $frames += [pscustomobject]@{
                    Size = $size
                    Data = Get-PngBytes -Bitmap $bitmap
                }
            } finally {
                $bitmap.Dispose()
            }
        }

        $stream = [System.IO.File]::Create($Path)
        $writer = [System.IO.BinaryWriter]::new($stream)
        try {
            $writer.Write([uint16]0)
            $writer.Write([uint16]1)
            $writer.Write([uint16]$frames.Count)

            $offset = 6 + (16 * $frames.Count)
            foreach ($frame in $frames) {
                $dimension = if ($frame.Size -ge 256) { 0 } else { $frame.Size }
                $writer.Write([byte]$dimension)
                $writer.Write([byte]$dimension)
                $writer.Write([byte]0)
                $writer.Write([byte]0)
                $writer.Write([uint16]1)
                $writer.Write([uint16]32)
                $writer.Write([uint32]$frame.Data.Length)
                $writer.Write([uint32]$offset)
                $offset += $frame.Data.Length
            }

            foreach ($frame in $frames) {
                $writer.Write([byte[]]$frame.Data)
            }
        } finally {
            $writer.Dispose()
            $stream.Dispose()
        }
    } finally {
        $source.Dispose()
    }
}

function Write-SetupBitmap {
    param(
        [string]$Path,
        [string]$SourcePath
    )

    $source = [System.Drawing.Image]::FromFile($SourcePath)
    try {
        $bitmap = New-ResizedBitmap -Source $source -Width 138 -Height 140
        try {
            $bitmap.Save($Path, [System.Drawing.Imaging.ImageFormat]::Bmp)
        } finally {
            $bitmap.Dispose()
        }
    } finally {
        $source.Dispose()
    }
}

Add-Type -AssemblyName System.Drawing

$iconSource = Resolve-AssetImage -Explicit $SourceImage -Candidates @(
    "images\app.png",
    "images\icon.png",
    "images\logo.png",
    "docs\images\logo.png",
    "docs\images\favicon.png"
)
$setupSource = Resolve-AssetImage -Explicit $SetupImage -Candidates @(
    "images\setup.png",
    "images\logo.png",
    "docs\images\logo.png"
)

New-Item -ItemType Directory -Force -Path $AssetsDir | Out-Null

Write-Ico -Path (Join-Path $AssetsDir "app.ico") -SourcePath $iconSource -Sizes @(16, 24, 32, 48, 64, 128, 256)
Write-Ico -Path (Join-Path $AssetsDir "tray.ico") -SourcePath $iconSource -Sizes @(16, 20, 24, 32, 40, 48, 64)
Write-Ico -Path (Join-Path $AssetsDir "tray_upgrade.ico") -SourcePath $iconSource -Sizes @(16, 20, 24, 32, 40, 48, 64)
Write-SetupBitmap -Path (Join-Path $AssetsDir "setup.bmp") -SourcePath $setupSource

Write-Host "Icon source:  $iconSource"
Write-Host "Setup source: $setupSource"
Get-Item -LiteralPath `
    (Join-Path $AssetsDir "app.ico"), `
    (Join-Path $AssetsDir "tray.ico"), `
    (Join-Path $AssetsDir "tray_upgrade.ico"), `
    (Join-Path $AssetsDir "setup.bmp") |
    Select-Object FullName, Length, LastWriteTime
