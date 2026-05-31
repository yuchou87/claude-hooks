package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// DoctorReport runs all health checks and returns a slice of Check results.
// settingsPath is the absolute path to settings.json for the active scope.
// configPath and scriptsDir are paths to dynamic rules; pass empty strings to
// skip dynamic rule validation. The CLI always resolves these to defaults.
func DoctorReport(settingsPath, configPath, scriptsDir string) []Check {
	var checks []Check

	checks = append(checks, checkInstalledAt(settingsPath))

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

func checkLaunchd() Check {
	label := "launchd daemon loaded"
	plistPath := launchdPlistPath()
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return Check{Label: label, OK: false, Detail: fmt.Sprintf("plist not found: %s", plistPath)}
	}
	if _, err := exec.Command("launchctl", "list", "com.claude-hooks.daemon").Output(); err != nil {
		return Check{Label: label, OK: false, Detail: "daemon not loaded in launchd"}
	}
	return Check{Label: label, OK: true}
}
