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
| AT-001 | `--version` flag | `claude-hooks --version` | output contains `claude-hooks` | 🔲 PENDING |
| AT-002 | `--help` lists `run` subcommand | `claude-hooks --help` | output contains `run` | 🔲 PENDING |

---

> **Adding a new scenario:**
> 1. Add a row to the table above with a new `AT-NNN` ID.
> 2. Add a matching `run`/`contains`/`exits_with` call in `scripts/acceptance.sh`.
> 3. Include both changes in the same commit as the feature.
