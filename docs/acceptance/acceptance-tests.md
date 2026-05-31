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

### HTTP Daemon

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-005 | HTTP local deny rule fires without dialog | `POST /hook` Bash `rm -rf /` payload | body contains `permissionDecision`, status 200 | ✅ PASS |
| AT-006 | HTTP SessionEnd → empty 200 (no dialog) | `POST /hook` SessionEnd payload | status 200, empty body | ✅ PASS |

### Dynamic Rules (YAML + Scripts)

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-007 | HTTP server loads YAML deny rule | `POST /hook` Write tool with config having deny-write rule | body contains `permissionDecision`, status 200 | ✅ PASS |
| AT-008 | HTTP server loads JS script deny rule | `POST /hook` Write tool with script returning deny | body contains `permissionDecision`, status 200 | ✅ PASS |
| AT-009 | Hot-reload: write YAML rule while server running, new rule fires | mutate config.yaml, wait 400ms, `POST /hook` | body contains `permissionDecision` after reload | ✅ PASS |

### CLI Utility Commands

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-010 | `list` enumerates native Go rules | `claude-hooks list` | stderr contains `Native Go rules` and `Total:` | ✅ PASS |
| AT-011 | `uninstall --help` shows usage | `claude-hooks uninstall --help` | output contains `Remove claude-hooks` | ✅ PASS |
| AT-012 | `validate` passes with valid YAML | `claude-hooks validate --config <valid.yaml>` | exit 0 | ✅ PASS |
| AT-012a | `validate` stdout is empty on success (stdout-purity) | `claude-hooks validate --config <valid.yaml>` | stdout is empty | ✅ PASS |
| AT-012b | `validate` reports all checks passed | `claude-hooks validate --config <valid.yaml>` | stderr contains `All checks passed` | ✅ PASS |
| AT-012c | `validate` exits 1 on invalid YAML (empty rule name) | `claude-hooks validate --config <bad.yaml>` | exit 1 | ✅ PASS |

### Test Command

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-013a | `test` dispatches payload through rules | `echo <PreToolUse rm -rf /> \| claude-hooks test` | exit 0 | ✅ PASS |
| AT-013b | `test` outputs decision JSON | `echo <PreToolUse rm -rf /> \| claude-hooks test` | stdout contains `permissionDecision` | ✅ PASS |

### Doctor Command

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-014 | `doctor --help` shows usage | `claude-hooks doctor --help` | output contains `Diagnose` | ✅ PASS |

### gen-types Command

| ID | Scenario | Command | Expected | Status |
|----|----------|---------|----------|--------|
| AT-015 | `gen-types` outputs TypeScript interfaces | `claude-hooks gen-types` | stdout contains `HookInput` and `permissionDecision` | ✅ PASS |

---

> **Adding a new scenario:**
> 1. Add a row to the table above with a new `AT-NNN` ID.
> 2. Add a matching `run`/`contains`/`exits_with` call in `scripts/acceptance.sh`.
> 3. Include both changes in the same commit as the feature.
