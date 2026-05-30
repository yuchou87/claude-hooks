#!/usr/bin/env bash
# register.sh — add or remove the spike http hook from ~/.claude/settings.json
# Requires: jq
#
# Usage:
#   ./register.sh add     — register PermissionRequest http hook
#   ./register.sh remove  — remove the spike hook entry

set -euo pipefail

SETTINGS="$HOME/.claude/settings.json"
HOOK_URL="http://127.0.0.1:8787/hook"

check_jq() {
  if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required. Install with: brew install jq"
    exit 1
  fi
}

add() {
  check_jq
  if [ ! -f "$SETTINGS" ]; then
    echo '{}' > "$SETTINGS"
  fi

  jq --arg url "$HOOK_URL" '
    if ((.hooks.PermissionRequest // []) | map(.hooks[]?.url) | index($url)) != null then
      .
    else
      .hooks.PermissionRequest = ((.hooks.PermissionRequest // []) + [{
        "hooks": [{"type": "http", "url": $url}]
      }])
    end
  ' "$SETTINGS" > "$SETTINGS.tmp" && mv "$SETTINGS.tmp" "$SETTINGS"

  echo "✅ Added spike hook: PermissionRequest → $HOOK_URL"
  echo "   Start server: cd spike/http-hook-verify && go run ."
}

remove() {
  check_jq
  if [ ! -f "$SETTINGS" ]; then
    echo "No settings.json found — nothing to remove."
    return
  fi

  jq --arg url "$HOOK_URL" '
    if .hooks.PermissionRequest then
      .hooks.PermissionRequest = [
        .hooks.PermissionRequest[] |
        select(.hooks | map(select(.url == $url)) | length == 0)
      ] |
      if (.hooks.PermissionRequest | length) == 0 then del(.hooks.PermissionRequest) else . end
    else . end
  ' "$SETTINGS" > "$SETTINGS.tmp" && mv "$SETTINGS.tmp" "$SETTINGS"

  echo "✅ Removed spike hook entry (matched url: $HOOK_URL)"
}

case "${1:-}" in
  add)    add ;;
  remove) remove ;;
  *)
    echo "Usage: $0 add | remove"
    exit 1
    ;;
esac
