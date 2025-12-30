#!/bin/bash
set -e

INSTALL_DIR="$HOME/.codex-launcher"
PORT=55136
REPO="dinesh883248/codex-launcher"

echo "Installing Codex Launcher..."

# Create install directory
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

# Download latest release binary
echo "Downloading binary..."
curl -sL "https://github.com/$REPO/releases/latest/download/codex-launcher" -o codex-launcher
chmod +x codex-launcher

# Kill existing process if any
pkill -f "codex-launcher" 2>/dev/null || true
sleep 1

# Start in background
nohup ./codex-launcher -db "$INSTALL_DIR/db.sqlite3" > "$INSTALL_DIR/launcher.log" 2>&1 &

echo ""
echo "Codex Launcher is running!"
echo ""
echo "  URL: http://127.0.0.1:$PORT"
echo ""
echo "  Logs: tail -f $INSTALL_DIR/launcher.log"
echo "  Stop: pkill -f codex-launcher"
echo ""
