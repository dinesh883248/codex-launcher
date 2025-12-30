#!/bin/bash
set -e

INSTALL_DIR="$HOME/.codex-launcher"
PORT=55136
REPO="dinesh883248/codex-launcher"

echo "Installing Codex Launcher..."

# Create install directory
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

# Download latest release binaries
echo "Downloading binaries..."
curl -sL "https://github.com/$REPO/releases/latest/download/codex-launcher-web" -o codex-launcher-web
curl -sL "https://github.com/$REPO/releases/latest/download/codex-launcher-worker" -o codex-launcher-worker
chmod +x codex-launcher-web codex-launcher-worker

# Check if tmux is installed
if ! command -v tmux &> /dev/null; then
    echo "Error: tmux is required but not installed."
    exit 1
fi

# Kill existing sessions if any
tmux kill-session -t codex-launcher 2>/dev/null || true

# Start tmux session with server and worker
tmux new-session -d -s codex-launcher -n server "./codex-launcher-web -addr :$PORT -db $INSTALL_DIR/db.sqlite3"
tmux new-window -t codex-launcher -n worker "./codex-launcher-worker -db $INSTALL_DIR/db.sqlite3"

echo ""
echo "Codex Launcher is running!"
echo ""
echo "  URL: http://127.0.0.1:$PORT"
echo ""
echo "  tmux attach -t codex-launcher  # to view logs"
echo "  tmux kill-session -t codex-launcher  # to stop"
echo ""
