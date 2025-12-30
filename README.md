# Codex Launcher

A web interface for running codex requests and viewing responses.

## Quick Install

```bash
curl -sL https://github.com/dinesh883248/codex-launcher/releases/latest/download/install.sh | bash
```

This installs to `~/.codex-launcher` and starts the server on http://127.0.0.1:55136

## Requirements

- Go 1.21+ (for building from source)
- SQLite
- DejaVu fonts (for terminal image generation)
- tmux (for install script)

## Build from Source

```bash
go build -o codex-launcher-web ./cmd/web
go build -o codex-launcher-worker ./cmd/worker
```

## Run Manually

```bash
# Start web server
./codex-launcher-web -addr :55136 -db db.sqlite3

# Start worker (in separate terminal)
./codex-launcher-worker -db db.sqlite3
```

## Features

- Submit requests via web form
- View processing status with auto-refresh
- Final response rendered as terminal-style image with markdown support
