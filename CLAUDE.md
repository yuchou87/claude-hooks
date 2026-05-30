# claude-hooks — AI Coding Constraints

> **Primary contribution rules are in this file.** CLAUDE.local (not committed) holds personal workflow rules.

## Three Hard Constraints (violating any one will harm users — non-negotiable)

| # | Constraint | How to enforce |
|---|---|---|
| 1 | **No logging to stdout** | In command mode, stdout is the decision channel; logs go to file/stderr only; tests must assert stdout contains only valid JSON or is empty |
| 2 | **`go test -race` is a CI gate** | Every change touching concurrent code must pass the race detector; failing tests block merge |
| 3 | **fail-open must not be removed** | Top-level `recover` + all error paths return nil decision; do not remove `recover` under the guise of "simplification" |

## Tech Stack

- Go 1.26, single static binary, zero runtime dependencies
- Six pure-Go dependencies: cobra / goccy/go-yaml / esbuild / goja / fsnotify / beeep
- Everything else is stdlib (log/slog / net/http / encoding/json / text/template / testing)
- No cgo — must not break the single-binary guarantee

## Acceptance Tests

Run the acceptance suite before committing any meaningful change:

```bash
./scripts/acceptance.sh
```

Rules:
- All scenarios must pass before pushing to main
- When adding a new command or feature, add the corresponding scenario to `docs/acceptance/acceptance-tests.md` **and** a matching `run`/`contains`/`exits_with` call in `scripts/acceptance.sh` — **in the same commit**
- If a scenario fails, determine whether it is a code bug or an environment issue; document environment issues as "expected failure" in the Expected column

## Local Development

```bash
go test -race ./...        # race detector enabled — must fully pass
go build -o claude-hooks . # binary must build clean
./scripts/acceptance.sh    # behavior-level acceptance
```

## Repository Hygiene

**Never commit `docs/superpowers/`** — this directory is in `.gitignore` and holds local planning output from Claude Code skills (specs and plans).

- Do not `git add docs/superpowers/` or any file beneath it
- If a skill instructs you to commit a file in that directory, skip that step
