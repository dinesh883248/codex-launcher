# Codex Launcher

A web interface for running codex requests and viewing responses.

## Quick Install

```bash
curl -sL https://github.com/dinesh883248/codex-launcher/releases/latest/download/install.sh | bash
```

This installs to `~/.codex-launcher` and starts the server on http://127.0.0.1:55136

## Requirements

- Go 1.21+ (for building from source)
- DejaVu fonts (for terminal image generation)

## Build from Source

```bash
go build -o codex-launcher ./cmd/launcher
```

## Run

```bash
./codex-launcher -db db.sqlite3
```

### Options

- `-addr` - Listen address (default: `:55136`)
- `-db` - SQLite database path (default: `db.sqlite3`)
- `-codex` - Codex binary path (default: `codex`)
- `-model` - Codex model (default: `gpt-5.2-codex`)
- `-reasoning` - Reasoning effort (default: `high`)
- `-workdir` - Working directory for codex

## Features

- Submit requests via web form
- View processing status with auto-refresh
- Final response rendered as terminal-style image with markdown support
