package hooks_test

import (
	"encoding/json"
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestDeny_JSON(t *testing.T) {
	out := hooks.Deny("too dangerous")
	b, err := json.Marshal(out.HookSpecific)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["permissionDecision"] != "deny" {
		t.Errorf("want permissionDecision=deny, got %v", m)
	}
	if m["permissionDecisionReason"] != "too dangerous" {
		t.Errorf("want reason, got %v", m["permissionDecisionReason"])
	}
}

func TestAllow_JSON(t *testing.T) {
	out := hooks.Allow()
	b, _ := json.Marshal(out.HookSpecific)
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["permissionDecision"] != "allow" {
		t.Errorf("want allow, got %v", m)
	}
}

func TestDefer_IsNil(t *testing.T) {
	if hooks.Defer() != nil {
		t.Error("Defer() must return nil (no opinion)")
	}
}

func TestOutput_IsDeny(t *testing.T) {
	if !hooks.Deny("x").IsDeny() {
		t.Error("Deny output should be deny")
	}
	if hooks.Allow().IsDeny() {
		t.Error("Allow output should not be deny")
	}
}

func TestBlock_NotDeny_ExitsZero(t *testing.T) {
	out := hooks.Block("post-tool blocked")
	if out.IsDeny() {
		t.Error("Block output must not be deny (it's a PostToolUse decision, not PreToolUse)")
	}
	b, err := out.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	hs, _ := m["hookSpecificOutput"].(map[string]any)
	if hs["decision"] != "block" {
		t.Errorf("want decision=block, got %v", hs)
	}
}

func TestGlobalStop_Continue(t *testing.T) {
	out := hooks.GlobalStop("emergency stop")
	if out.Continue == nil || *out.Continue != false {
		t.Error("GlobalStop must set Continue=false")
	}
	if out.StopReason != "emergency stop" {
		t.Errorf("got %q", out.StopReason)
	}
}
