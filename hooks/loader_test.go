package hooks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestBuildDynamicRules_NoPaths_ReturnsEmpty(t *testing.T) {
	rules, err := hooks.BuildDynamicRules("", "")
	if err != nil {
		t.Fatalf("want nil error, got: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("want 0 rules, got %d", len(rules))
	}
}

func TestBuildDynamicRules_MissingFiles_ReturnsEmpty(t *testing.T) {
	rules, err := hooks.BuildDynamicRules("/nonexistent/config.yaml", "/nonexistent/scripts/")
	if err != nil {
		t.Fatalf("missing files must not error, got: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("want 0 rules, got %d", len(rules))
	}
}

func TestBuildDynamicRules_WithConfig_HasYAMLRules(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(`
rules:
  - name: yaml-rule
    event: PreToolUse
    when:
      tool: [Write]
    decision: deny
    reason: "yaml"
`), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := hooks.BuildDynamicRules(configPath, "")
	if err != nil {
		t.Fatalf("BuildDynamicRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rules))
	}
	if rules[0].Name != "yaml-rule" {
		t.Errorf("want name=yaml-rule, got %q", rules[0].Name)
	}
}

func TestBuildDynamicRules_WithScripts_HasScriptRules(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	dir := t.TempDir()
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "deny_write.js"), []byte(`
export const events = ["PreToolUse"];
export function decide(e) {
    return e.tool_name === "Edit" ? { permissionDecision: "deny", permissionDecisionReason: "script" } : null;
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := hooks.BuildDynamicRules("", scriptsDir)
	if err != nil {
		t.Fatalf("BuildDynamicRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("want 1 script rule, got %d", len(rules))
	}
	if rules[0].Name != "script:deny_write.js" {
		t.Errorf("want name=script:deny_write.js, got %q", rules[0].Name)
	}
}

func TestBuildDynamicRules_CombinesConfigAndScripts(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(configPath, []byte(`
rules:
  - name: yaml-rule
    event: PreToolUse
    when:
      tool: [Write]
    decision: deny
    reason: "yaml"
`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scriptsDir, "script.js"), []byte(`
export const events = ["PreToolUse"];
export function decide(e) { return null; }
`), 0644); err != nil {
		t.Fatal(err)
	}

	rules, err := hooks.BuildDynamicRules(configPath, scriptsDir)
	if err != nil {
		t.Fatalf("BuildDynamicRules: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("want 2 rules (1 yaml + 1 script), got %d", len(rules))
	} else {
		// YAML rules must come before script rules (ordering is a meaningful contract)
		if rules[0].Name != "yaml-rule" {
			t.Errorf("want rules[0].Name=yaml-rule, got %q", rules[0].Name)
		}
		if rules[1].Name != "script:script.js" {
			t.Errorf("want rules[1].Name=script:script.js, got %q", rules[1].Name)
		}
	}
}
