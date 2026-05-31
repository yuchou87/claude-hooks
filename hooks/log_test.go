package hooks_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestLogInvocation_WritesWhenDebugEnabled(t *testing.T) {
	hooks.ResetLogFileForTest() // reset any handle cached by earlier tests
	dir := t.TempDir()
	t.Setenv("CLAUDE_HOOKS_DEBUG", "1")
	t.Setenv("CLAUDE_HOOKS_LOG_DIR", dir)
	t.Cleanup(hooks.ResetLogFileForTest)

	ev := hooks.Input{HookEventName: "PreToolUse", SessionID: "s1", ToolName: "Bash"}
	hooks.LogInvocation(ev)

	data, err := os.ReadFile(filepath.Join(dir, "claude-hooks.jsonl"))
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `"level":"debug"`) {
		t.Errorf("want level=debug in log, got: %s", content)
	}
	if !strings.Contains(content, `"event":"PreToolUse"`) {
		t.Errorf("want event=PreToolUse in log, got: %s", content)
	}
}

func TestLogInvocation_SilentWhenDebugDisabled(t *testing.T) {
	hooks.ResetLogFileForTest() // reset any handle cached by earlier tests
	dir := t.TempDir()
	t.Setenv("CLAUDE_HOOKS_DEBUG", "0")
	t.Setenv("CLAUDE_HOOKS_LOG_DIR", dir)
	t.Cleanup(hooks.ResetLogFileForTest)

	ev := hooks.Input{HookEventName: "PreToolUse", ToolName: "Bash"}
	hooks.LogInvocation(ev)

	logPath := filepath.Join(dir, "claude-hooks.jsonl")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		data, _ := os.ReadFile(logPath)
		t.Errorf("log file must not exist when CLAUDE_HOOKS_DEBUG=0, got content: %s", data)
	}
}
