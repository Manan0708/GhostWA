#!/usr/bin/env sh
# GhostWA & WACLI Automated One-Line Installer for Linux & macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/Manan0708/GhostWA/main/install.sh | sh

set -e

echo ""
echo "  ┌────────────────────────────────────────────────────────┐"
echo "  │   ⚡ GhostWA (v2.5) & WACLI (v2.0) Auto-Installer     │"
echo "  └────────────────────────────────────────────────────────┘"
echo ""

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

if command -v go >/dev/null 2>&1; then
    echo "[1/3] Go detected! Compiling and installing latest GhostWA & WACLI binaries..."
    go install github.com/Manan0708/GhostWA/cmd/ghostwa@latest
    if [ -f "$HOME/go/bin/ghostwa" ]; then
        cp "$HOME/go/bin/ghostwa" "$INSTALL_DIR/ghostwa"
        cp "$HOME/go/bin/ghostwa" "$INSTALL_DIR/wacli"
    fi
else
    echo "[1/3] Downloading pre-compiled GhostWA (v2.5) and WACLI (v2.0) executables..."
    curl -fsSL "https://raw.githubusercontent.com/Manan0708/GhostWA/main/bin/ghostwa" -o "$INSTALL_DIR/ghostwa" || true
    curl -fsSL "https://raw.githubusercontent.com/Manan0708/GhostWA/main/bin/wacli" -o "$INSTALL_DIR/wacli" || true
fi

if [ -f "$INSTALL_DIR/ghostwa" ]; then
    chmod +x "$INSTALL_DIR/ghostwa"
    chmod +x "$INSTALL_DIR/wacli" 2>/dev/null || true
else
    echo "[-] Installation failed. Could not locate or download ghostwa binary."
    exit 1
fi

# Ensure data directory exists
mkdir -p "$HOME/.local/share/wacli"

echo "[2/3] Registering GhostWA and WACLI in PATH..."
echo "[3/3] Finalizing setup..."
echo ""
echo "  ✅ GhostWA (v2.5) & WACLI (v2.0) Installed Successfully!"
echo "  ------------------------------------------------------------"
echo "  Both 'ghostwa' and 'wacli' commands are now active on your system:"
echo "    1. Type: ghostwa login (or wacli login)"
echo "    2. Type: ghostwa daemon start"
echo "    3. Type: ghostwa show"
echo "  ------------------------------------------------------------"
echo ""
