# Almono

A web interface for running codex requests and viewing responses.

## Requirements

- Go 1.21+
- SQLite
- DejaVu fonts (for terminal image generation)

## Build

```bash
# Build web server
go build -o almono-web ./cmd/web

# Build worker
go build -o almono-worker ./cmd/worker
```

## Run

```bash
# Start web server
./almono-web -addr :8080 -db db.sqlite3

# Start worker (in separate terminal)
./almono-worker -db db.sqlite3
```

## Configuration

### Web Server Options

- `-addr` - Listen address (default: `:8080`)
- `-db` - SQLite database path (default: `db.sqlite3`)

### Worker Options

- `-db` - SQLite database path (default: `db.sqlite3`)

## Features

- Submit requests via web form
- View processing status with auto-refresh
- Final response rendered as terminal-style image with markdown support
