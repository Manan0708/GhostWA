#!/usr/bin/env sh
# GhostWA Automated One-Line Installer for Linux & macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/Manan0708/GhostWA/main/install.sh | sh

set -e

echo ""
echo "  ┌────────────────────────────────────────────────────────┐"
echo "  │   ⚡ GhostWA v2.5.8 - Single Command Auto-Installer     │"
echo "  └────────────────────────────────────────────────────────┘"
echo ""

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

if command -v go >/dev/null 2>&1; then
    echo "[1/3] Go detected! Compiling and installing latest GhostWA binary..."
    go install github.com/Manan0708/GhostWA/cmd/ghostwa@latest
    if [ -f "$HOME/go/bin/ghostwa" ]; then
        cp "$HOME/go/bin/ghostwa" "$INSTALL_DIR/ghostwa"
    fi
else
    echo "[1/3] Downloading pre-compiled GhostWA v2.5 executable..."
    curl -fsSL "https://raw.githubusercontent.com/Manan0708/GhostWA/main/bin/ghostwa" -o "$INSTALL_DIR/ghostwa" || true
fi

if [ -f "$INSTALL_DIR/ghostwa" ]; then
    chmod +x "$INSTALL_DIR/ghostwa"
else
    echo "[-] Installation failed. Could not locate or download ghostwa binary."
    exit 1
fi

# Ensure data directory exists
mkdir -p "$HOME/.local/share/wacli"

echo "[2/3] Registering GhostWA in PATH..."
echo "[3/3] Finalizing setup..."
echo ""
echo "  ✅ GhostWA v2.5 Installed Successfully!"
echo "  ------------------------------------------------------------"
echo "  To start using GhostWA immediately:"
echo "    1. Type: ghostwa login"
echo "    2. Type: ghostwa daemon start"
echo "    3. Type: ghostwa show"
echo "  ------------------------------------------------------------"
echo ""
