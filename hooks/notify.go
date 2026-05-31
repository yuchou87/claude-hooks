package hooks

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// NotifyCompletion fires a macOS notification banner when a session completes.
// Only fires for Stop and SubagentStop events. No-op on non-darwin platforms.
// Runs in a background goroutine — never blocks the hook response path.
func NotifyCompletion(ev Input) {
	if runtime.GOOS != "darwin" {
		return
	}
	if ev.HookEventName != "Stop" && ev.HookEventName != "SubagentStop" {
		return
	}

	dir := ev.Cwd
	if dir == "" {
		dir = "session"
	}
	msg := "✅ 完成 · " + filepath.Base(dir)
	script := fmt.Sprintf(`display notification %q with title "Claude Code"`, msg)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Fire-and-forget: failure is silently ignored to keep the hook response
		// path clean.
		exec.CommandContext(ctx, "osascript", "-e", script).Run() //nolint:errcheck
	}()
}
