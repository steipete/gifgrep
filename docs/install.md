---
title: Install
description: "Install gifgrep via Homebrew, go install, or by grabbing a binary from GitHub Releases."
---

# Install

`gifgrep` is a single Go binary. No background daemons. No config file. No telemetry.

## Homebrew (macOS, Linux)

```bash
brew install steipete/tap/gifgrep
gifgrep --version
```

This is the recommended path — `brew upgrade` will pick up new releases.

## go install

```bash
go install github.com/steipete/gifgrep/cmd/gifgrep@latest
```

You'll need Go ≥ 1.25 and `$(go env GOPATH)/bin` on your `PATH`.

## Pre-built binaries

Download for your platform from the [latest release](https://github.com/steipete/gifgrep/releases/latest):

- macOS (`darwin_amd64`, `darwin_arm64`)
- Linux (`linux_amd64`, `linux_arm64`)

Unpack, drop the binary on your `PATH`, done.

## From source

```bash
git clone https://github.com/steipete/gifgrep.git
cd gifgrep
make build       # builds ./bin/gifgrep
./bin/gifgrep --version
```

## Verify

```bash
gifgrep --version
gifgrep --help
```

If `--version` prints something like `gifgrep 0.2.x`, you're good. Continue to the [Quickstart](quickstart.md).

## API keys (one-time)

Pick a provider — see [Providers](providers/) for the full breakdown:

```bash
# GIPHY (preferred when set)
export GIPHY_API_KEY=your-key

# KLIPY (fallback / free tier)
export KLIPY_API_KEY=your-key
```

`gifgrep` does not write or persist these keys; it only reads them from the environment.
