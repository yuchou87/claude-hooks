package hooks_test

import (
	"testing"
	"time"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestNotifyCompletion_NonStop_IsNoOp(t *testing.T) {
	// Must not panic for non-Stop events
	hooks.NotifyCompletion(hooks.Input{HookEventName: "PreToolUse", Cwd: "/tmp"})
	hooks.NotifyCompletion(hooks.Input{HookEventName: "SessionEnd", Cwd: "/tmp"})
	hooks.NotifyCompletion(hooks.Input{HookEventName: "PostToolUse", Cwd: "/tmp"})
}

func TestNotifyCompletion_Stop_DoesNotBlock(t *testing.T) {
	// Caller must not be blocked — notification runs in background goroutine.
	start := time.Now()
	hooks.NotifyCompletion(hooks.Input{HookEventName: "Stop", Cwd: "/tmp/myproject"})
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("NotifyCompletion blocked for %v, want <100ms", elapsed)
	}
}

func TestNotifyCompletion_SubagentStop_DoesNotBlock(t *testing.T) {
	start := time.Now()
	hooks.NotifyCompletion(hooks.Input{HookEventName: "SubagentStop", Cwd: "/home/user/project"})
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("NotifyCompletion blocked for %v, want <100ms", elapsed)
	}
}
