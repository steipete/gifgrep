# Changelog

## 0.1.1 - Unreleased

### Fixes
- TUI: `f` (reveal) now auto-downloads the selected GIF if needed, then reveals it.

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
