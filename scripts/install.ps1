# Install Vecna on Windows.
# Usage:
#   irm https://raw.githubusercontent.com/zhravan/vecna/main/scripts/install.ps1 | iex
# Optional environment variables: $env:VERSION and $env:BIN_DIR

$ErrorActionPreference = "Stop"
$Repo = "zhravan/vecna"
$Version = if ($env:VERSION) { $env:VERSION.TrimStart("v") } else { "latest" }
$BinDir = if ($env:BIN_DIR) { $env:BIN_DIR } else { Join-Path $env:LOCALAPPDATA "vecna\bin" }

$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    default { throw "Unsupported Windows architecture: $env:PROCESSOR_ARCHITECTURE" }
}

if ($Version -eq "latest") {
    $releases = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "Accept" = "application/vnd.github+json" }
    $tag = $releases.tag_name -replace "^v", ""
} else { $tag = $Version }

$zipName = "vecna_${tag}_windows_${arch}.zip"
$baseUrl = "https://github.com/$Repo/releases/download/v$tag"
$url = "$baseUrl/$zipName"
$checksumUrl = "$baseUrl/SHA256SUMS.txt"

if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir -Force | Out-Null }
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("vecna-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    $zipPath = Join-Path $tempDir $zipName
    $checksumPath = Join-Path $tempDir "SHA256SUMS.txt"
    Write-Host "Downloading Vecna v$tag (windows/$arch)..."
    Invoke-WebRequest -Uri $url -OutFile $zipPath
    Invoke-WebRequest -Uri $checksumUrl -OutFile $checksumPath
    $expected = Get-Content $checksumPath | Where-Object { $_ -match ("\s" + [regex]::Escape($zipName) + "$") } | Select-Object -First 1 | ForEach-Object { ($_ -split "\s+")[0] }
    if (-not $expected) { throw "No checksum found for $zipName." }
    $actual = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($expected.ToLowerInvariant() -ne $actual) { throw "Checksum verification failed for $zipName. Expected $expected, got $actual." }
    Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force
    $exePath = Join-Path $tempDir "vecna.exe"
    if (-not (Test-Path $exePath)) { throw "Install failed: vecna.exe was not found in the release archive." }
    Copy-Item -Path $exePath -Destination (Join-Path $BinDir "vecna.exe") -Force
    $installedPath = Join-Path $BinDir "vecna.exe"
    if (-not (Test-Path $installedPath)) { throw "Install failed: vecna.exe was not installed to $BinDir." }
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = if ($userPath) { $userPath -split ";" } else { @() }
    if ($pathEntries -notcontains $BinDir) {
        $newUserPath = (($pathEntries | Where-Object { $_ }) + $BinDir) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        $env:Path = "$env:Path;$BinDir"
        Write-Host "Added $BinDir to the user PATH."
    }
    Write-Host "Installed Vecna v$tag to $installedPath"
    Write-Host "Run: vecna version"
} finally {
    if (Test-Path $tempDir) { Remove-Item $tempDir -Recurse -Force }
}
