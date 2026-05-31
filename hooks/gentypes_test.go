package hooks

import (
	"strings"
	"testing"
)

func TestTypeScriptTypes_ContainsKeyInterfaces(t *testing.T) {
	ts := TypeScriptTypes()

	required := []string{
		"interface HookInput",
		"hook_event_name",
		"tool_name",
		"tool_input",
		"type HookOutput",
		"permissionDecision",
		"interface DenyOutput",
		"interface AllowOutput",
	}

	for _, want := range required {
		if !strings.Contains(ts, want) {
			t.Errorf("TypeScriptTypes() missing %q", want)
		}
	}
}

func TestTypeScriptTypes_BalancedBraces(t *testing.T) {
	ts := TypeScriptTypes()
	opens := strings.Count(ts, "{")
	closes := strings.Count(ts, "}")
	if opens != closes {
		t.Errorf("unbalanced braces: %d open, %d close", opens, closes)
	}
}

func TestTypeScriptTypes_NotEmpty(t *testing.T) {
	ts := TypeScriptTypes()
	if len(strings.TrimSpace(ts)) == 0 {
		t.Error("TypeScriptTypes() returned empty string")
	}
}
