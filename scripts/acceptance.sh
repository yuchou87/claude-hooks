#!/usr/bin/env bash
# claude-hooks Acceptance Test Runner
# Usage: ./scripts/acceptance.sh [--verbose]
# Exit 0 if all pass, 1 if any fail.

set -euo pipefail

VERBOSE=${1:-""}
PASS=0
FAIL=0
ERRORS=()
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$WORKDIR/claude-hooks"

# Build fresh binary
if [ ! -f "$REPO_ROOT/go.mod" ]; then
  echo "SKIP: no go.mod found — acceptance runner requires Go source files"
  echo "Results: 0 passed, 0 failed (skipped — no binary to test)"
  exit 0
fi
echo "Building claude-hooks..."
go build -o "$BIN" "$REPO_ROOT" 2>&1

log()  { echo "$1"; }
ok()   { PASS=$((PASS+1)); log "  ✅ PASS  $1"; }
fail() { FAIL=$((FAIL+1)); ERRORS+=("$1"); log "  ❌ FAIL  $1"; }

run() {
  local id="$1"; local desc="$2"; shift 2
  [ -n "$VERBOSE" ] && { echo; log "--- $id: $desc ---"; }
  set +e
  eval "$@" > "$WORKDIR/out" 2>&1
  local actual_code=$?
  set -e
  if [ "$actual_code" -eq 0 ]; then
    ok "$id: $desc"
  else
    fail "$id: $desc  [exit $actual_code]"
    [ -n "$VERBOSE" ] && cat "$WORKDIR/out"
  fi
}

contains() {
  local id="$1"; local desc="$2"; local pattern="$3"; shift 3
  [ -n "$VERBOSE" ] && { echo; log "--- $id: $desc ---"; }
  eval "$@" > "$WORKDIR/out" 2>&1 || true
  if grep -q "$pattern" "$WORKDIR/out"; then
    ok "$id: $desc"
  else
    fail "$id: $desc  [pattern '$pattern' not found]"
    [ -n "$VERBOSE" ] && cat "$WORKDIR/out"
  fi
}

exits_with() {
  local id="$1"; local desc="$2"; local expected_code="$3"; shift 3
  [ -n "$VERBOSE" ] && { echo; log "--- $id: $desc ---"; }
  set +e
  eval "$@" > "$WORKDIR/out" 2>&1
  local actual_code=$?
  set -e
  if [ "$actual_code" -eq "$expected_code" ]; then
    ok "$id: $desc"
  else
    fail "$id: $desc  [expected exit $expected_code, got $actual_code]"
    [ -n "$VERBOSE" ] && cat "$WORKDIR/out"
  fi
}

skip() {
  local id="$1"; local desc="$2"; local reason="${3:-}"
  log "  ⏭  SKIP  $id: $desc${reason:+  [$reason]}"
}

# ── Scenarios ────────────────────────────────────────────────────────────────

echo ""
echo "=== Core / CLI ==="
contains "AT-001" "--version flag"   "claude-hooks"  "$BIN --version"
contains "AT-002" "--help lists run" "run"            "$BIN --help"

# ── Summary ──────────────────────────────────────────────────────────────────

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ ${#ERRORS[@]} -gt 0 ]; then
  echo "Failed:"
  for e in "${ERRORS[@]}"; do echo "  - $e"; done
  exit 1
fi
exit 0
