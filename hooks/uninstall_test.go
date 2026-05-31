package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUninstallFromFile_RemovesEntry(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"type":                "command",
					"command":             "/usr/local/bin/claude-hooks run",
					"_claudeHooksVersion": "1.0.0",
				},
			},
		},
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(settingsPath, data, 0600)

	err := UninstallFromFile(settingsPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ = os.ReadFile(settingsPath)
	var result map[string]any
	json.Unmarshal(data, &result)

	hm, _ := result["hooks"].(map[string]any)
	if _, exists := hm["PreToolUse"]; exists {
		t.Errorf("expected PreToolUse key to be removed entirely, but it still exists")
	}
}

func TestUninstallFromFile_NoFile(t *testing.T) {
	err := UninstallFromFile("/nonexistent/path/settings.json", false)
	if err != nil {
		t.Errorf("expected nil for missing file, got %v", err)
	}
}

func TestUninstallFromFile_DryRun(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"_claudeHooksVersion": "1.0.0",
					"type":               "command",
					"command":            "/bin/claude-hooks run",
				},
			},
		},
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(settingsPath, data, 0600)

	err := UninstallFromFile(settingsPath, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should be unchanged after dry-run
	after, _ := os.ReadFile(settingsPath)
	if string(after) != string(data) {
		t.Error("dry-run should not modify file")
	}
}

func TestUninstallFromFile_KeepsOtherEntries(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				// Our entry — should be removed
				map[string]any{
					"_claudeHooksVersion": "1.0.0",
					"type":               "command",
					"command":            "/bin/claude-hooks run",
				},
				// Third-party entry — should stay
				map[string]any{
					"type":    "command",
					"command": "/usr/bin/some-other-hook",
				},
			},
		},
	}
	data, _ := json.Marshal(settings)
	os.WriteFile(settingsPath, data, 0600)

	err := UninstallFromFile(settingsPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ = os.ReadFile(settingsPath)
	var result map[string]any
	json.Unmarshal(data, &result)

	hm, _ := result["hooks"].(map[string]any)
	entries, _ := hm["PreToolUse"].([]any)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry remaining (third-party), got %d", len(entries))
	}
}
