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

func TestAsk_JSON(t *testing.T) {
	out := hooks.Ask("needs review")
	b, err := json.Marshal(out.HookSpecific)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	if m["permissionDecision"] != "ask" {
		t.Errorf("want permissionDecision=ask, got %v", m)
	}
	if m["permissionDecisionReason"] != "needs review" {
		t.Errorf("want reason=needs review, got %v", m["permissionDecisionReason"])
	}
}

func TestAsk_IsAsk(t *testing.T) {
	if !hooks.Ask("x").IsAsk() {
		t.Error("Ask() output should report IsAsk()=true")
	}
	if hooks.Deny("x").IsAsk() {
		t.Error("Deny() output should report IsAsk()=false")
	}
	if hooks.Allow().IsAsk() {
		t.Error("Allow() output should report IsAsk()=false")
	}
}

func TestAsk_IsNotDeny(t *testing.T) {
	if hooks.Ask("x").IsDeny() {
		t.Error("Ask() output must not report IsDeny()=true")
	}
}

func TestAllowWithUpdatedInput_JSON(t *testing.T) {
	updates := map[string]any{"command": "echo modified"}
	out := hooks.AllowWithUpdatedInput(updates)
	if out == nil {
		t.Fatal("want non-nil output")
	}
	b, err := out.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	hs, _ := m["hookSpecificOutput"].(map[string]any)
	if hs["permissionDecision"] != "allow" {
		t.Errorf("want permissionDecision=allow, got %v", hs["permissionDecision"])
	}
	ui, _ := hs["updatedInput"].(map[string]any)
	if ui["command"] != "echo modified" {
		t.Errorf("want updatedInput.command='echo modified', got %v", ui)
	}
}

func TestAllowWithUpdatedInput_IsNotDeny(t *testing.T) {
	out := hooks.AllowWithUpdatedInput(map[string]any{"x": 1})
	if out.IsDeny() {
		t.Error("AllowWithUpdatedInput must not be deny")
	}
	if out.IsAsk() {
		t.Error("AllowWithUpdatedInput must not be ask")
	}
}

func TestAllowWithUpdatedInput_NilUpdates(t *testing.T) {
	out := hooks.AllowWithUpdatedInput(nil)
	b, err := out.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	hs, _ := m["hookSpecificOutput"].(map[string]any)
	if _, ok := hs["updatedInput"]; ok {
		t.Error("nil updates should produce no updatedInput field in JSON")
	}
}
