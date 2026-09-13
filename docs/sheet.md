---
title: sheet
description: "gifgrep sheet — generate a contact sheet PNG from a GIF."
---

# `gifgrep sheet`

Sample N frames from a GIF and lay them out as a single PNG contact sheet. Local files or URLs, no provider key needed.

```text
gifgrep sheet <gif> [--frames N] [--cols N] [--padding px] [-o <file>|-]
```

## Examples

```bash
gifgrep sheet cat.gif -o sheet.png
gifgrep sheet cat.gif --frames 16 --cols 4 --padding 4 -o sheet.png
gifgrep sheet https://example.com/cat.gif --frames 9 --cols 3 -o - > sheet.png
gifgrep sheet cat.gif --frames 12 -o sheet.png --reveal
```

## Flags

```text
--frames <N>          frames to sample (default 12)
--cols <N>            columns; 0 = auto (default 0)
--padding <px>        pixels of padding between frames (default 2)
-o, --output <file>   output path (default: sheet.png), '-' for stdout
--reveal              open the result in Finder/Explorer after writing
```

## How sampling works

Frames are sampled at evenly-spaced timestamps across the GIF's full duration. If you ask for more frames than the GIF actually has, you get every frame (no duplication, no resampling).

`--cols 0` (the default) picks a column count that yields a roughly square grid; otherwise the layout is `cols × ceil(frames / cols)`.

Output is limited to 40 million pixels. Dimensions or padding that overflow or exceed that budget return an error before allocating the sheet.

## Writing to stdout

```bash
gifgrep sheet cat.gif --frames 9 --cols 3 -o - > sheet.png
gifgrep sheet cat.gif -o - | pngquant - > sheet-small.png
```

In stdout mode, progress and warnings go to stderr.

## Pairs well with

- [`still`](still.md) when you want a single frame at a known timestamp.
- [`search`](search.md) + `--download` to grab the GIF first, then sheet it.

```bash
gifgrep cats --download --max 1 --format url
gifgrep sheet ~/Downloads/cat-typing.gif --frames 12 --cols 4 -o sheet.png
```
