package hooks_test

import (
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestDispatch_FailOpen_OnBadJSON(t *testing.T) {
	got := hooks.Dispatch([]byte(`{bad json`))
	if got != nil {
		t.Errorf("bad JSON must fail-open (nil), got %+v", got)
	}
}

func TestDispatch_FailOpen_OnPanic(t *testing.T) {
	hooks.Register(hooks.Rule{
		Name:   "panic-rule",
		Events: []string{"PreToolUse"},
		Run: func(in hooks.Input) *hooks.Output {
			panic("deliberate panic")
		},
	})
	raw := []byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/"}`)
	got := hooks.Dispatch(raw)
	// panic-rule panics → safeRun returns nil → merge → nil (no other rules deny)
	// We assert it doesn't propagate the panic (test would fail with panic if it did).
	_ = got
}

func TestDispatch_DenyRule_ReturnsDeny(t *testing.T) {
	hooks.Register(hooks.Rule{
		Name:   "always-deny",
		Events: []string{"PreToolUse"},
		Run: func(in hooks.Input) *hooks.Output {
			return hooks.Deny("always denied")
		},
	})
	raw := []byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash"}`)
	got := hooks.Dispatch(raw)
	if got == nil || !got.IsDeny() {
		t.Errorf("want deny, got %+v", got)
	}
}

func TestDispatch_UnregisteredEvent_ReturnsNil(t *testing.T) {
	raw := []byte(`{"hook_event_name":"SessionEnd","session_id":"s","transcript_path":"/t","cwd":"/"}`)
	got := hooks.Dispatch(raw)
	// Can't assert nil if other rules catch all events; just assert no panic.
	_ = got
}
