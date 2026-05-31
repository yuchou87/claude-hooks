package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const markerKey = "_claudeHooksVersion"
const markerVersion = "1.0.0"

// Install registers claude-hooks in the appropriate settings.json.
// addr is the daemon listen address used when mode == "http" (ignored for command mode).
func Install(mode, scope, addr string, dryRun bool) error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine binary path: %w", err)
	}

	settingsPath, err := settingsFilePath(scope)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0700); err != nil {
		return fmt.Errorf("cannot create settings dir: %w", err)
	}

	return InstallToFile(settingsPath, binPath, mode, dryRun)
}

// InstallToFile writes hook entries into the given settings file.
// Exported for testing with temporary directories.
func InstallToFile(settingsPath, binaryPath, mode string, dryRun bool) error {
	var settings map[string]any
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("settings.json is corrupt (will not overwrite): %w", err)
		}
	} else {
		settings = map[string]any{}
	}

	hooksMap, _ := settings["hooks"].(map[string]any)
	if hooksMap == nil {
		hooksMap = map[string]any{}
		settings["hooks"] = hooksMap
	}

	var entry map[string]any
	switch mode {
	case "command":
		entry = map[string]any{
			"type":    "command",
			"command": binaryPath + " run",
			markerKey: markerVersion,
		}
	default:
		return fmt.Errorf("unsupported mode: %s (use command or http)", mode)
	}

	// Idempotent: check if our entry is already there (match by markerKey).
	existing, _ := hooksMap["PreToolUse"].([]any)
	for _, e := range existing {
		m, _ := e.(map[string]any)
		if m[markerKey] == markerVersion {
			fmt.Println("claude-hooks: already installed (idempotent, no change)")
			return nil
		}
	}
	hooksMap["PreToolUse"] = append(existing, entry)

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

	fmt.Printf("claude-hooks installed (%s mode) -> %s\n", mode, settingsPath)
	return nil
}

func settingsFilePath(scope string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch scope {
	case "user":
		return filepath.Join(home, ".claude", "settings.json"), nil
	case "project":
		return filepath.Join(".claude", "settings.json"), nil
	case "local":
		return filepath.Join(".claude", "settings.local.json"), nil
	default:
		return "", fmt.Errorf("unknown scope: %s (use user|project|local)", scope)
	}
}
