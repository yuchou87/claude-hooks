package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Uninstall removes claude-hooks entries from the settings.json at the given scope.
func Uninstall(scope string, dryRun bool) error {
	settingsPath, err := settingsFilePath(scope)
	if err != nil {
		return err
	}
	return UninstallFromFile(settingsPath, dryRun)
}

// UninstallFromFile removes all claude-hooks entries from settingsPath.
// It is safe to call when the file does not exist (no-op, no error).
func UninstallFromFile(settingsPath string, dryRun bool) error {
	data, err := os.ReadFile(settingsPath)
	if os.IsNotExist(err) {
		fmt.Println("claude-hooks: not installed (settings file not found)")
		return nil
	}
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("settings.json is corrupt: %w", err)
	}

	hooksMap, _ := settings["hooks"].(map[string]any)
	if hooksMap == nil {
		fmt.Println("claude-hooks: not installed (no hooks section)")
		return nil
	}

	removed := 0
	for eventKey, val := range hooksMap {
		entries, ok := val.([]any)
		if !ok {
			continue // not an array — not our entry, leave it alone
		}
		var keep []any
		for _, e := range entries {
			m, _ := e.(map[string]any)
			if m[markerKey] == markerVersion {
				removed++
			} else {
				keep = append(keep, e)
			}
		}
		if len(keep) == 0 {
			delete(hooksMap, eventKey)
		} else {
			hooksMap[eventKey] = keep
		}
	}

	if removed == 0 {
		fmt.Println("claude-hooks: not installed (no matching entries found)")
		return nil
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot encode settings: %w", err)
	}
	out = append(out, '\n')

	if dryRun {
		fmt.Printf("[dry-run] would write to %s:\n%s\n", settingsPath, out)
		return nil
	}

	tmp := settingsPath + ".tmp"
	if err := os.WriteFile(tmp, out, 0600); err != nil {
		return fmt.Errorf("cannot write temp file: %w", err)
	}
	if err := os.Rename(tmp, settingsPath); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("atomic rename failed: %w", err)
	}

	fmt.Printf("claude-hooks uninstalled: removed %d entry/entries from %s\n", removed, settingsPath)

	if runtime.GOOS == "darwin" {
		if err := unloadLaunchd(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: launchd unload failed: %v\n", err)
		}
	}
	return nil
}

func unloadLaunchd() error {
	plistPath := launchdPlistPath()
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return nil // not installed via launchd
	}
	if out, err := exec.Command("launchctl", "unload", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl unload failed: %v\n%s", err, out)
	}
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	fmt.Printf("claude-hooks daemon unloaded and plist removed: %s\n", plistPath)
	return nil
}
