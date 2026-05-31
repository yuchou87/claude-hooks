package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDoctorReport_NotInstalled(t *testing.T) {
	checks := DoctorReport("/nonexistent/settings.json", "/nonexistent/config.yaml", "/nonexistent/scripts")
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	var installCheck *Check
	for i, c := range checks {
		if c.Label == "installed in settings.json" {
			installCheck = &checks[i]
		}
	}
	if installCheck == nil {
		t.Fatal("expected 'installed in settings.json' check")
	}
	if installCheck.OK {
		t.Error("expected install check to fail for missing settings.json")
	}
}

func TestDoctorReport_Installed(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					markerKey:   markerVersion,
					"type":      "command",
					"command":   "/bin/claude-hooks run",
				},
			},
		},
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	checks := DoctorReport(settingsPath, "/nonexistent/config.yaml", "")
	var installCheck *Check
	for i, c := range checks {
		if c.Label == "installed in settings.json" {
			installCheck = &checks[i]
		}
	}
	if installCheck == nil {
		t.Fatal("expected 'installed in settings.json' check")
	}
	if !installCheck.OK {
		t.Errorf("expected install check to pass, got detail: %s", installCheck.Detail)
	}
}
