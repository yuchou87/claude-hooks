package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logMu   sync.Mutex
	logFile *os.File
)

type logEntry struct {
	Ts      string `json:"ts"`
	Event   string `json:"event"`
	Session string `json:"session,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Outcome string `json:"outcome"`
	Rule    string `json:"rule,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// LogInvocation writes a debug-level entry for every event received by Dispatch.
// Only fires when CLAUDE_HOOKS_DEBUG=1.
func LogInvocation(ev Input) {
	if os.Getenv("CLAUDE_HOOKS_DEBUG") != "1" {
		return
	}
	entry := map[string]any{
		"ts":      time.Now().UTC().Format(time.RFC3339),
		"level":   "debug",
		"event":   ev.HookEventName,
		"session": ev.SessionID,
		"tool":    ev.ToolName,
	}
	line, _ := json.Marshal(entry)
	line = append(line, '\n')
	logMu.Lock()
	defer logMu.Unlock()
	if f, err := openLogFile(); err == nil {
		f.Write(line) //nolint:errcheck
	}
}

// LogDecision writes a decision log entry. Only logs deny/block outcomes by default.
// Set CLAUDE_HOOKS_DEBUG=1 to log all outcomes.
func LogDecision(ev Input, out *Output, ruleName string) {
	outcome := "skip"
	if out != nil {
		if out.IsDeny() {
			outcome = "deny"
		} else if out.Continue != nil && !*out.Continue {
			outcome = "stop"
		} else {
			outcome = "allow"
		}
	}

	debug := os.Getenv("CLAUDE_HOOKS_DEBUG") == "1"
	if (outcome == "skip" || outcome == "allow") && !debug {
		return
	}

	entry := logEntry{
		Ts:      time.Now().UTC().Format(time.RFC3339),
		Event:   ev.HookEventName,
		Session: ev.SessionID,
		Tool:    ev.ToolName,
		Outcome: outcome,
		Rule:    ruleName,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return
	}
	line = append(line, '\n')

	logMu.Lock()
	defer logMu.Unlock()
	f, err := openLogFile()
	if err != nil {
		return
	}
	f.Write(line) //nolint:errcheck
}

func openLogFile() (*os.File, error) {
	if logFile != nil {
		return logFile, nil
	}
	dir := logDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "claude-hooks.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	logFile = f
	return f, nil
}

func logDir() string {
	if override := os.Getenv("CLAUDE_HOOKS_LOG_DIR"); override != "" {
		return override
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-hooks", "logs")
}

// LogError writes an error log entry (always written, regardless of debug mode).
func LogError(ev Input, msg string, err error) {
	entry := map[string]any{
		"ts":    time.Now().UTC().Format(time.RFC3339),
		"level": "error",
		"event": ev.HookEventName,
		"msg":   msg,
		"error": fmt.Sprintf("%v", err),
	}
	line, _ := json.Marshal(entry)
	line = append(line, '\n')

	logMu.Lock()
	defer logMu.Unlock()
	if f, err := openLogFile(); err == nil {
		f.Write(line) //nolint:errcheck
	}
}

// ResetLogFileForTest closes and resets the cached log file handle.
// Call from t.Cleanup in tests that change CLAUDE_HOOKS_LOG_DIR.
func ResetLogFileForTest() {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}
