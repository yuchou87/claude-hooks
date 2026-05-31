package hooks_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestInstall_WritesHookEntry(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	os.WriteFile(settingsPath, []byte("{}"), 0600)

	err := hooks.InstallToFile(settingsPath, "/usr/local/bin/claude-hooks", "command", false)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	data, _ := os.ReadFile(settingsPath)
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooksMap, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("no hooks key in settings: %s", data)
	}
	preToolUse, ok := hooksMap["PreToolUse"]
	if !ok {
		t.Fatalf("no PreToolUse hook registered: %s", data)
	}
	_ = preToolUse
}

func TestInstall_Idempotent(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	os.WriteFile(settingsPath, []byte("{}"), 0600)

	hooks.InstallToFile(settingsPath, "/bin/claude-hooks", "command", false)
	hooks.InstallToFile(settingsPath, "/bin/claude-hooks", "command", false)

	data, _ := os.ReadFile(settingsPath)
	var settings map[string]any
	json.Unmarshal(data, &settings)

	hooksMap, _ := settings["hooks"].(map[string]any)
	arr, _ := hooksMap["PreToolUse"].([]any)
	if len(arr) != 1 {
		t.Errorf("idempotent install should not duplicate entries, got %d", len(arr))
	}
}

func TestInstall_CorruptJSON_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	os.WriteFile(settingsPath, []byte("{not valid json"), 0600)

	err := hooks.InstallToFile(settingsPath, "/bin/claude-hooks", "command", false)
	if err == nil {
		t.Fatal("corrupt JSON should return error, not silently overwrite")
	}
}

func TestInstall_DryRun_DoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	os.WriteFile(settingsPath, []byte("{}"), 0600)

	hooks.InstallToFile(settingsPath, "/bin/claude-hooks", "command", true /* dry-run */)

	data, _ := os.ReadFile(settingsPath)
	if string(data) != "{}" {
		t.Errorf("dry-run must not modify file, got %s", data)
	}
}
