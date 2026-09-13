---
title: still
description: "gifgrep still — extract a single frame from a GIF as PNG."
---

# `gifgrep still`

Extract one frame from a GIF as a PNG. Works on local files **or** URLs, no provider key required.

```text
gifgrep still <gif> --at <duration> [-o <file>|-]
```

## Examples

```bash
# First frame as still.png
gifgrep still cat.gif --at 0 -o still.png

# 1.5s into the GIF, write to stdout (suitable for piping)
gifgrep still cat.gif --at 1.5s -o - > frame.png

# Pull a remote GIF and grab a frame at 250ms
gifgrep still https://example.com/cat.gif --at 0.25s -o cover.png

# Open the result in Preview/Finder after writing it
gifgrep still cat.gif --at 0 -o cover.png --reveal
```

## Flags

```text
--at <duration>            timestamp; e.g. 0, 0.5, 1.5s, 2500ms
-o, --output <file|->      output path (default: still.png), '-' for stdout
--reveal                   open the resulting file in your file manager
-q / -v / --no-color       standard logging flags
```

`--at` accepts seconds (`1.5`, `1.5s`) or milliseconds (`250ms`). If you ask for a timestamp past the GIF's total duration, the last available frame is used.

Extraction uses the complete animation, including frames beyond the TUI's preview limit. Negative, non-finite, and overflowing timestamps are rejected.

## How frame timing works

GIFs encode per-frame delays (in centiseconds). gifgrep accumulates these deltas to map your `--at` value onto the closest preceding frame. There's no resampling — you always get a frame that actually exists in the source.

For a denser look at the GIF, see [`sheet`](sheet.md).

## Writing to stdout

`-o -` pipes raw PNG bytes to stdout. Combine with anything that expects PNG on stdin:

```bash
gifgrep still cat.gif --at 0 -o - | pngquant - > cat-tiny.png
gifgrep still cat.gif --at 0.5s -o - | imagemagick convert - cat.jpg
```

In that mode, all human-facing output stays on stderr.
