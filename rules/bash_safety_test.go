package rules_test

import (
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
	_ "github.com/yuchou87/claude-hooks/rules" // trigger init()
)

func TestBashSafety_BlocksRmRf(t *testing.T) {
	raw := []byte(`{
		"hook_event_name": "PreToolUse",
		"session_id": "s", "transcript_path": "/t", "cwd": "/",
		"tool_name": "Bash",
		"tool_input": {"command": "rm -rf /"}
	}`)
	out := hooks.Dispatch(raw)
	if out == nil || !out.IsDeny() {
		t.Errorf("rm -rf / must be denied, got %+v", out)
	}
}

func TestBashSafety_BlocksRmRfHome(t *testing.T) {
	raw := []byte(`{
		"hook_event_name": "PreToolUse",
		"session_id": "s", "transcript_path": "/t", "cwd": "/",
		"tool_name": "Bash",
		"tool_input": {"command": "rm -rf ~"}
	}`)
	out := hooks.Dispatch(raw)
	if out == nil || !out.IsDeny() {
		t.Errorf("rm -rf ~ must be denied, got %+v", out)
	}
}

func TestBashSafety_AllowsRmRfTmpBuild(t *testing.T) {
	raw := []byte(`{
		"hook_event_name": "PreToolUse",
		"session_id": "s", "transcript_path": "/t", "cwd": "/",
		"tool_name": "Bash",
		"tool_input": {"command": "rm -rf /tmp/build"}
	}`)
	out := hooks.Dispatch(raw)
	if out != nil && out.IsDeny() {
		t.Errorf("rm -rf /tmp/build must NOT be denied (legitimate cleanup), got %+v", out)
	}
}

func TestBashSafety_AllowsSafeCommand(t *testing.T) {
	raw := []byte(`{
		"hook_event_name": "PreToolUse",
		"session_id": "s", "transcript_path": "/t", "cwd": "/",
		"tool_name": "Bash",
		"tool_input": {"command": "ls /tmp"}
	}`)
	out := hooks.Dispatch(raw)
	if out != nil && out.IsDeny() {
		t.Errorf("ls /tmp must not be denied, got %+v", out)
	}
}

func TestBashSafety_NonBashEvent_NotTriggered(t *testing.T) {
	raw := []byte(`{
		"hook_event_name": "PreToolUse",
		"session_id": "s", "transcript_path": "/t", "cwd": "/",
		"tool_name": "Read",
		"tool_input": {"file_path": "/etc/passwd"}
	}`)
	out := hooks.Dispatch(raw)
	if out != nil && out.IsDeny() {
		t.Errorf("bash_safety must not deny Read tool, got %+v", out)
	}
}
