# Almono notes (work log)

This file captures context and decisions from recent work so the next person can
understand how the pieces fit together.

## What changed recently
- Added a dedicated request-cast page that uses the same table layout, with the
  Asciinema player mounted inside a td (`id="demo"`).
- Livestream and request-cast pages load local Asciinema assets (JS/CSS) and
  hide controls via `controls: false`.
- Livestream auto-start is forced with autoplay flags and `player.play()`.
- `/stream` now emits a v2-style `{cols,rows}` header for the SSE player when
  the underlying cast is asciicast v3. This is required so the player can
  initialize on live streams.
- AsciinemaPlayer assets are pinned to 3.6.3 (local files under `web/static`).
- Asciinema recorder binary was updated to 3.0.1 (in `.venv/bin/asciinema`) so
  the live cast header is version 3.

## How streaming works
- `/livestream/` renders the HTML page and initializes AsciinemaPlayer.
- `/stream` is SSE and forwards lines from the live cast file. The first line is
  special: for v3 casts the server emits a v2-style header with `cols`/`rows` so
  the player can start rendering immediately.

## Design checks and constraints
- HTML is table-only in `<body>` and widths are enforced (380px) via colgroups.
- `immutable.css` in `<head>` is required for design checks.
- For `request_cast.html`, the design checks allow `link`/`script` tags and a
  `td` with an `id` attribute (needed for the player target).
- Design checks are recorded in `design_checks.md`.

## Bytecode scoring rules
- Python changes require re-running `python3 count_bytecode.py` and updating
  `score.md` with new scores.

## Runtime notes
- Web and worker run in tmux sessions `almono-web` and `almono-worker`.
- The worker is recorded via asciinema, producing `casts/live.cast`.
- The web server serves the player assets and the cast streams.

