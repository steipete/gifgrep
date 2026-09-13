---
title: Quickstart
description: "Two minutes from brew install to your first GIF URL in stdout."
---

# Quickstart

Two minutes from a clean machine to GIFs in your terminal.

## 1. Install

```bash
brew install steipete/tap/gifgrep
gifgrep --version
```

Other options (Go install, prebuilt binaries, source) are on [Install](install.md).

## 2. Pick a provider

`gifgrep` needs at least one of:

```bash
export GIPHY_API_KEY=your-key   # preferred
# or
export KLIPY_API_KEY=your-key   # also fine
```

Get a GIPHY key at <https://developers.giphy.com>. KLIPY keys come from <https://klipy.com>.

If both are set, GIPHY wins. See [`auto`](providers/auto.md) for the resolution rules.

## 3. Search

```bash
gifgrep cats                # plain on TTY, URLs in pipes
gifgrep cats --max 5        # cap results
gifgrep cats --json | jq    # structured output
gifgrep cats --format url | head -n 3
```

Default behaviour adapts to where you're sending output: TTY gets a readable list, pipes get URL-per-line.

## 4. Browse interactively

```bash
gifgrep tui "office handshake"
```

Keys:

- `/` — edit search
- `↑` / `↓` — select
- `d` — download to `~/Downloads`
- `f` — download if needed and reveal the selected GIF in your file manager
- `q` — quit

In Kitty, Ghostty, or iTerm2 you'll get **animated inline previews**. See [Previews](previews.md) for protocol details.

## 5. Save GIFs

```bash
gifgrep cats --download --max 1 --format url
gifgrep cats --download --reveal --max 1     # also opens Finder/Files
```

Files land in `~/Downloads`.

## 6. Pull frames out of a GIF

```bash
gifgrep still cat.gif --at 1.5s -o still.png       # one frame at 1.5s
gifgrep sheet cat.gif --frames 9 --cols 3          # 3×3 contact sheet
gifgrep still https://media.giphy.com/x.gif --at 0 -o - > frame.png
```

Both [`still`](still.md) and [`sheet`](sheet.md) work on local files **or** URLs, and don't need a provider key.

## Where next

- [Search](search.md) — every flag the CLI accepts.
- [TUI](tui.md) — keyboard-driven browser.
- [JSON output](json.md) — stable envelope for scripts and agents.
- [Providers](providers/) — GIPHY vs KLIPY vs auto.
- [Previews](previews.md) — Kitty / Ghostty / iTerm2 protocol notes.
