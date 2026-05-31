package hooks

import (
	"io"
	"os"
)

// RunCommand is the command-mode adapter. It reads a hook event from stdin,
// dispatches it, and writes the decision JSON to stdout.
//
// Returns 0 (allow/nil/error) or 2 (deny).
// CRITICAL: stdout must contain only valid JSON or be empty — never log here.
func RunCommand(stdin io.Reader) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		os.Stderr.WriteString("claude-hooks: failed to read stdin\n")
		return 0
	}

	out := Dispatch(raw)
	if out == nil {
		return 0 // no decision → empty stdout, exit 0
	}

	b, err := out.JSON()
	if err != nil {
		os.Stderr.WriteString("claude-hooks: failed to encode output\n")
		return 0
	}

	os.Stdout.Write(b)
	os.Stdout.Write([]byte("\n"))

	if out.IsDeny() {
		return 2
	}
	return 0
}
