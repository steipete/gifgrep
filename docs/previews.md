---
title: Inline previews
description: "How gifgrep renders animated GIFs inline in Kitty, Ghostty, and iTerm2."
---

# Inline previews

`gifgrep` renders GIFs using the graphics protocol supported by your terminal.

| Terminal       | Protocol                                | TUI animation         | CLI `--thumbs`         |
|----------------|------------------------------------------|-----------------------|------------------------|
| Kitty          | Kitty graphics                           | Native, hardware       | Still frame            |
| Ghostty        | Kitty graphics                           | Software (frame ticks) | Still frame            |
| iTerm2         | OSC 1337 (Inline Images)                 | Native (raw GIF)       | Animated thumb         |
| WezTerm / Windows Terminal | Sixel                        | Software (frame ticks) | Still frame            |
| Truecolor (forced) | ANSI half-blocks                      | Software (frame ticks) | No thumbs              |
| Apple Terminal | None detected                            | Refuses to start unless forced | No thumbs       |

The CLI never outputs image bytes when stdout is a pipe — only on a TTY.

## Kitty graphics

Used by Kitty and Ghostty. The escape envelope:

```text
ESC _G <key=value;...> ; <base64 payload> ESC \
```

`gifgrep` uses these actions:

| Action | What it does                                                |
|--------|-------------------------------------------------------------|
| `a=T`  | Upload a base image (PNG bytes).                            |
| `a=f`  | Append an animation frame with per-frame delay.             |
| `a=a`  | Set animation timing / start playback.                      |
| `a=p`  | Place an image into a cell rectangle.                       |
| `a=d`  | Delete an image by id (cleanup on scroll/quit).             |

Each PNG payload is base64 encoded and chunked into 4096-character pieces to stay under terminal limits.

**Kitty** plays the animation natively once frames are uploaded.

**Ghostty** does not yet animate Kitty graphics natively, so `gifgrep` falls back to **software playback**: it ticks frames on a timer and re-uploads the current frame each time. That's the `GIFGREP_SOFTWARE_ANIM` knob:

```bash
GIFGREP_SOFTWARE_ANIM=1 gifgrep tui cats   # always tick in software
GIFGREP_SOFTWARE_ANIM=0 gifgrep tui cats   # always trust terminal animation
```

By default, software playback is auto-enabled when Ghostty is detected.

Cell aspect ratio for sizing previews:

```bash
GIFGREP_CELL_ASPECT=0.5 gifgrep tui cats   # default
GIFGREP_CELL_ASPECT=0.45 gifgrep tui cats  # tweak if previews look squashed
```

Reference: <https://sw.kovidgoyal.net/kitty/graphics-protocol/>.

## iTerm2 (OSC 1337)

iTerm2 has its own protocol — it accepts the original file bytes directly:

```text
ESC ] 1337 ; File = <key=value;...> : <base64 file bytes> ESC \
```

Keys `gifgrep` sets:

- `name` — base64-encoded filename (used for the iTerm2 download tray).
- `size` — bytes (drives the progress indicator).
- `inline=1` — render inline rather than queueing a download.
- `width` / `height` — sized in **terminal cells**, not pixels.
- `preserveAspectRatio` — `1` in the TUI; `0` when filling a fixed CLI thumbnail block.

iTerm2 plays animated GIFs natively, so the TUI sends the *preview* GIF bytes for each result and lets iTerm2 animate them.

Reference: <https://iterm2.com/documentation-images.html>.

## Detection

`gifgrep` decides which protocol to use from environment hints:

- iTerm2: `TERM_PROGRAM=iTerm.app` or `ITERM_SESSION_ID`.
- Ghostty: `TERM_PROGRAM=ghostty` or `GHOSTTY_RESOURCES_DIR`.
- Kitty: `TERM=xterm-kitty` (or compatible).
- Sixel: `TERM` contains `sixel`, `TERM_PROGRAM=WezTerm`, or `WT_SESSION`.

Truecolor ANSI fallback is opt-in:

```bash
GIFGREP_INLINE=ansi    # render GIF frames as terminal color blocks
```

Override:

```bash
GIFGREP_INLINE=kitty   # force Kitty graphics
GIFGREP_INLINE=iterm   # force OSC 1337
GIFGREP_INLINE=sixel   # force DEC Sixel
GIFGREP_INLINE=ansi    # force truecolor ANSI fallback
GIFGREP_INLINE=off     # disable inline previews entirely
```

## Diagnostics

If previews look wrong:

```bash
gifgrep --version
echo "$TERM $TERM_PROGRAM"
gifgrep tui cats -vv               # very verbose stderr
```

Common gotchas:

- **Pipes never get image bytes.** Inline previews only render to a TTY.
- **tmux** can pass through Kitty graphics with the right config, but it's terminal-dependent. `set -ga terminal-features ',xterm-256color:hyperlinks'` and friends help, but expect rough edges.
- **SSH** works as long as the destination terminal speaks the protocol; the bytes flow through the pipe.

## Related docs

- [TUI](tui.md) — keys, behaviour, when to reach for it.
- [Search `--thumbs`](search.md#inline-thumbnails-cli) — still-frame inline thumbs in pipe-shaped CLI output.
