# HTTP Hook Verify — Spike Results

**Date tested:** YYYY-MM-DD
**Claude Code version:** (run `claude --version`)
**Tester:** (your name)

---

## Setup

Server started with:
```
cd spike/http-hook-verify
go run .
```

Hook registered with:
```
./register.sh add
```

---

## Round 1: deny

**Tool used to trigger PermissionRequest:**
(e.g., `run ls /tmp` in a Claude session with default permissions)

**Server log output:**
```
(paste RECV and SEND log lines here)
```

**Elapsed time (send_ts − recv_ts):**
(e.g., `5.002s`)

**Claude behavior:**
- [ ] Tool was refused by Claude
- [ ] Claude showed an error / denial message
- [ ] Claude did NOT wait (connected dropped immediately)

**Assumption 1 (connection hold):** ✅ CONFIRMED / ❌ FAILED
**Assumption 2 — deny control:** ✅ CONFIRMED / ❌ FAILED

---

## Round 2: allow

Changed `const MODE = "deny"` → `"allow"` in main.go, rebuilt, restarted server.

**Server log output:**
```
(paste RECV and SEND log lines here)
```

**Elapsed time:**

**Claude behavior:**
- [ ] Tool executed successfully
- [ ] No permission popup appeared
- [ ] Tool was refused (unexpected)

**Assumption 2 — allow control:** ✅ CONFIRMED / ❌ FAILED

---

## Conclusion

**Both assumptions hold → proceed with http-mode design**

OR

**Assumption(s) failed → fall back to command + blocking curl (masko pattern)**

### Notes
(any unexpected behavior, edge cases, or version-specific observations)
