# 🧲 gifgrep — Grep the GIF. Stick the landing.

Two modes. Same mission.
- **Scriptable CLI (default):** prints GIF URLs (or JSON) for pipes
- **Interactive TUI (`--tui`):** arrow-key browse + inline preview (kitty protocol)

Website: `https://gifgrep.com`

## Install
- Homebrew: `brew install steipete/tap/gifgrep`
- Go: `go install github.com/steipete/gifgrep/cmd/gifgrep@latest`

## Quickstart
```bash
gifgrep cats -m 5
gifgrep cats --json | jq '.[] | .url'
gifgrep --tui "office handshake"
```

## GIF providers
gifgrep supports multiple backends via `--source`:

### Tenor (default)
- Default provider (`--source tenor`)
- `TENOR_API_KEY` is optional; if unset, gifgrep uses Tenor’s public demo key.

### Giphy
- Use via `--source giphy`
- Requires `GIPHY_API_KEY` (no fallback).

Create a GIPHY key:
1) Go to the GIPHY Developer Dashboard
2) Create an app / API key (pick “API”, not “SDK”)
3) Copy the API key

Set it for your shell:
```bash
export GIPHY_API_KEY="…"
```

Then run:
```bash
gifgrep --source giphy cats
gifgrep --source giphy --tui cats
```

## Requirements
- **CLI mode:** any terminal
- **TUI preview:** terminal with Kitty graphics (Kitty, Ghostty)
- **Build from source:** Go 1.25+ (see `go.mod` toolchain)

## Flags
```text
gifgrep [flags] <query>
gifgrep --tui [flags] <query>

-i            ignore case
-v            invert vibe (exclude mood)
-E            regex filter over title+tags
-n            number results
-m <N>        max results
--mood <s>    mood filter
--json        json output
--tui         interactive mode
--source <s>  source (tenor)
--version     show version
```

## JSON output
`--json` prints an array of results like:
- `id`, `title`, `url`, `preview_url`
- `tags` (optional), `width`, `height` (optional)

## Environment variables
- `TENOR_API_KEY` (optional): defaults to Tenor’s public demo key if unset
- `GIPHY_API_KEY` (required for `--source giphy`)
- `GIFGREP_SOFTWARE_ANIM=1` (optional): force software animation (auto-detects Ghostty)
- `GIFGREP_CELL_ASPECT=0.5` (optional): tweak preview sizing for your terminal’s cell geometry

## Development
```bash
go test ./...
go run ./cmd/gifgrep --help
```
If you’re hacking on the Ghostty web snapshot script:
```bash
pnpm install
pnpm snap
```

## GitHub Pages
The landing page lives in `docs/` (set GitHub Pages to “Deploy from a branch” → `main` → `/docs`).
