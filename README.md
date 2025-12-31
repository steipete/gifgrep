# 🧲 gifgrep — Grep the GIF. Stick the landing.

gifgrep searches GIF providers and gives you two fast paths: scriptable CLI output for pipes, and an interactive TUI with inline previews.

<!-- Screenshot coming soon -->

CLI • TUI • Stills (PNG) • Kitty previews

Website: `https://gifgrep.com`

Homebrew (recommended):

```bash
brew install steipete/tap/gifgrep
```

Go install:

```bash
go install github.com/steipete/gifgrep/cmd/gifgrep@latest
```

## Features

- Scriptable search: readable plain output by default (TTY), plus `--format`, `--json`, `--max`, `--source`.
- Inline thumbnails in search output: `--thumbs` (Kitty graphics; TTY only).
- TUI browser: inline preview, quick download (`d`), reveal last download (`f`).
- Stills: `still` extracts one frame; `sheet` creates a PNG grid (`--frames`, `--cols`, `--padding`).
- Color + logging: `--color/--no-color`, `--quiet`, `--verbose`.
- Providers: `auto` (prefers Giphy when keyed), `tenor`, `giphy`.

## Quickstart

```bash
gifgrep cats --max 5
gifgrep cats --format url | head -n 5
gifgrep search --json cats | jq '.[0].url'
gifgrep tui "office handshake"

gifgrep still ./clip.gif --at 1.5s -o still.png
gifgrep sheet ./clip.gif --frames 9 --cols 3 -o sheet.png
```

## Providers

Select via `--source` (search + TUI):

- `auto` (default): picks Giphy when `GIPHY_API_KEY` is set, else Tenor.
- `tenor`: uses public demo key if `TENOR_API_KEY` is unset.
- `giphy`: requires `GIPHY_API_KEY`.

## CLI

```text
gifgrep [global flags] <query...>
gifgrep search [flags] <query...>
gifgrep tui [flags] [<query...>]
gifgrep still <gif> --at <time> [-o <file>|-]
gifgrep sheet <gif> [--frames <N>] [--cols <N>] [--padding <px>] [-o <file>|-]
```

## JSON output

`--json` prints an array with: `id`, `title`, `url`, `preview_url`, `tags`, `width`, `height`.

## Environment

- `TENOR_API_KEY` (optional)
- `GIPHY_API_KEY` (required for `--source giphy`)
- `GIFGREP_SOFTWARE_ANIM=1` (force software animation)
- `GIFGREP_CELL_ASPECT=0.5` (tweak preview cell geometry)

## Test fixtures licensing

See `docs/gif-sources.md`.

## Development

```bash
go test ./...
go run ./cmd/gifgrep --help
```

Ghostty web snapshot:

```bash
pnpm install
pnpm snap
```

## GitHub Pages

Landing page lives in `docs/` (GitHub Pages -> `main` -> `/docs`).
