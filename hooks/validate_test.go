package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDynamicRules_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(config, []byte(`rules:
  - name: deny-rm
    event: PreToolUse
    when:
      tool: [Bash]
    decision: deny
    reason: no rm
`), 0600); err != nil {
		t.Fatal(err)
	}

	checks, ok := ValidateDynamicRules(config, "")
	if !ok {
		for _, c := range checks {
			if !c.OK {
				t.Errorf("unexpected failure: %s — %s", c.Label, c.Detail)
			}
		}
	}
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
}

func TestValidateDynamicRules_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config.yaml")
	// Rule with empty name — should fail validation
	if err := os.WriteFile(config, []byte(`rules:
  - name: ""
    event: PreToolUse
    decision: deny
`), 0600); err != nil {
		t.Fatal(err)
	}

	checks, ok := ValidateDynamicRules(config, "")
	if ok {
		t.Error("expected ok=false for rule with empty name")
	}
	found := false
	for _, c := range checks {
		if !c.OK {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one failing check")
	}
}

func TestValidateDynamicRules_MissingFile(t *testing.T) {
	checks, ok := ValidateDynamicRules("/nonexistent/config.yaml", "")
	// Missing file is treated as empty config — valid
	if !ok {
		t.Errorf("missing file should be valid, got checks: %v", checks)
	}
}

func TestValidateDynamicRules_ValidScript(t *testing.T) {
	dir := t.TempDir()
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write a minimal valid JS script (ES module syntax with required events export)
	if err := os.WriteFile(filepath.Join(scriptsDir, "deny.js"), []byte(`
export const events = ["PreToolUse"];
export function decide(e) { return null; }
`), 0644); err != nil {
		t.Fatal(err)
	}

	checks, ok := ValidateDynamicRules("", scriptsDir)
	if !ok {
		for _, c := range checks {
			if !c.OK {
				t.Errorf("unexpected failure: %s — %s", c.Label, c.Detail)
			}
		}
	}
}

func TestValidateDynamicRules_BothEmpty(t *testing.T) {
	checks, ok := ValidateDynamicRules("", "")
	if !ok {
		t.Errorf("both-empty should be valid, got: %v", checks)
	}
	// Should have exactly one "nothing specified" check
	if len(checks) != 1 {
		t.Errorf("expected 1 check for both-empty, got %d", len(checks))
	}
}

func TestValidateDynamicRules_InvalidScript(t *testing.T) {
	dir := t.TempDir()
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Script missing the required 'events' export — should fail validation
	if err := os.WriteFile(filepath.Join(scriptsDir, "bad.js"), []byte(`
export function decide(input) { return null; }
`), 0644); err != nil {
		t.Fatal(err)
	}

	checks, ok := ValidateDynamicRules("", scriptsDir)
	if ok {
		t.Error("expected ok=false for script missing 'events' export")
	}
	found := false
	for _, c := range checks {
		if !c.OK {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one failing check")
	}
}
