# Mio installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/Yarov/mio/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "Yarov/mio"
$Binary = "mio.exe"
$InstallDir = if ($env:MIO_INSTALL_DIR) { $env:MIO_INSTALL_DIR } else { "$env:LOCALAPPDATA\mio\bin" }

function Write-Info($msg)  { Write-Host "[mio] " -ForegroundColor Cyan -NoNewline; Write-Host $msg }
function Write-Ok($msg)    { Write-Host "[mio] " -ForegroundColor Green -NoNewline; Write-Host $msg }
function Write-Err($msg)   { Write-Host "[mio] " -ForegroundColor Red -NoNewline; Write-Host $msg; exit 1 }

Write-Info "Installing Mio — persistent memory for AI agents"
Write-Host ""

# Detect architecture
$Arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else { Write-Err "32-bit Windows is not supported" }
Write-Info "Detected: windows/$Arch"

# Get latest version
Write-Info "Fetching latest version..."
try {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name -replace '^v', ''
} catch {
    Write-Err "Could not fetch latest version. Check https://github.com/$Repo/releases"
}
Write-Info "Latest version: v$Version"

# Download
$Filename = "mio_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/v$Version/$Filename"
$TmpDir = Join-Path $env:TEMP "mio-install-$(Get-Random)"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null
$ZipPath = Join-Path $TmpDir $Filename

Write-Info "Downloading $Filename..."
try {
    Invoke-WebRequest -Uri $Url -OutFile $ZipPath -UseBasicParsing
} catch {
    Write-Err "Download failed: $_"
}

# Extract
Write-Info "Extracting..."
Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

# Install
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item -Path (Join-Path $TmpDir $Binary) -Destination (Join-Path $InstallDir $Binary) -Force

# Add to PATH if not already there
$UserPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Info "Adding $InstallDir to PATH..."
    [System.Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path = "$env:Path;$InstallDir"
}

# Cleanup
Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue

# Verify
try {
    $VerOut = & (Join-Path $InstallDir $Binary) version 2>&1
    Write-Host ""
    Write-Ok "Mio v$Version installed to $InstallDir\$Binary"
    Write-Host ""
    Write-Info "Next steps:"
    Write-Host "  1. Run 'mio setup' to configure your AI agent"
    Write-Host "  2. Restart your agent (Claude Code, Cursor, etc.)"
    Write-Host "  3. Open http://localhost:7438 for the dashboard"
    Write-Host ""
} catch {
    Write-Err "Installation failed. Binary not working."
}
