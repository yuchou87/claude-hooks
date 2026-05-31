# HTTP Hook Verify — Spike Results

**Date tested:** 2026-05-30
**Claude Code version:** v2.1.145
**Tester:** yuchou

---

## Setup

Server started with:
```
cd spike/http-hook-verify
go run . >> /tmp/spike-server.log 2>&1 &
```

Hook registered with:
```
./register.sh add
```

---

## Round 1: deny (PreToolUse)

**Tool used to trigger PreToolUse:**
New Claude Code session → asked Claude to run `ls /tmp`

**Server log output:**
```
22:40:10 Spike server starting  addr=127.0.0.1:8787  mode=deny
22:40:11 RECV  event=PreToolUse  tool=Bash  recv_at=2026-05-30T22:40:11.701365+08:00
22:40:16 SEND  behavior=deny   send_at=2026-05-30T22:40:16.702482+08:00  elapsed=5.001s
```

**Elapsed time:** 5.001s

**Claude behavior:**
- [x] Tool was refused by Claude
- [x] Claude showed "Denied by spike server" reason message
- [ ] Claude did NOT wait — connection dropped immediately (FAILURE: assumption 1 failed)

**Assumption 1 (connection hold):** ✅ CONFIRMED
**Assumption 2 — deny control:** ✅ CONFIRMED

**Note:** `PermissionRequest` hook was tested first and FAILED — the tool still executed despite returning `behavior:deny`. Only `PreToolUse` with `permissionDecision:deny` reliably blocks tool execution.

---

## Round 2: allow (PreToolUse)

Changed `const MODE = "deny"` → `"allow"` in main.go, rebuilt, restarted server.

**Server log output:**
```
22:46:02 Spike server starting  addr=127.0.0.1:8787  mode=allow
22:46:35 RECV  event=PreToolUse  tool=Bash  recv_at=2026-05-30T22:46:35.404658+08:00
22:46:40 SEND  behavior=allow  send_at=2026-05-30T22:46:40.405448+08:00  elapsed=5.001s
```

**Elapsed time:** 5.001s

**Claude behavior:**
- [x] Tool executed successfully
- [x] No permission popup appeared
- [ ] Tool was refused (unexpected)

**Assumption 2 — allow control:** ✅ CONFIRMED

---

## Teardown

```bash
./register.sh remove
lsof -ti:8787 | xargs kill -9
```

---

## Conclusion

**Both assumptions hold → proceed with http-mode design**

**Critical finding:** Must use `PreToolUse` hook, NOT `PermissionRequest`.

- `PermissionRequest`: fires when permission dialog appears — `decision.behavior:deny` does NOT prevent tool execution
- `PreToolUse`: fires before tool call — `permissionDecision:deny` reliably blocks all tool types (Bash, Edit, Read, etc.)

### Response format (PreToolUse)

**Deny:**
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "deny",
    "permissionDecisionReason": "reason string"
  }
}
```

**Allow:**
```json
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow"
  }
}
```

### Notes

- deny mode blocks ALL tool types (Bash, Edit, Read) — the entire Claude Code agent is paused
- connection hold is rock-solid: consistently 5.001s across all test rounds
- fail-open works: when server is down, HTTP hook fails and Claude Code allows the tool to proceed normally
- `go run .` spawns a child compiled binary — kill via `lsof -ti:8787 | xargs kill -9`, not just the parent go process
