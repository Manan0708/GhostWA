# WACLI One-Line Automated Installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/Manan0708/Whatsapp-CLI/main/install.ps1 | iex

$ErrorActionPreference = "SilentlyContinue"

Write-Host ""
Write-Host "  ┌──────────────────────────────────────────────┐" -ForegroundColor Cyan
Write-Host "  │   ⚡  WACLI - WhatsApp CLI Auto-Installer    │" -ForegroundColor Cyan
Write-Host "  └──────────────────────────────────────────────┘" -ForegroundColor Cyan
Write-Host ""

$installDir = "$env:LOCALAPPDATA\Programs\wacli"
$binPath = "$installDir\wacli.exe"

# 1. Create Installation Directory
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# 2. Check if Go is installed on the machine
$hasGo = Get-Command "go" -ErrorAction SilentlyContinue

if ($hasGo) {
    Write-Host "[1/3] Go detected! Compiling and installing latest WACLI binary..." -ForegroundColor Yellow
    & go install github.com/Manan0708/wacli/cmd/wacli@latest
    
    # Check if GOPATH/bin binary exists
    $gopathBin = "$env:USERPROFILE\go\bin\wacli.exe"
    if (Test-Path $gopathBin) {
        Copy-Item $gopathBin $binPath -Force
    }
} else {
    Write-Host "[1/3] Go not found on system. Downloading pre-compiled WACLI executable..." -ForegroundColor Yellow
    
    # Download executable from repository
    $downloadUrl = "https://raw.githubusercontent.com/Manan0708/Whatsapp-CLI/main/bin/wacli.exe"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $binPath -UseBasicParsing
}

# Fallback: Check if local wacli.exe exists if script is run locally
if (!(Test-Path $binPath)) {
    $localBin = Join-Path $PSScriptRoot "bin\wacli.exe"
    if (Test-Path $localBin) {
        Copy-Item $localBin $binPath -Force
    }
}

if (!(Test-Path $binPath)) {
    Write-Host "[-] Installation failed. Could not locate or download wacli.exe executable." -ForegroundColor Red
    exit 1
}

# 3. Add install directory to User PATH environment variable
Write-Host "[2/3] Registering WACLI in User Environment PATH..." -ForegroundColor Yellow
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    $newPath = "$userPath;$installDir"
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    $env:PATH = "$env:PATH;$installDir"
}

# 4. Ensure Data Directory Exists
$dataDir = "$env:USERPROFILE\.local\share\wacli"
if (!(Test-Path $dataDir)) {
    New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
}

Write-Host "[3/3] Finalizing setup..." -ForegroundColor Yellow
Write-Host ""
Write-Host "  ✅ WACLI Installed Successfully!" -ForegroundColor Green
Write-Host "  ------------------------------------------------" -ForegroundColor Gray
Write-Host "  To start using WACLI immediately:" -ForegroundColor White
Write-Host "    1. Type: " -NoNewline; Write-Host "wacli login" -ForegroundColor Cyan -NoNewline; Write-Host " (to scan QR code)"
Write-Host "    2. Type: " -NoNewline; Write-Host "wacli daemon start" -ForegroundColor Cyan -NoNewline; Write-Host " (to launch background service)"
Write-Host "    3. Type: " -NoNewline; Write-Host "wacli show" -ForegroundColor Cyan -NoNewline; Write-Host " (to open interactive TUI dashboard)"
Write-Host "  ------------------------------------------------" -ForegroundColor Gray
Write-Host ""
