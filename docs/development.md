---
title: Development
description: "Build gifgrep, regenerate its docs, and capture the Ghostty web snapshot."
---

# Development

## Go workflow

Run the test suite and static checks before submitting a change:

```bash
make test
make check
make gifgrep -- --help
```

## Documentation site

The Markdown sources and generated GitHub Pages site live in `docs/`.

```bash
npm install
make docs-site
```

## Ghostty web snapshot

Install Playwright's Chromium build once, then run the snapshot target:

```bash
npx playwright install chromium
make snap
```

`make snap` starts the Ghostty web demo, runs the TUI with the KLIPY provider, and writes `ghostty-web-snap.png` at the repository root. It requires `KLIPY_API_KEY` in the shell environment.
