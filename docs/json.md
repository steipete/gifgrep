---
title: JSON output
description: "Stable JSON envelope for gifgrep — designed for scripts, jq pipelines, and coding agents."
---

# JSON output

`gifgrep` is built for pipes, and `--json` is the contract.

## Shape

```bash
gifgrep cats --json --max 2
```

```json
[
  {
    "id": "abc123",
    "title": "Cat typing furiously",
    "url": "https://media.giphy.com/.../cat-typing.gif",
    "preview_url": "https://media.giphy.com/.../cat-typing-tiny.gif",
    "tags": ["cat", "typing", "computer"],
    "width": 480,
    "height": 360
  },
  {
    "id": "def456",
    "title": "Cat says yes",
    "url": "https://media.giphy.com/.../cat-yes.gif",
    "preview_url": "https://media.giphy.com/.../cat-yes-tiny.gif",
    "tags": ["cat", "yes", "approve"],
    "width": 320,
    "height": 240
  }
]
```

## Fields

| Field         | Type      | Notes                                                              |
|---------------|-----------|--------------------------------------------------------------------|
| `id`          | string    | Provider's stable identifier.                                      |
| `title`       | string    | Human title; falls back to provider description, then `id`.        |
| `url`         | string    | Direct GIF URL — what `--format url` prints.                       |
| `preview_url` | string    | Smaller GIF good for thumbnails / inline previews.                 |
| `tags`        | string[]  | Omitted when empty; supplied by KLIPY when available.               |
| `width`       | number    | Original GIF width in pixels; omitted when zero/unknown.          |
| `height`      | number    | Original GIF height in pixels; omitted when zero/unknown.         |

Both providers use this shape. GIPHY results omit tags; either provider can omit unknown dimensions.

## Stability

- New optional fields may be added without warning.
- Existing fields will not be renamed or removed without a major version bump.
- Stdout is JSON; stderr remains human (progress, warnings, errors).
- A non-zero exit code means the array is **not** valid — check stderr.

## Recipes

```bash
# Top URL only
gifgrep cats --json --max 1 | jq -r '.[0].url'

# All preview URLs (for thumbnails in another tool)
gifgrep cats --json --max 20 | jq -r '.[].preview_url'

# Filter to wide GIFs
gifgrep "office handshake" --json --max 50 \
  | jq '[.[] | select(.width >= 480)]'

# Convert to TSV
gifgrep cats --json --max 10 \
  | jq -r '.[] | [.id, .title, .url] | @tsv'

# Pipe into an LLM (URLs only)
gifgrep "ship it" --json --max 10 \
  | jq -r '.[].url' \
  | llm "Pick the most enthusiastic one"
```

## When to use JSON vs `--format`

- **`--json`** — when something downstream parses structured data (`jq`, `python`, an LLM, a script).
- **`--format url`** — when the consumer is `xargs`, `wget`, `pbcopy`, etc.
- **`--format md`** — when you're pasting into a markdown doc.
- **`--format tsv`** — when you want columns without leaving the shell.

See [Search](search.md) for the full format matrix.
