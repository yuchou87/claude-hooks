package hooks

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Approver handles interactive approval of hook events via a dialog.
// DialogFn and Timeout are exported so tests can inject alternatives.
type Approver struct {
	// DialogFn is called to show the approval UI. Returns (approved, err).
	// err non-nil means the dialog could not run → fail-open (nil output).
	// err nil + approved false means the user rejected.
	DialogFn func(ctx context.Context, ev Input) (bool, error)

	// Timeout is how long to wait for a user response. Default 55s.
	Timeout time.Duration
}

// NewApprover returns an Approver that calls the real macOS dialog.
func NewApprover() *Approver {
	return &Approver{
		DialogFn: macOSDialog,
		Timeout:  55 * time.Second,
	}
}

// Handle shows the approval dialog and returns a decision.
// Returns nil (defer/fail-open) on timeout, context cancellation, or dialog error.
func (a *Approver) Handle(ctx context.Context, ev Input) *Output {
	timeout := a.Timeout
	if timeout <= 0 {
		timeout = 55 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		approved bool
		err      error
	}
	ch := make(chan result, 1)
	go func() {
		approved, err := a.DialogFn(ctx, ev)
		ch <- result{approved, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return nil // dialog error → fail-open
		}
		if r.approved {
			return Allow()
		}
		return Deny("rejected via approval dialog")
	case <-ctx.Done():
		return nil // timeout or cancellation → defer (fail-open)
	}
}

// macOSDialog shows a blocking osascript modal dialog.
// Returns (false, nil) when the user dismisses/cancels.
// Returns (false, err) when osascript cannot run (fail-open).
func macOSDialog(ctx context.Context, ev Input) (bool, error) {
	prompt := formatApprovalPrompt(ev)
	script := fmt.Sprintf(
		`tell application "System Events"
    display dialog %q buttons {"拒绝", "批准"} default button "拒绝" with title "Claude Code 工具审批" giving up after 54
end tell`, prompt)

	out, err := exec.CommandContext(ctx, "osascript", "-e", script).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 1 {
				// osascript exit 1 = user cancelled / Esc / gave up → reject, not an error
				return false, nil
			}
			// Other non-zero exit (script syntax error, signal, etc.) → fail-open
			return false, err
		}
		// osascript not found or other execution error → fail-open
		return false, err
	}
	result := string(out)
	if strings.Contains(result, "gave up:true") {
		return false, nil // OS-level dialog timeout → reject
	}
	return strings.Contains(result, "button returned:批准"), nil
}

func formatApprovalPrompt(ev Input) string {
	var detail string
	switch ev.ToolName {
	case "Bash":
		cmd, _ := ev.ToolInput["command"].(string)
		if len(cmd) > 120 {
			cmd = cmd[:117] + "..."
		}
		detail = "命令：" + cmd
	case "Write", "Edit":
		fp, _ := ev.ToolInput["file_path"].(string)
		detail = "文件：" + fp
	default:
		detail = fmt.Sprintf("输入：%v", ev.ToolInput)
		if len(detail) > 120 {
			detail = detail[:117] + "..."
		}
	}
	cwd := ev.Cwd
	if len(cwd) > 60 {
		cwd = "..." + cwd[len(cwd)-57:]
	}
	return fmt.Sprintf("工具：%s\n%s\n目录：%s", ev.ToolName, detail, cwd)
}
