package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// DoctorReport runs all health checks and returns a slice of Check results.
// settingsPath is the absolute path to settings.json for the active scope.
// configPath and scriptsDir are paths to dynamic rules; pass empty strings to
// skip dynamic rule validation. The CLI always resolves these to defaults.
func DoctorReport(settingsPath, configPath, scriptsDir string) []Check {
	var checks []Check

	checks = append(checks, checkInstalledAt(settingsPath))
	checks = append(checks, checkBinaryPath(settingsPath))
	checks = append(checks, checkVersionMatch(settingsPath))
	checks = append(checks, checkConflicts(settingsPath))

	if runtime.GOOS == "darwin" {
		checks = append(checks, checkLaunchd())
	}

	if configPath != "" || scriptsDir != "" {
		dynChecks, _ := ValidateDynamicRules(configPath, scriptsDir)
		checks = append(checks, dynChecks...)
	}

	return checks
}

func checkInstalledAt(settingsPath string) Check {
	label := "installed in settings.json"
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return Check{Label: label, OK: false, Detail: fmt.Sprintf("cannot read %s: %v", settingsPath, err)}
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return Check{Label: label, OK: false, Detail: "settings.json is not valid JSON"}
	}

	hm, _ := settings["hooks"].(map[string]any)
	for _, val := range hm {
		entries, ok := val.([]any)
		if !ok {
			continue
		}
		for _, e := range entries {
			m, _ := e.(map[string]any)
			if m[markerKey] == markerVersion {
				return Check{Label: label, OK: true}
			}
		}
	}
	return Check{Label: label, OK: false, Detail: "no claude-hooks entry found in settings.json"}
}

// checkBinaryPath verifies the binary path in the command-mode settings entry exists on disk.
func checkBinaryPath(settingsPath string) Check {
	label := "binary path exists"
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return Check{Label: label, OK: false, Detail: fmt.Sprintf("cannot read %s: %v", settingsPath, err)}
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return Check{Label: label, OK: false, Detail: "settings.json is not valid JSON"}
	}

	hm, _ := settings["hooks"].(map[string]any)
	for _, val := range hm {
		entries, ok := val.([]any)
		if !ok {
			continue
		}
		for _, e := range entries {
			m, _ := e.(map[string]any)
			if m[markerKey] != markerVersion {
				continue
			}
			cmd, ok := m["command"].(string)
			if !ok {
				// http mode entry has no "command" field
				return Check{Label: label, OK: true, Detail: "http mode (no local binary path)"}
			}
			// command is "/abs/path/to/claude-hooks run" — extract just the binary path
			binPath := strings.SplitN(cmd, " ", 2)[0]
			if _, err := os.Stat(binPath); err != nil {
				return Check{
					Label:  label,
					OK:     false,
					Detail: fmt.Sprintf("binary not found at %s — run: claude-hooks install", binPath),
				}
			}
			return Check{Label: label, OK: true, Detail: binPath}
		}
	}
	return Check{Label: label, OK: false, Detail: "no claude-hooks entry found"}
}

// checkVersionMatch verifies the _claudeHooksVersion marker matches the current binary version.
func checkVersionMatch(settingsPath string) Check {
	label := "version is current"
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return Check{Label: label, OK: false, Detail: fmt.Sprintf("cannot read %s: %v", settingsPath, err)}
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return Check{Label: label, OK: false, Detail: "settings.json is not valid JSON"}
	}

	hm, _ := settings["hooks"].(map[string]any)
	for _, val := range hm {
		entries, ok := val.([]any)
		if !ok {
			continue
		}
		for _, e := range entries {
			m, _ := e.(map[string]any)
			installed, ok := m[markerKey].(string)
			if !ok {
				continue
			}
			if installed == markerVersion {
				return Check{Label: label, OK: true, Detail: installed}
			}
			return Check{
				Label:  label,
				OK:     false,
				Detail: fmt.Sprintf("installed=%s current=%s — run: claude-hooks install", installed, markerVersion),
			}
		}
	}
	return Check{Label: label, OK: false, Detail: "no claude-hooks version marker found"}
}

// knownConflictMarkers are entry keys used by tools that conflict with claude-hooks.
var knownConflictMarkers = []string{"_claudeBarVersion", "_maskoVersion"}

// checkConflicts scans settings.json for known conflicting hook tools.
func checkConflicts(settingsPath string) Check {
	label := "no conflicting tools"
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return Check{Label: label, OK: true} // unreadable → assume no conflict
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return Check{Label: label, OK: true}
	}

	hm, _ := settings["hooks"].(map[string]any)
	for _, val := range hm {
		entries, ok := val.([]any)
		if !ok {
			continue
		}
		for _, e := range entries {
			m, _ := e.(map[string]any)
			for _, marker := range knownConflictMarkers {
				if _, found := m[marker]; found {
					return Check{
						Label:  label,
						OK:     false,
						Detail: fmt.Sprintf("conflicting tool detected (%s) — consider using one hook framework at a time", marker),
					}
				}
			}
		}
	}
	return Check{Label: label, OK: true}
}

func checkLaunchd() Check {
	label := "launchd daemon loaded"
	plistPath := launchdPlistPath()
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		// No plist = command mode install (not HTTP mode) — this is expected, not a failure.
		return Check{Label: label, OK: true, Detail: "not in HTTP mode (no plist expected)"}
	}
	if _, err := exec.Command("launchctl", "list", "com.claude-hooks.daemon").Output(); err != nil {
		return Check{Label: label, OK: false, Detail: "plist found but daemon not loaded in launchd"}
	}
	return Check{Label: label, OK: true}
}
