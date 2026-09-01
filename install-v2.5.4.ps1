# GhostWA Dev v2.5.4 Automated One-Line Installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/Manan0708/GhostWA/main/install-v2.5.4.ps1 | iex

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

Write-Host ""
Write-Host "  ┌────────────────────────────────────────────────────────┐" -ForegroundColor Cyan
Write-Host "  │   ⚡ GhostWA v2.5.4 (Dev Release) - Auto-Installer    │" -ForegroundColor Cyan
Write-Host "  └────────────────────────────────────────────────────────┘" -ForegroundColor Cyan
Write-Host ""

$installDir = "$env:LOCALAPPDATA\Programs\ghostwa"
$ghostwaPath = "$installDir\ghostwa.exe"

# 1. Ensure Installation Directory Exists
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Helper function to download binary cleanly
function Download-Binary {
    param ([string]$url, [string]$outFile)
    
    $hasCurl = Get-Command "curl.exe" -ErrorAction SilentlyContinue
    if ($hasCurl) {
        & curl.exe -sSL -f $url -o $outFile
        if ((Test-Path $outFile) -and ((Get-Item $outFile).Length -gt 1000000)) {
            return $true
        }
    }
    
    try {
        Invoke-WebRequest -Uri $url -OutFile $outFile -UseBasicParsing -ErrorAction Stop
        if ((Test-Path $outFile) -and ((Get-Item $outFile).Length -gt 1000000)) {
            return $true
        }
    } catch { }
    
    return $false
}

Write-Host "[1/3] Downloading GhostWA v2.5.4 pre-compiled executable..." -ForegroundColor Yellow
$urlGhostWA = "https://raw.githubusercontent.com/Manan0708/GhostWA/main/bin/ghostwa-v2.5.4.exe"
$dl = Download-Binary -url $urlGhostWA -outFile $ghostwaPath

# Fallback: Check local repo bin directory if script is run locally
if (!(Test-Path $ghostwaPath)) {
    $localGhostWA = Join-Path $PSScriptRoot "bin\ghostwa-v2.5.4.exe"
    if (Test-Path $localGhostWA) {
        Copy-Item $localGhostWA $ghostwaPath -Force
    }
}

if (!(Test-Path $ghostwaPath)) {
    Write-Host "[-] Installation failed. Could not locate or download ghostwa-v2.5.4.exe executable." -ForegroundColor Red
    exit 1
}

# 2. Add install directory to User PATH environment variable
Write-Host "[2/3] Registering 'ghostwa' in User Environment PATH..." -ForegroundColor Yellow
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    $newPath = "$userPath;$installDir"
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    $env:PATH = "$env:PATH;$installDir"
}

# 3. Ensure Data Directory Exists
$dataDir = "$env:USERPROFILE\.local\share\wacli"
if (!(Test-Path $dataDir)) {
    New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
}

Write-Host "[3/3] Finalizing setup..." -ForegroundColor Yellow
Write-Host ""
Write-Host "  ✅ GhostWA v2.5.4 (Dev) Installed Successfully!" -ForegroundColor Green
Write-Host "  ------------------------------------------------------------" -ForegroundColor Gray
Write-Host "  To start using GhostWA v2.5.4:" -ForegroundColor White
Write-Host "    1. Type: " -NoNewline; Write-Host "ghostwa login" -ForegroundColor Cyan -NoNewline; Write-Host " (to link device via QR or Phone pairing)"
Write-Host "    2. Type: " -NoNewline; Write-Host "ghostwa daemon start" -ForegroundColor Cyan -NoNewline; Write-Host " (to launch background service)"
Write-Host "    3. Type: " -NoNewline; Write-Host "ghostwa show" -ForegroundColor Cyan -NoNewline; Write-Host " (to open interactive TUI dashboard)"
Write-Host "  ------------------------------------------------------------" -ForegroundColor Gray
Write-Host ""
