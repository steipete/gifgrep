---
title: GIPHY provider
description: "Use GIPHY as gifgrep's GIF backend."
---

# GIPHY

GIPHY is gifgrep's recommended provider — broad catalogue, good metadata, fast search.

## Enable

```bash
export GIPHY_API_KEY=your-key
gifgrep --source giphy cats
```

When `GIPHY_API_KEY` is set, [`--source auto`](auto.md) prefers GIPHY automatically.

## Get a key

1. Visit <https://developers.giphy.com>.
2. Create an app — "API" tier is free for personal use.
3. Copy the API key. Store it in your shell profile or a secret manager:

```bash
echo 'export GIPHY_API_KEY=...' >> ~/.zshrc
```

`gifgrep` only reads the variable; it never writes or persists it.

## What gifgrep sends

A standard GIPHY v1 search:

```text
GET https://api.giphy.com/v1/gifs/search
    ?q=<query>
    &api_key=<key>
    &limit=<--max>
    &rating=g
```

Responses are normalised into the [`Result`](../json.md#fields) shape:

- `url` ← `images.original.url`
- `preview_url` ← `images.fixed_width_small.url`
- `width` / `height` ← `images.original.width`/`.height`
- `tags` is omitted (GIPHY doesn't expose a native tag list on search).

## Limits and ratings

- gifgrep currently requests `rating=g`. Higher-rating content is filtered server-side.
- Default `--max` is 20; the GIPHY API caps individual requests at 50.
- Be polite to your quota: avoid tight loops without `sleep`/backoff.

## When to choose GIPHY explicitly

```bash
gifgrep --source giphy "office handshake"
```

- Broad meme coverage and consistent quality.
- Good per-result metadata.
- Slightly stricter content filter (`rating=g`).

If GIPHY returns nothing for a query you expect, try `--source klipy` as a sanity check — different catalogues, different hits.

## See also

- [`auto`](auto.md) — auto-resolution rules.
- [KLIPY](klipy.md) — the other provider.
- [Search](../search.md) — full CLI surface.
