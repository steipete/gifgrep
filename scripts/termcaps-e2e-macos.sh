#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
bin="$tmp_dir/termcaps-check"

cleanup() { rm -rf "$tmp_dir"; }
trap cleanup EXIT

cd "$repo_root"
go build -o "$bin" ./cmd/termcaps-check

pass() { printf "PASS %s\n" "$1"; }
fail() { printf "FAIL %s\n" "$1"; exit 1; }
skip() { printf "SKIP %s\n" "$1"; }

run_check() {
  local label="$1"
  local expect="$2"
  local out="$tmp_dir/${label//[^A-Za-z0-9_.-]/_}.json"
  shift 2

  if env -i PATH="$PATH" HOME="$HOME" "$@" "$bin" --expect "$expect" --json >"$out" 2>&1; then
    pass "$label"
  else
    echo "FAIL $label"
    cat "$out"
    exit 1
  fi
}

run_pty_check() {
  local label="$1"
  local expect="$2"
  local out="$tmp_dir/${label//[^A-Za-z0-9_.-]/_}.json"
  local transcript="$tmp_dir/${label//[^A-Za-z0-9_.-]/_}.pty"
  shift 2

  if ! command -v script >/dev/null 2>&1; then
    skip "$label (script not installed)"
    return
  fi

  local exports=""
  local kv
  for kv in "$@"; do
    exports+="export ${kv}; "
  done

  if script -q "$transcript" /bin/zsh -lc "${exports}\"$bin\" --expect \"$expect\" --json >\"$out\" 2>&1" >/dev/null 2>&1; then
    pass "$label"
  else
    echo "FAIL $label"
    [[ -f "$out" ]] && cat "$out"
    [[ -f "$transcript" ]] && cat "$transcript"
    exit 1
  fi
}

echo "termcaps e2e (deterministic)"

# Current terminal sanity. No expectation: this only proves the checker runs in
# the caller's actual environment.
current_out="$tmp_dir/current.json"
if "$bin" --json >"$current_out"; then
  pass "current-terminal"
else
  fail "current-terminal"
fi

# Explicit overrides cover user-forced code paths without opening GUI apps.
run_check "override-none" "none" GIFGREP_INLINE=none
run_check "override-kitty" "kitty" GIFGREP_INLINE=kitty
run_check "override-iterm" "iterm" GIFGREP_INLINE=iterm
run_check "override-iterm2" "iterm" GIFGREP_INLINE=iterm2

# Environment detection without GUI automation.
run_check "env-apple-terminal" "none" TERM_PROGRAM=Apple_Terminal
run_check "env-iterm-program" "iterm" TERM_PROGRAM=iTerm.app
run_check "env-iterm-session" "iterm" ITERM_SESSION_ID=test-session
run_check "env-kitty-window" "kitty" KITTY_WINDOW_ID=test-window
run_check "env-ghostty-program" "kitty" TERM_PROGRAM=Ghostty

# PTY smoke: proves the checker also works when stdin/stdout are terminal-like.
run_pty_check "pty-none" "none" GIFGREP_INLINE=none
run_pty_check "pty-kitty-override" "kitty" GIFGREP_INLINE=kitty

echo "Outputs:"
ls -1 "$tmp_dir"/*.json 2>/dev/null || true
