# Install vecna on Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/shravan20/vecna/master/scripts/install.ps1 | iex
# Or: .\install.ps1 [-Version 1.0.0] [-BinDir C:\path\to\bin]

$ErrorActionPreference = "Stop"
$Repo = "shravan20/vecna"
$Version = if ($env:VERSION) { $env:VERSION } else { "latest" }
$BinDir = if ($env:BIN_DIR) { $env:BIN_DIR } else { Join-Path $env:LOCALAPPDATA "vecna\bin" }

# Map Windows architecture to Go arch
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64"   { "amd64" }
    "ARM64"   { "arm64" }
    default   { "amd64" }
}

if ($Version -eq "latest") {
    $releases = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{
        "Accept" = "application/vnd.github.v3+json"
    }
    $tag = $releases.tag_name -replace "^v", ""
} else {
    $tag = $Version
}

$zipName = "vecna_${tag}_windows_${arch}.zip"
$url = "https://github.com/$Repo/releases/download/v$tag/$zipName"

if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
}

$zipPath = Join-Path $env:TEMP $zipName
Write-Host "Downloading vecna $tag (windows/$arch)..."
Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing

Expand-Archive -Path $zipPath -DestinationPath $BinDir -Force
Remove-Item $zipPath -Force

$exePath = Join-Path $BinDir "vecna.exe"
if (-not (Test-Path $exePath)) {
    Write-Error "Install failed: vecna.exe not found in $BinDir"
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$BinDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$BinDir", "User")
    $env:Path = "$env:Path;$BinDir"
    Write-Host "Added $BinDir to user PATH."
}

Write-Host "Installed vecna $tag to $BinDir"
Write-Host "Run: vecna"
