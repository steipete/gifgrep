---
title: Commands
description: "Every gifgrep subcommand at a glance."
---

# Commands

Every subcommand has its own page. The CLI itself is `gifgrep --help`.

## Top-level

```text
gifgrep <command> [flags]
```

| Command                     | What it does                                                    |
|-----------------------------|-----------------------------------------------------------------|
| [`gifgrep search`](search.md) | Search a provider; print URLs, JSON, or markdown.             |
| [`gifgrep tui`](tui.md)       | Interactive browser with inline animated previews.            |
| [`gifgrep still`](still.md)   | Extract one frame from a GIF as PNG.                          |
| [`gifgrep sheet`](sheet.md)   | Generate a contact-sheet PNG of N frames.                     |

`gifgrep <query>` is shorthand for `gifgrep search <query>` — pick whichever feels natural.

## Global flags

```text
--color <auto|always|never>   color output
--no-color                    shorthand for --color=never
--reveal                      after a write, open the file/folder
-v / -vv                      verbose stderr
-q                            quiet stderr
--version                     print version and exit
-h / --help                   help (per-command when used after a subcommand)
```

## Environment

| Variable                  | Meaning                                                  |
|---------------------------|----------------------------------------------------------|
| `GIPHY_API_KEY`           | Required for `--source giphy` (and preferred by `auto`). |
| `KLIPY_API_KEY`           | Required for `--source klipy` / `--source tenor`.        |
| `GIFGREP_CACHE`          | `1` to enable persistent explicit-download reuse in search/TUI; off by default. |
| `GIFGREP_CACHE_DIR`      | Cache root; requires `--cache` or `GIFGREP_CACHE=1`. See [download cache](search.md#download-cache). |
| `GIFGREP_SOFTWARE_ANIM`   | `1` to force software animation; `0` to disable.         |
| `GIFGREP_CELL_ASPECT`     | Tweak preview cell aspect ratio (default `0.5`).         |
| `GIFGREP_INLINE`          | `kitty` / `iterm` / `sixel` / `ansi` / `off` — override protocol detection. |

## Related

- [JSON output](json.md) — stable envelope for scripts and agents.
- [Providers](providers/) — GIPHY vs KLIPY vs auto.
- [Previews](previews.md) — Kitty / Ghostty / iTerm2 protocol details.
