# gifgrep - grep the gif

Two modes, one tool.
- Scriptable CLI: URLs/JSON for pipes
- TUI: arrow-key browse + inline preview (Kitty graphics)
- Stills: extract a single frame or a contact sheet (grid of key frames)

Website: `https://gifgrep.com`

## Install
- Homebrew: `brew install steipete/tap/gifgrep`
- Go: `go install github.com/steipete/gifgrep/cmd/gifgrep@latest`

## Quickstart
```bash
gifgrep cats -m 5
gifgrep cats --json | jq '.[] | .url'
gifgrep --tui "office handshake"

gifgrep --gif ./clip.gif --still 1.5s --out still.png
gifgrep --gif ./clip.gif --stills 9 --stills-cols 3 --out sheet.png
```

## Contact sheet
Single PNG grid of sampled frames. Use `--stills` to pick how many frames, `--stills-cols` to control the grid.

## Providers
Select via `--source`:
- `tenor` (default) - uses public demo key if `TENOR_API_KEY` unset
- `giphy` - requires `GIPHY_API_KEY`
- `auto` - picks giphy when `GIPHY_API_KEY` is set, else tenor

## Flags
```text
gifgrep [flags] <query>
gifgrep --tui [flags] <query>
gifgrep --gif <path|url> --still <time> [--out <file>]
gifgrep --gif <path|url> --stills <N> [--stills-cols <N>] [--out <file>]

-i            ignore case
-v            invert vibe (exclude mood)
-E            regex filter over title+tags
-n            number results
-m <N>        max results
--mood <s>    mood filter
--json        json output
--tui         interactive mode
--source <s>  source (tenor, giphy, auto)
--gif <s>     gif input path or URL
--still <s>   extract still at time (e.g. 1.5s)
--stills <N>  contact sheet frame count
--stills-cols <N>    contact sheet columns
--stills-padding <N> contact sheet padding (px)
--out <s>     output path or '-' for stdout
--version     show version
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
