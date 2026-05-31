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

func TestCheckBinaryPath_Exists(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	binPath, _ := os.Executable()
	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					markerKey:    markerVersion,
					markerBinKey: binPath, // new explicit binary path marker
					"type":       "command",
					"command":    binPath + " run",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0600)

	check := checkBinaryPath(settingsPath)
	if !check.OK {
		t.Errorf("expected OK=true for existing binary, got detail: %s", check.Detail)
	}
}

func TestCheckBinaryPath_FallbackCommandSplit(t *testing.T) {
	// Entries installed before markerBinKey existed use command-string fallback.
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	binPath, _ := os.Executable()
	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					markerKey: markerVersion,
					"type":    "command",
					"command": binPath + " run", // no markerBinKey
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0600)

	check := checkBinaryPath(settingsPath)
	if !check.OK {
		t.Errorf("expected OK=true with fallback path detection, got detail: %s", check.Detail)
	}
}

func TestCheckBinaryPath_Missing(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					markerKey: markerVersion,
					"type":    "command",
					"command": "/nonexistent/path/claude-hooks run",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0600)

	check := checkBinaryPath(settingsPath)
	if check.OK {
		t.Error("expected OK=false for missing binary")
	}
	if check.Detail == "" {
		t.Error("expected non-empty Detail for missing binary")
	}
}

func TestCheckVersionMatch_Current(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					markerKey: markerVersion,
					"type":    "command",
					"command": "/bin/claude-hooks run",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0600)

	check := checkVersionMatch(settingsPath)
	if !check.OK {
		t.Errorf("expected OK=true for current version, got detail: %s", check.Detail)
	}
}

func TestCheckVersionMatch_Outdated(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					markerKey: "0.9.0",
					"type":    "command",
					"command": "/bin/claude-hooks run",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0600)

	check := checkVersionMatch(settingsPath)
	if check.OK {
		t.Error("expected OK=false for outdated version")
	}
	if check.Detail == "" {
		t.Error("expected non-empty Detail for version mismatch")
	}
}

func TestCheckConflicts_NoConflict(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					markerKey: markerVersion,
					"type":    "command",
					"command": "/bin/claude-hooks run",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0600)

	check := checkConflicts(settingsPath)
	if !check.OK {
		t.Errorf("expected OK=true with no conflicts, got detail: %s", check.Detail)
	}
}

func TestCheckConflicts_UnreadableSettings(t *testing.T) {
	check := checkConflicts("/nonexistent/settings.json")
	if check.OK {
		t.Error("expected OK=false when settings.json is unreadable")
	}
	if check.Detail == "" {
		t.Error("expected non-empty Detail when settings.json is unreadable")
	}
}

func TestCheckConflicts_ClaudeBarDetected(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"_claudeBarVersion": "2.0.0",
					"type":              "command",
					"command":           "claudebar run",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0600)

	check := checkConflicts(settingsPath)
	if check.OK {
		t.Error("expected OK=false when ClaudeBar marker is present")
	}
	if check.Detail == "" {
		t.Error("expected non-empty Detail for conflict detection")
	}
}

func TestDoctorReport_IncludesNewChecks(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")

	binPath, _ := os.Executable()
	settings := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					markerKey: markerVersion,
					"type":    "command",
					"command": binPath + " run",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0600)

	checks := DoctorReport(settingsPath, "", "")

	labels := make(map[string]bool)
	for _, c := range checks {
		labels[c.Label] = true
	}
	for _, expected := range []string{"binary path exists", "version is current", "no conflicting tools"} {
		if !labels[expected] {
			t.Errorf("DoctorReport missing check: %q", expected)
		}
	}
}
