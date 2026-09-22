---
title: Releasing
description: "Ship signed and notarized gifgrep binaries with the shared release workflow."
---

# Releasing

The `Release` workflow calls the Go CLI archetype from `openclaw/release-workflows`, pinned to v1.10.0 (`d22bcb545b8e43b51fb332be6bcd97b5e7d40234`). Its `personal` policy requires **Developer ID Application: Peter Steinberger (Y5PE65HELJ)** and the identifier `com.steipete.gifgrep.gifgrep`.

## Prepare

Merge the release changes through a reviewed PR with green CI. Finalize the versioned `Unreleased` changelog section with its date and highlights. Update `internal/model/types.go`, `package.json`, `package-lock.json`, and the installation version example, then regenerate the docs with `make docs-site`.

Run the local gate on macOS:

```bash
actionlint
./scripts/check-release-metadata
test -z "$(gofumpt -l .)"
make lint test check
goreleaser check
goreleaser release --snapshot --clean --skip=publish --parallelism=2
```

The four targets remain `darwin_amd64`, `darwin_arm64`, `linux_amd64`, and `linux_arm64`. Archive names remain `gifgrep_<version>_<target>.tar.gz`, containing `gifgrep`, `README.md`, and `LICENSE`. `SHA256SUMS` replaces the old per-archive `.sha256` files. `ASSET-INVENTORY.json` binds each payload to the repository, tag, commit, size, and digest; `RELEASE-NOTES.md` contains the exact dated changelog section.

## macOS minimum

Release binaries support **macOS 13.0 or newer**, the minimum emitted by the pinned Go 1.27 toolchain. GoReleaser uses `CGO_ENABLED=0` and `MACOSX_DEPLOYMENT_TARGET=13.0`; every Darwin build runs `scripts/check-macos-target`, which requires the actual Mach-O deployment target to equal 13.0. Both CI and release builds run on macOS so `otool` can inspect both architecture slices before publication.

The environment variable alone does not prove the binary minimum. If cgo is introduced, also supply `-mmacosx-version-min=13.0` in the C compiler and linker flags and keep this artifact check. Do not let the runner SDK silently raise the minimum.

## Dispatch

The repository must protect `main`, allow Actions read/write workflow permissions, and enable `can_approve_pull_request_reviews` so the workflow can open its closeout PR. Each workflow still declares its own limited permissions.

Provision these repository secrets: `MACOS_SIGN_P12`, `MACOS_SIGN_P12_PASSWORD`, `ASC_KEY_ID`, `ASC_ISSUER_ID`, `ASC_PRIVATE_KEY`, and `HOMEBREW_TAP_TOKEN`. The caller maps them to the shared workflow's signing and tap inputs. The tap token needs Contents read and Actions write access to `steipete/homebrew-tap`.

After the release PR merges, wait for CI on the exact `main` commit. Recheck the latest releases and local/remote tags; if the requested tag already exists, investigate the existing run instead of starting a new release.

```bash
gh release list --repo steipete/gifgrep --limit 3 --json tagName
git fetch origin --tags
git tag --list 'v<version>'
gh workflow run release.yml --repo steipete/gifgrep --ref main -f version=<version>
```

Do not create or push the tag manually. The shared workflow freezes the protected head, checks its CI and metadata, and creates an annotated tag. It builds without signing secrets, signs and notarizes the frozen Darwin binaries in a separate job, then independently verifies immutable release bytes on Intel and Apple Silicon. The publisher requires both attestations and re-downloads the draft assets to verify exact byte equality before publication. A failed run can be rerun against its frozen annotated tag; never move or replace that tag.

The Homebrew handoff dispatches `steipete/homebrew-tap` with the verified asset mapping, waits for the update, and verifies the formula URLs and hashes. After success, review and merge the workflow's next-Unreleased PR, preserving the versioned changelog convention.

## Verify the release

Download all assets fresh, verify `shasum -a 256 -c SHA256SUMS`, compare REST asset digests and inventory with the tagged commit, and compare the release body to `RELEASE-NOTES.md` and the finalized changelog section. For a downloaded macOS binary:

```bash
xattr -l ./gifgrep
codesign -dvv ./gifgrep
codesign --verify --deep --strict --verbose=4 ./gifgrep
codesign --verify --strict --check-notarization -R=notarized ./gifgrep
spctl -a -vv -t open --context context:primary-signature ./gifgrep
otool -l ./gifgrep | grep -A5 LC_BUILD_VERSION
./gifgrep --version
```

The `spctl` command uses the primary-signature **open** context, which accepts notarized bare CLIs. Do not substitute `--type execute`: that assessment expects an app bundle and can reject a valid CLI. Online `codesign --check-notarization` remains the shared workflow's notarization gate.

Use a quarantined download for the Gatekeeper execution check. `curl` does not necessarily set quarantine; if absent, explicitly apply `com.apple.quarantine` and record that it was added for the test. Do not remove quarantine to make the check pass. The expected authority is Peter Steinberger, team `Y5PE65HELJ`, and `minos` is `13.0`.

Finally verify the Go proxy with `GOPROXY=https://proxy.golang.org go list -m github.com/steipete/gifgrep@v<version>` and upgrade the Homebrew formula, checking its installed version and signature. Leave the checkout clean on `main` at `origin/main`.
