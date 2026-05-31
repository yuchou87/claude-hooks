package hooks_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestScriptEngine_DenyScript_ReturnsDeny(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	script := `
export const events = ["PreToolUse"];
export function decide(e) {
    if (e.tool_name === "Write") {
        return { permissionDecision: "deny", permissionDecisionReason: "script deny" };
    }
    return null;
}
`
	path := writeScriptTemp(t, "deny.js", script)
	eng, err := hooks.NewScriptEngine(path)
	if err != nil {
		t.Fatalf("NewScriptEngine: %v", err)
	}

	rule := eng.AsRule()
	if rule.Name == "" {
		t.Fatal("rule name must not be empty")
	}

	hooks.StoreDynamic([]hooks.Rule{rule})

	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{}}`))
	if out == nil || !out.IsDeny() {
		t.Errorf("Write must be denied by script, got %+v", out)
	}

	out = hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Read","tool_input":{}}`))
	if out != nil {
		t.Errorf("Read must not be denied, got %+v", out)
	}
}

func TestScriptEngine_AllowScript_ReturnsAllow(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	script := `
export const events = ["PreToolUse"];
export function decide(e) {
    return { permissionDecision: "allow" };
}
`
	path := writeScriptTemp(t, "allow.js", script)
	eng, _ := hooks.NewScriptEngine(path)
	hooks.StoreDynamic([]hooks.Rule{eng.AsRule()})

	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash","tool_input":{}}`))
	if out == nil || out.IsDeny() {
		t.Errorf("must return Allow (non-nil, non-deny), got %+v", out)
	}
}

func TestScriptEngine_NullReturn_ReturnsNil(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	script := `
export const events = ["PreToolUse"];
export function decide(e) { return null; }
`
	path := writeScriptTemp(t, "nil.js", script)
	eng, _ := hooks.NewScriptEngine(path)
	hooks.StoreDynamic([]hooks.Rule{eng.AsRule()})

	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash","tool_input":{}}`))
	if out != nil {
		t.Errorf("null return must produce nil, got %+v", out)
	}
}

func TestScriptEngine_Throw_ReturnsNil(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	script := `
export const events = ["PreToolUse"];
export function decide(e) { throw new Error("boom"); }
`
	path := writeScriptTemp(t, "throw.js", script)
	eng, _ := hooks.NewScriptEngine(path)
	hooks.StoreDynamic([]hooks.Rule{eng.AsRule()})

	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash","tool_input":{}}`))
	if out != nil {
		t.Errorf("throw must fail-open (nil), got %+v", out)
	}
}

func TestScriptEngine_Timeout_ReturnsNil(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	// Script spins forever — must be interrupted in ≤100ms
	script := `
export const events = ["PreToolUse"];
export function decide(e) { while (true) {} }
`
	path := writeScriptTemp(t, "timeout.js", script)
	eng, _ := hooks.NewScriptEngine(path)
	hooks.StoreDynamic([]hooks.Rule{eng.AsRule()})

	start := time.Now()
	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash","tool_input":{}}`))
	elapsed := time.Since(start)

	if out != nil {
		t.Errorf("timeout must fail-open (nil), got %+v", out)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout must fire within 500ms, took %v", elapsed)
	}
}

func TestScriptEngine_Events_Extracted(t *testing.T) {
	script := `
export const events = ["PreToolUse", "PostToolUse"];
export function decide(e) { return null; }
`
	path := writeScriptTemp(t, "events.js", script)
	eng, err := hooks.NewScriptEngine(path)
	if err != nil {
		t.Fatalf("NewScriptEngine: %v", err)
	}
	rule := eng.AsRule()
	if len(rule.Events) != 2 {
		t.Errorf("want 2 events, got %v", rule.Events)
	}
	if rule.Events[0] != "PreToolUse" || rule.Events[1] != "PostToolUse" {
		t.Errorf("wrong events: %v", rule.Events)
	}
}

func TestScriptEngine_TypeScript_Works(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	// TypeScript syntax: typed parameter
	script := `
export const events = ["PreToolUse"] as const;
export function decide(e: { tool_name: string }): { permissionDecision: string } | null {
    if (e.tool_name === "Write") {
        return { permissionDecision: "deny", permissionDecisionReason: "ts deny" };
    }
    return null;
}
`
	path := writeScriptTemp(t, "typed.ts", script)
	eng, err := hooks.NewScriptEngine(path)
	if err != nil {
		t.Fatalf("NewScriptEngine with TypeScript: %v", err)
	}
	hooks.StoreDynamic([]hooks.Rule{eng.AsRule()})

	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{}}`))
	if out == nil || !out.IsDeny() {
		t.Errorf("TS script must deny Write, got %+v", out)
	}
}

// writeScriptTemp writes a script to a temp dir and returns the path.
func writeScriptTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
