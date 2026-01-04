# Changelog

## 0.2.2 — Unreleased

## 0.2.1 - 2026-01-04

### Fixes
- iTerm2 CLI `--thumbs`: animated GIFs (send raw bytes), correct title/URL alignment, wrap URLs, and add spacing between results.

### Docs
- iTerm2 protocol notes: document raw byte sending and aspect-ratio behavior for CLI vs TUI.

## 0.2.0 - 2026-01-01

### Features
- Inline previews: add iTerm2 support (OSC 1337) for TUI preview and CLI `--thumbs`.
- Inline detection: robust Kitty graphics probing (a=q), plus `GIFGREP_INLINE=kitty|iterm|none`.

### Fixes
- TUI: `f` (reveal) now auto-downloads the selected GIF if needed, then reveals it.
- TUI: avoid emitting Kitty graphics sequences on unsupported terminals (no more base64 spew).
- TUI: when inline images aren’t supported, exit with a helpful “supported terminals / protocol” message.
- TUI: hint row centering (wide glyphs).
- iTerm2: keep animated previews running after UI redraws (don’t clear preview every render).
- iTerm2: clear previous preview when re-sending (avoid “stacked” images after reveal/resize).
- iTerm2: fix repaint artifacts when preview size changes (gap clean + row erases + hard clear on shrink).
- TUI: show download/reveal transient status in the header (no “sticky” status row spam).
- TUI: clearer iTerm send pipeline when resizing/moving preview rect.

### Dev
- Replace pnpm workflow with `make` + npm (`make snap`, `make gifgrep ...`).
- Drop Node `run-go` wrapper; Go-only runner.
- Add macOS terminal capability e2e smoke: `make termcaps-e2e`.
- Tests: add iTerm2 redraw regression for inline previews.
- Make: `--` passthrough (`make gifgrep -- --version`), `gifgrek` alias, always rebuild `make gifgrep ...`.

### Docs
- Add `docs/kitty.md` and `docs/iterm.md` (protocol notes).

## 0.1.0 - 2025-12-31

First public release of gifgrep — GIF search for terminals: scriptable CLI output plus a TUI with inline previews.

### Highlights
- Fast CLI search (`gifgrep <query...>` / `gifgrep search`) with Tenor + Giphy sources.
- Output formats for humans and pipes: plain (TTY default), URL-only, TSV, Markdown, and comment style (plus JSON).
- Inline previews:
  - TUI browser with inline preview + animation (Kitty graphics, with Ghostty software fallback).
  - Optional inline thumbnails in search output (`--thumbs`) that render into scrollback.
- Stills: extract a single PNG frame or a sampled “contact sheet” PNG from any GIF (file or URL).
- Convenience: download/reveal in file manager (`--download`/`--reveal` in CLI, `d`/`f` in TUI), plus rich color help/output.

### Feature overview
- **Global flags**
  - `--color/--no-color` for rich TTY output.
  - `--quiet/--verbose` for stderr logging.
  - `--reveal` to reveal output files in the OS file manager.

- **CLI search**
  - `gifgrep search --source auto|tenor|giphy --max N <query...>` (bare `gifgrep <query...>` is an alias).
  - `--format auto|plain|tsv|md|url|comment|json` and `--json` for structured output.
  - `--thumbs auto|always|never` for Kitty thumbnails in plain output (TTY-only).
  - `--download` to save results to `~/Downloads` (combine with `--reveal`).

- **Interactive TUI**
  - `gifgrep tui [query...]` with arrow-key navigation, quick search editing (`/`), and key hints.
  - Aspect-ratio-aware inline preview; animated playback via Kitty image protocol or software redraw.
  - `d` downloads the current selection to `~/Downloads`; `f` reveals the last download.

- **Stills & sheets**
  - `gifgrep still <gif> --at <time> -o <file|->` (single frame PNG).
  - `gifgrep sheet <gif> --frames N --cols N --padding N -o <file|->` (sampled PNG grid).

- **Providers**
  - Tenor search (defaults to the public demo key; `TENOR_API_KEY` optional).
  - Giphy search (`GIPHY_API_KEY` required); `--source auto` prefers Giphy when keyed.
