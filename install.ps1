# WACLI / GhostWA Default Automated One-Line Installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/Manan0708/GhostWA/main/install.ps1 | iex

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

Write-Host ""
Write-Host "  ┌────────────────────────────────────────────────────────┐" -ForegroundColor Cyan
Write-Host "  │   🔒 WACLI v2.0 (Stable Release) - Auto-Installer      │" -ForegroundColor Cyan
Write-Host "  └────────────────────────────────────────────────────────┘" -ForegroundColor Cyan
Write-Host ""

$installDir = "$env:LOCALAPPDATA\Programs\wacli"
$wacliPath = "$installDir\wacli.exe"

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

Write-Host "[1/3] Downloading stable WACLI v2.0 pre-compiled executable..." -ForegroundColor Yellow
$urlWACLI = "https://raw.githubusercontent.com/Manan0708/GhostWA/main/bin/wacli-v2.0.exe"
$dl = Download-Binary -url $urlWACLI -outFile $wacliPath

# Fallback: Check local repo bin directory if script is run locally
if (!(Test-Path $wacliPath)) {
    $localWACLI = Join-Path $PSScriptRoot "bin\wacli-v2.0.exe"
    if (Test-Path $localWACLI) {
        Copy-Item $localWACLI $wacliPath -Force
    }
}

if (!(Test-Path $wacliPath)) {
    Write-Host "[-] Installation failed. Could not locate or download wacli-v2.0.exe executable." -ForegroundColor Red
    exit 1
}

# 2. Add install directory to User PATH environment variable
Write-Host "[2/3] Registering 'wacli' in User Environment PATH..." -ForegroundColor Yellow
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
Write-Host "  ✅ WACLI v2.0 (Stable) Installed Successfully!" -ForegroundColor Green
Write-Host "  ------------------------------------------------------------" -ForegroundColor Gray
Write-Host "  To start using WACLI v2.0:" -ForegroundColor White
Write-Host "    1. Type: " -NoNewline; Write-Host "wacli login" -ForegroundColor Cyan -NoNewline; Write-Host " (to scan QR code)"
Write-Host "    2. Type: " -NoNewline; Write-Host "wacli daemon start" -ForegroundColor Cyan -NoNewline; Write-Host " (to launch background service)"
Write-Host "    3. Type: " -NoNewline; Write-Host "wacli show" -ForegroundColor Cyan -NoNewline; Write-Host " (to open interactive TUI dashboard)"
Write-Host "  ------------------------------------------------------------" -ForegroundColor Gray
Write-Host ""
