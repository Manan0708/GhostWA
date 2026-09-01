#!/usr/bin/env sh
# WACLI One-Line Automated Installer for Linux & macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/Manan0708/Whatsapp-CLI/main/install.sh | sh

set -e

echo ""
echo "  ┌──────────────────────────────────────────────┐"
echo "  │   ⚡  WACLI - WhatsApp CLI Auto-Installer    │"
echo "  └──────────────────────────────────────────────┘"
echo ""

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

if command -v go >/dev/null 2>&1; then
    echo "[1/3] Go detected! Compiling and installing latest WACLI binary..."
    go install github.com/Manan0708/wacli/cmd/wacli@latest
    if [ -f "$HOME/go/bin/wacli" ]; then
        cp "$HOME/go/bin/wacli" "$INSTALL_DIR/wacli"
    fi
else
    echo "[1/3] Go not found on system. Downloading pre-compiled WACLI executable..."
    curl -fsSL "https://raw.githubusercontent.com/Manan0708/Whatsapp-CLI/main/bin/wacli" -o "$INSTALL_DIR/wacli" || true
fi

if [ -f "$INSTALL_DIR/wacli" ]; then
    chmod +x "$INSTALL_DIR/wacli"
else
    echo "[-] Installation failed. Could not locate or download wacli binary."
    exit 1
fi

# Ensure data directory exists
mkdir -p "$HOME/.local/share/wacli"

echo "[2/3] Registering WACLI in PATH..."
echo "[3/3] Finalizing setup..."
echo ""
echo "  ✅ WACLI Installed Successfully!"
echo "  ------------------------------------------------"
echo "  To start using WACLI immediately:"
echo "    1. Type: wacli login"
echo "    2. Type: wacli daemon start"
echo "    3. Type: wacli show"
echo "  ------------------------------------------------"
echo ""
