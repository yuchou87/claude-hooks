# claude-hooks Acceptance Test Suite

> **Usage:** Run `./scripts/acceptance.sh` to execute all scenarios.
> Update this document whenever a new command or feature is added — in the same commit.

---

## How to Run

```bash
# Full acceptance run (builds binary + executes all scenarios)
./scripts/acceptance.sh

# Verbose output (shows test headers + output on failure)
./scripts/acceptance.sh --verbose
```

---

## Acceptance Scenarios

### Core / CLI

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-001 | `--version` flag | `claude-hooks --version` | output contains `claude-hooks` | ✅ PASS |
| AT-002 | `--help` lists `run` subcommand | `claude-hooks --help` | output contains `run` | ✅ PASS |

### Command Mode

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-003a | Bad JSON → fail-open, exit 0 | `echo '{bad}' \| claude-hooks run` | exit 0 | ✅ PASS |
| AT-003b | Bad JSON → stdout empty (stdout-purity) | `echo '{bad}' \| claude-hooks run` | stdout is empty | ✅ PASS |
| AT-004a | `rm -rf /` denied → exit 2 | `PreToolUse Bash rm -rf /` payload | exit 2 | ✅ PASS |
| AT-004b | `rm -rf /` denied → deny JSON in stdout | `PreToolUse Bash rm -rf /` payload | stdout contains `permissionDecision` | ✅ PASS |

### Dynamic Rules (YAML + Scripts)

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-007 | HTTP server loads YAML deny rule | `POST /hook` Write tool with config having deny-write rule | body contains `permissionDecision`, status 200 | ✅ PASS |
| AT-008 | HTTP server loads JS script deny rule | `POST /hook` Write tool with script returning deny | body contains `permissionDecision`, status 200 | ✅ PASS |

---

> **Adding a new scenario:**
> 1. Add a row to the table above with a new `AT-NNN` ID.
> 2. Add a matching `run`/`contains`/`exits_with` call in `scripts/acceptance.sh`.
> 3. Include both changes in the same commit as the feature.
