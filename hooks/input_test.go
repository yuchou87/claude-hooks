package hooks_test

import (
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestParseInput_PreToolUse(t *testing.T) {
	raw := []byte(`{
		"hook_event_name": "PreToolUse",
		"session_id": "sess-1",
		"transcript_path": "/tmp/t.jsonl",
		"cwd": "/home/user",
		"permission_mode": "default",
		"tool_name": "Bash",
		"tool_input": {"command": "ls /tmp"},
		"tool_use_id": "tu-1"
	}`)
	in, err := hooks.ParseInput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.HookEventName != "PreToolUse" {
		t.Errorf("got %q, want PreToolUse", in.HookEventName)
	}
	if in.ToolName != "Bash" {
		t.Errorf("got %q, want Bash", in.ToolName)
	}
	cmd, _ := in.ToolInput["command"].(string)
	if cmd != "ls /tmp" {
		t.Errorf("got %q, want ls /tmp", cmd)
	}
}

func TestParseInput_UnknownFields_GoToExtra(t *testing.T) {
	raw := []byte(`{
		"hook_event_name": "SessionStart",
		"session_id": "s1",
		"transcript_path": "/tmp/t",
		"cwd": "/",
		"unknown_field_v99": "some_value"
	}`)
	in, err := hooks.ParseInput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.Extra["unknown_field_v99"] != "some_value" {
		t.Errorf("unknown field not in Extra: %v", in.Extra)
	}
}

func TestParseInput_InvalidJSON_ReturnsError(t *testing.T) {
	_, err := hooks.ParseInput([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseInput_EmptyBody_ReturnsError(t *testing.T) {
	_, err := hooks.ParseInput([]byte(``))
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}
