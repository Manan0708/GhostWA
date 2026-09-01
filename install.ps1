# GhostWA Automated One-Line Installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/Manan0708/GhostWA/main/install.ps1 | iex

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

Write-Host ""
Write-Host "  ┌────────────────────────────────────────────────────────┐" -ForegroundColor Cyan
Write-Host "  │   ⚡ GhostWA v2.5 - Single Command Auto-Installer     │" -ForegroundColor Cyan
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
    
    # Try curl.exe first (native in Windows 10/11)
    $hasCurl = Get-Command "curl.exe" -ErrorAction SilentlyContinue
    if ($hasCurl) {
        & curl.exe -sSL -f $url -o $outFile
        if ((Test-Path $outFile) -and ((Get-Item $outFile).Length -gt 1000000)) {
            return $true
        }
    }
    
    # Fallback to Invoke-WebRequest
    try {
        Invoke-WebRequest -Uri $url -OutFile $outFile -UseBasicParsing -ErrorAction Stop
        if ((Test-Path $outFile) -and ((Get-Item $outFile).Length -gt 1000000)) {
            return $true
        }
    } catch { }
    
    return $false
}

# 2. Check if Go is installed on system
$hasGo = Get-Command "go" -ErrorAction SilentlyContinue

if ($hasGo) {
    Write-Host "[1/3] Go detected! Compiling and installing latest GhostWA binary..." -ForegroundColor Yellow
    & go install github.com/Manan0708/GhostWA/cmd/ghostwa@latest
    
    $gopathBin = "$env:USERPROFILE\go\bin\ghostwa.exe"
    if (Test-Path $gopathBin) {
        Copy-Item $gopathBin $ghostwaPath -Force
    }
} else {
    Write-Host "[1/3] Downloading pre-compiled GhostWA v2.5 executable..." -ForegroundColor Yellow
    
    $urlGhostWA = "https://raw.githubusercontent.com/Manan0708/GhostWA/main/bin/ghostwa-v2.5.exe"
    $dl = Download-Binary -url $urlGhostWA -outFile $ghostwaPath
    
    # If binary download fails, attempt to install Go automatically via winget
    if (-not $dl) {
        Write-Host "[-] Installing Go compiler environment automatically..." -ForegroundColor Yellow
        $hasWinget = Get-Command "winget" -ErrorAction SilentlyContinue
        if ($hasWinget) {
            & winget install --id GoLang.Go -e --source winget --accept-source-agreements --accept-package-agreements --silent
            $env:PATH = "$env:PATH;C:\Program Files\Go\bin"
            $hasGoNow = Get-Command "go" -ErrorAction SilentlyContinue
            if ($hasGoNow) {
                & go install github.com/Manan0708/GhostWA/cmd/ghostwa@latest
                $gopathBin = "$env:USERPROFILE\go\bin\ghostwa.exe"
                if (Test-Path $gopathBin) {
                    Copy-Item $gopathBin $ghostwaPath -Force
                }
            }
        }
    }
}

# Fallback: Check local repo bin directory if script is run locally
if (!(Test-Path $ghostwaPath)) {
    $localGhostWA = Join-Path $PSScriptRoot "bin\ghostwa-v2.5.exe"
    if (Test-Path $localGhostWA) {
        Copy-Item $localGhostWA $ghostwaPath -Force
    }
}

if (!(Test-Path $ghostwaPath)) {
    Write-Host "[-] Installation failed. Could not locate or download ghostwa.exe executable." -ForegroundColor Red
    exit 1
}

# 3. Add install directory to User PATH environment variable
Write-Host "[2/3] Registering 'ghostwa' in User Environment PATH..." -ForegroundColor Yellow
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
Write-Host "  ✅ GhostWA v2.5 Installed Successfully!" -ForegroundColor Green
Write-Host "  ------------------------------------------------------------" -ForegroundColor Gray
Write-Host "  To start using GhostWA immediately:" -ForegroundColor White
Write-Host "    1. Type: " -NoNewline; Write-Host "ghostwa login" -ForegroundColor Cyan -NoNewline; Write-Host " (to link device via QR or Phone pairing)"
Write-Host "    2. Type: " -NoNewline; Write-Host "ghostwa daemon start" -ForegroundColor Cyan -NoNewline; Write-Host " (to launch background service)"
Write-Host "    3. Type: " -NoNewline; Write-Host "ghostwa show" -ForegroundColor Cyan -NoNewline; Write-Host " (to open interactive TUI dashboard)"
Write-Host "  ------------------------------------------------------------" -ForegroundColor Gray
Write-Host ""
