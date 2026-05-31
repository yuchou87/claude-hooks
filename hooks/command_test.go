package hooks_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

// captureStdout replaces os.Stdout with a pipe, calls f, returns captured output.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRunCommand_BadJSON_ExitsZero_StdoutEmpty(t *testing.T) {
	r, w, _ := os.Pipe()
	w.WriteString("{bad json")
	w.Close()

	var exitCode int
	out := captureStdout(t, func() {
		exitCode = hooks.RunCommand(r)
	})

	if exitCode != 0 {
		t.Errorf("want exit 0, got %d", exitCode)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("want empty stdout, got %q", out)
	}
}

func TestRunCommand_NoRuleMatch_ExitsZero_StdoutEmpty(t *testing.T) {
	// SessionEnd with no matching rule → nil → exit 0, stdout empty.
	raw := `{"hook_event_name":"SessionEnd","session_id":"s","transcript_path":"/t","cwd":"/"}`
	r, w, _ := os.Pipe()
	w.WriteString(raw)
	w.Close()

	var exitCode int
	out := captureStdout(t, func() {
		exitCode = hooks.RunCommand(r)
	})

	if exitCode != 0 {
		t.Errorf("want exit 0, got %d", exitCode)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("want empty stdout for nil output, got %q", out)
	}
}

func TestRunCommand_StdoutIsValidJSON(t *testing.T) {
	hooks.Register(hooks.Rule{
		Name:   "test-allow-read",
		Events: []string{"PreToolUse"},
		Run: func(in hooks.Input) *hooks.Output {
			if in.ToolName == "Read" {
				return hooks.Allow()
			}
			return nil
		},
	})

	raw := `{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Read","tool_input":{}}`
	r, w, _ := os.Pipe()
	w.WriteString(raw)
	w.Close()

	var exitCode int
	out := captureStdout(t, func() {
		exitCode = hooks.RunCommand(r)
	})

	if exitCode != 0 {
		t.Errorf("want exit 0 for allow, got %d", exitCode)
	}
	if !json.Valid([]byte(strings.TrimSpace(out))) {
		t.Errorf("stdout is not valid JSON: %q", out)
	}
}
