---
title: Search
description: "gifgrep search — pipe-friendly GIF search from your terminal."
---

# `gifgrep search`

Search a provider for GIFs and print results. This is what `gifgrep <query>` runs by default.

```text
gifgrep <query> ... [flags]
gifgrep search <query> ... [flags]
```

## At a glance

```bash
gifgrep cats                              # plain on TTY, URL-per-line in pipes
gifgrep cats --max 5                      # cap results
gifgrep cats --json | jq '.[0].url'       # structured
gifgrep cats --format md                  # markdown image links
gifgrep cats --thumbs                     # inline still thumbs (Kitty/iTerm2 TTY)
gifgrep cats --download --max 1           # save to ~/Downloads
gifgrep --source giphy "office handshake" # force GIPHY
```

## Output formats

| Mode               | What you get                                                  |
|--------------------|---------------------------------------------------------------|
| `--format auto`    | Plain readable list on TTY, URL-per-line on pipes (default).  |
| `--format url`     | One URL per line. Best for `xargs`, `wget`, or `pbcopy`.      |
| `--format plain`   | `title — url` per line, no decoration.                        |
| `--format tsv`     | `id<TAB>title<TAB>url<TAB>preview_url<TAB>w<TAB>h`.           |
| `--format md`      | `![title](url)` markdown image lines.                         |
| `--format comment` | URLs prefixed with `# title` for clipboard-friendly snippets. |
| `--format json`    | Same envelope as `--json`.                                    |
| `--json`           | Pretty-printed JSON array — see [JSON output](json.md).       |

`-n` / `--number` prefixes each line with its 1-based index.

## Common flags

```text
--source <auto|giphy|klipy|tenor>   choose a provider (default: auto)
--max, -m <N>                       max results (default 20)
--json                              pretty JSON array
--format <auto|...>                 see above
--number, -n                        prefix lines with 1-based index
--thumbs[=auto|on|off]              inline still thumbs (Kitty/iTerm2, TTY only)
--download                          save results to ~/Downloads
--reveal                            after --download, open the folder
--color <auto|always|never>         color output
--no-color                          shorthand for --color=never
-v / -vv                            stderr debug logs
-q                                  quiet
```

## Inline thumbnails (CLI)

`--thumbs` shows a single still frame next to each result, decoded locally and uploaded via the [Kitty graphics protocol](previews.md#kitty-graphics) (Kitty/Ghostty) or [OSC 1337](previews.md#iterm2-osc-1337) (iTerm2). Notes:

- TTY only — pipes never receive image bytes.
- One **still** frame per row. For animated previews, use the [TUI](tui.md).
- `auto` enables thumbs only when the terminal is detected as supporting inline images.

## Downloading

```bash
gifgrep cats --download --max 3
gifgrep cats --download --reveal --max 1   # also opens Finder/Explorer
```

Downloads land in `~/Downloads`. Filenames use the sanitized result title, falling back to its id or URL; repeated saves receive numeric suffixes.

## Download cache

Use `--cache` to reuse previously downloaded GIFs across CLI and TUI sessions:

```bash
gifgrep cats --download --max 1 --cache
gifgrep tui cats --cache
GIFGREP_CACHE=1 GIFGREP_CACHE_DIR=/path/to/cache gifgrep cats --download
```

Explicit saves still create separate files in `~/Downloads`, including on a cache hit. Caching is off by default; setting a cache directory alone does not enable it. `--cache=false` overrides `GIFGREP_CACHE=1`.

| Flag | Default | Meaning |
| --- | --- | --- |
| `--cache` | `false` | Enable persistent download reuse; also `GIFGREP_CACHE`. |
| `--cache-dir` | OS user cache directory + `gifgrep` | Cache root; also `GIFGREP_CACHE_DIR`. The flag takes precedence. |
| `--cache-max-age` | `168h` | Expire entries seven days after download; `0` disables expiry. |
| `--cache-max-bytes` | `104857600` | Keep up to 100 MiB of cached downloads; `0` disables the size limit. |

The default root is `~/Library/Caches/gifgrep` on macOS and `$XDG_CACHE_HOME/gifgrep` (or `~/.cache/gifgrep`) on Linux. Entries live in its `downloads-v1` subdirectory. Remove that subdirectory to clear the cache without touching saved downloads.

Cache entries contain downloaded GIF bytes under a hash of the complete media URL, so different provider hosts and media variants stay separate. Queries, titles, and provider metadata are not stored. Expired entries and oldest downloads over the size budget are removed when saving; reading an entry does not extend its lifetime. Files larger than the budget are saved normally but not cached. Limits are best effort during concurrent saves, and cache failures fall back to ordinary downloading.

Search still calls the provider API and needs a key. This cache applies to explicit downloads (`--download`, or TUI `d`/`f`); preview/prefetch data stays temporary. It does not provide offline search, a local GIF library, or favorites.

## Pipe recipes

```bash
# Copy first URL to clipboard (macOS)
gifgrep "shipped it" --format url --max 1 | pbcopy

# Drop top 5 URLs into a markdown file
gifgrep "ship it" --format md --max 5 >> notes.md

# Bulk download
gifgrep "deploy" --format url --max 10 | xargs -n1 curl -O

# JSON pipeline with jq
gifgrep "office handshake" --json --max 20 | jq '.[] | select(.width > 400) | .url'
```

## Provider selection

```bash
gifgrep --source auto cats         # auto-pick: GIPHY when keyed, else KLIPY
gifgrep --source giphy cats        # force GIPHY (needs GIPHY_API_KEY)
gifgrep --source klipy cats        # force KLIPY (needs KLIPY_API_KEY)
gifgrep --source tenor cats        # legacy alias for KLIPY
```

Full provider matrix: [Providers](providers/).

## Exit behavior

- `0` — at least one result was printed.
- non-zero — provider error, missing API key, or transport failure (logged to stderr).

Human progress and errors always go to stderr, so `--json` and `--format url` pipes stay parseable.
