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

echo ""
echo "=== Command Mode ==="

# AT-003: bad JSON → fail-open (exit 0, stdout must be empty)
exits_with "AT-003a" "bad JSON → fail-open (exit 0)" 0 "echo '{bad}' | $BIN run"
AT003_STDOUT=$(echo '{bad}' | "$BIN" run 2>/dev/null || true)
if [ -z "$AT003_STDOUT" ]; then
  ok "AT-003b: bad JSON → stdout is empty (stdout-purity constraint)"
else
  fail "AT-003b: bad JSON → stdout not empty: $AT003_STDOUT"
fi

# AT-004: bash-safety blocks rm -rf / (exit 2, deny JSON in stdout only)
DENY_PAYLOAD='{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash","tool_input":{"command":"rm -rf /"}}'
exits_with "AT-004a" "rm-rf-slash denied → exit 2" 2 "printf '%s' '$DENY_PAYLOAD' | $BIN run"
AT004_STDOUT=$(printf '%s' "$DENY_PAYLOAD" | "$BIN" run 2>/dev/null || true)
if echo "$AT004_STDOUT" | grep -q "permissionDecision"; then
  ok "AT-004b: rm-rf-slash denied → permissionDecision in stdout (stdout-only check)"
else
  fail "AT-004b: permissionDecision not found in stdout: $AT004_STDOUT"
fi

echo ""
echo "=== HTTP Daemon ==="

# Start daemon in background; kill on exit
DAEMON_PORT=18787  # non-default port to avoid conflicts
$BIN serve --addr "127.0.0.1:$DAEMON_PORT" &
DAEMON_PID=$!
trap 'kill $DAEMON_PID 2>/dev/null; wait $DAEMON_PID 2>/dev/null; rm -rf "$WORKDIR"' EXIT

# Wait up to 2s for daemon to be ready
READY=0
for i in $(seq 1 20); do
  if curl -sf "http://127.0.0.1:$DAEMON_PORT/hook" -X POST -d '{}' -H 'Content-Type: application/json' >/dev/null 2>&1; then
    READY=1; break
  fi
  sleep 0.1
done
if [ $READY -eq 0 ]; then
  fail "AT-005: daemon did not start within 2s"
  fail "AT-006: daemon did not start within 2s"
else
  # AT-005: rm -rf / → bash-safety local rule → deny (no dialog)
  AT005_PAYLOAD='{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash","tool_input":{"command":"rm -rf /"}}'
  AT005_BODY=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT/hook" \
    -H 'Content-Type: application/json' -d "$AT005_PAYLOAD" 2>/dev/null)
  if echo "$AT005_BODY" | grep -q "permissionDecision"; then
    ok "AT-005: HTTP local deny rule fires without dialog"
  else
    fail "AT-005: HTTP local deny rule — permissionDecision missing in body: $AT005_BODY"
  fi

  # AT-006: SessionEnd → no rule, not PreToolUse → empty 200 (no dialog)
  AT006_PAYLOAD='{"hook_event_name":"SessionEnd","session_id":"s","transcript_path":"/t","cwd":"/"}'
  AT006_RESPONSE=$(curl -s -w "__HTTPSTATUS__%{http_code}" -X POST "http://127.0.0.1:$DAEMON_PORT/hook" \
    -H 'Content-Type: application/json' -d "$AT006_PAYLOAD" 2>/dev/null)
  AT006_BODY="${AT006_RESPONSE%__HTTPSTATUS__*}"
  AT006_STATUS="${AT006_RESPONSE##*__HTTPSTATUS__}"
  if [ "$AT006_STATUS" = "200" ] && [ -z "$(echo "$AT006_BODY" | tr -d '[:space:]')" ]; then
    ok "AT-006: HTTP SessionEnd → empty 200 (no dialog)"
  else
    fail "AT-006: HTTP SessionEnd — want 200+empty, got status=$AT006_STATUS body=$AT006_BODY"
  fi

  kill $DAEMON_PID 2>/dev/null
  wait $DAEMON_PID 2>/dev/null
  trap 'rm -rf "$WORKDIR"' EXIT
fi

echo ""
echo "=== Dynamic Rules ==="

# AT-007: YAML deny rule
AT007_DIR=$(mktemp -d)
trap 'rm -rf "$AT007_DIR"' EXIT
cat > "$AT007_DIR/config.yaml" <<'YAML'
rules:
  - name: at007-deny-write
    event: PreToolUse
    when:
      tool: [Write]
    decision: deny
    reason: "AT-007 test deny"
YAML

DAEMON_PORT_AT007=18788
"$BIN" serve --addr "127.0.0.1:$DAEMON_PORT_AT007" --config "$AT007_DIR/config.yaml" &
DAEMON_PID_AT007=$!
trap 'kill $DAEMON_PID_AT007 2>/dev/null; wait $DAEMON_PID_AT007 2>/dev/null; rm -rf "$AT007_DIR"' EXIT

READY_AT007=0
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:$DAEMON_PORT_AT007/hook" -X POST -d '{}' -H 'Content-Type: application/json' >/dev/null 2>&1; then
    READY_AT007=1; break
  fi
  sleep 0.1
done

if [ $READY_AT007 -eq 0 ]; then
  fail "AT-007: daemon did not start within 3s"
else
  AT007_BODY=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT_AT007/hook" \
    -H 'Content-Type: application/json' \
    -d '{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{"file_path":"/tmp/x"}}' 2>/dev/null)
  if echo "$AT007_BODY" | grep -q "permissionDecision"; then
    ok "AT-007: YAML deny rule fires via HTTP daemon"
  else
    fail "AT-007: YAML deny rule — permissionDecision missing in body: $AT007_BODY"
  fi
fi

kill $DAEMON_PID_AT007 2>/dev/null; wait $DAEMON_PID_AT007 2>/dev/null
trap 'rm -rf "$WORKDIR"' EXIT

# AT-008: JS script deny rule
AT008_DIR=$(mktemp -d)
trap 'rm -rf "$AT008_DIR"' EXIT
mkdir -p "$AT008_DIR/scripts"
cat > "$AT008_DIR/scripts/deny_write.js" <<'JS'
export const events = ["PreToolUse"];
export function decide(e) {
    if (e.tool_name === "Write") {
        return { permissionDecision: "deny", permissionDecisionReason: "AT-008 script deny" };
    }
    return null;
}
JS

DAEMON_PORT_AT008=18789
"$BIN" serve --addr "127.0.0.1:$DAEMON_PORT_AT008" --scripts-dir "$AT008_DIR/scripts" &
DAEMON_PID_AT008=$!
trap 'kill $DAEMON_PID_AT008 2>/dev/null; wait $DAEMON_PID_AT008 2>/dev/null; rm -rf "$AT008_DIR"' EXIT

READY_AT008=0
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:$DAEMON_PORT_AT008/hook" -X POST -d '{}' -H 'Content-Type: application/json' >/dev/null 2>&1; then
    READY_AT008=1; break
  fi
  sleep 0.1
done

if [ $READY_AT008 -eq 0 ]; then
  fail "AT-008: daemon did not start within 3s"
else
  AT008_BODY=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT_AT008/hook" \
    -H 'Content-Type: application/json' \
    -d '{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{"file_path":"/tmp/x"}}' 2>/dev/null)
  if echo "$AT008_BODY" | grep -q "permissionDecision"; then
    ok "AT-008: JS script deny rule fires via HTTP daemon"
  else
    fail "AT-008: JS script deny rule — permissionDecision missing in body: $AT008_BODY"
  fi
fi

kill $DAEMON_PID_AT008 2>/dev/null; wait $DAEMON_PID_AT008 2>/dev/null

# AT-009: hot-reload — mutate config file while daemon is running, new rule fires
AT009_DIR=$(mktemp -d)
trap 'rm -rf "$AT009_DIR"' EXIT

# Start with empty rules
cat > "$AT009_DIR/config.yaml" <<'YAML'
rules: []
YAML

DAEMON_PORT_AT009=18790
"$BIN" serve --addr "127.0.0.1:$DAEMON_PORT_AT009" --config "$AT009_DIR/config.yaml" &
DAEMON_PID_AT009=$!
trap 'kill $DAEMON_PID_AT009 2>/dev/null; wait $DAEMON_PID_AT009 2>/dev/null; rm -rf "$AT009_DIR"' EXIT

READY_AT009=0
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:$DAEMON_PORT_AT009/hook" -X POST -d '{}' -H 'Content-Type: application/json' >/dev/null 2>&1; then
    READY_AT009=1; break
  fi
  sleep 0.1
done

if [ $READY_AT009 -eq 0 ]; then
  fail "AT-009: daemon did not start within 3s"
else
  # Confirm YAML rule is NOT yet loaded (response must not contain the YAML rule's reason)
  AT009_BEFORE=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT_AT009/hook" \
    -H 'Content-Type: application/json' \
    -d '{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{"file_path":"/tmp/x"}}' 2>/dev/null)
  if echo "$AT009_BEFORE" | grep -q "AT-009 hot-reload test"; then
    fail "AT-009a: YAML rule should not be active before hot-reload, got: $AT009_BEFORE"
  else
    ok "AT-009a: before hot-reload — YAML rule not yet loaded"
  fi

  # Hot-reload: write a deny rule to config.yaml
  cat > "$AT009_DIR/config.yaml" <<'YAML'
rules:
  - name: at009-deny-write
    event: PreToolUse
    when:
      tool: [Write]
    decision: deny
    reason: "AT-009 hot-reload test"
YAML

  # Wait for debounce (200ms) + margin
  sleep 0.5

  # Confirm YAML rule fires after hot-reload (identified by its specific reason)
  AT009_AFTER=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT_AT009/hook" \
    -H 'Content-Type: application/json' \
    -d '{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{"file_path":"/tmp/x"}}' 2>/dev/null)
  if echo "$AT009_AFTER" | grep -q "AT-009 hot-reload test"; then
    ok "AT-009b: after hot-reload — YAML deny rule fires without restart"
  else
    fail "AT-009b: after hot-reload — expected YAML rule reason in body: $AT009_AFTER"
  fi
fi

kill $DAEMON_PID_AT009 2>/dev/null; wait $DAEMON_PID_AT009 2>/dev/null
trap 'rm -rf "$WORKDIR" "$AT007_DIR" "$AT008_DIR" "$AT009_DIR"' EXIT

echo ""
echo "=== CLI Utility Commands ==="

# AT-010: list command must write to stderr only (stdout must be empty)
AT010_STDOUT=$("$BIN" list 2>/dev/null || true)
if [ -z "$AT010_STDOUT" ]; then
  ok "AT-010a: list command stdout is empty (stdout-purity constraint)"
else
  fail "AT-010a: list command stdout not empty: $AT010_STDOUT"
fi
# AT-010b/c: expected strings appear on stderr
contains "AT-010b" "list command outputs 'Native Go rules' on stderr" "Native Go rules" "$BIN list 2>&1 1>/dev/null"
contains "AT-010c" "list command outputs 'Total:' on stderr"          "Total:"           "$BIN list 2>&1 1>/dev/null"

# AT-011: uninstall --help shows usage
contains "AT-011" "uninstall --help shows usage" "Remove claude-hooks" "$BIN uninstall --help"

# ── Summary ──────────────────────────────────────────────────────────────────

echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ ${#ERRORS[@]} -gt 0 ]; then
  echo "Failed:"
  for e in "${ERRORS[@]}"; do echo "  - $e"; done
  exit 1
fi
exit 0
