package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const markerKey = "_claudeHooksVersion"
const markerVersion = "1.0.0"

// Install registers claude-hooks in the appropriate settings.json.
func Install(mode, scope string, dryRun bool) error {
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
	case "http":
		entry = map[string]any{
			"type":             "http",
			"url":              "http://127.0.0.1:8787/hook",
			markerKey:          markerVersion,
			"_claudeHooksMode": "http",
		}
	default:
		return fmt.Errorf("unsupported mode: %s (use command or http)", mode)
	}

	// Idempotent: check if our entry (same mode) is already there.
	existing, _ := hooksMap["PreToolUse"].([]any)
	for _, e := range existing {
		m, _ := e.(map[string]any)
		if m[markerKey] == markerVersion && m["_claudeHooksMode"] == entry["_claudeHooksMode"] {
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

	// For http mode on macOS: generate and load the launchd plist.
	if mode == "http" && runtime.GOOS == "darwin" && !dryRun {
		if err := installLaunchd(binaryPath, "127.0.0.1:8787"); err != nil {
			fmt.Fprintf(os.Stderr, "warning: launchd setup failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "  Run manually: launchctl load %s\n", launchdPlistPath())
		}
	}
	return nil
}

// GeneratePlistContent returns the launchd plist XML for the HTTP daemon.
// Exported so tests can verify the content without side effects.
func GeneratePlistContent(binaryPath, addr string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.claude-hooks.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>serve</string>
        <string>--addr</string>
        <string>%s</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/tmp/claude-hooks-stderr.log</string>
    <key>EnvironmentVariables</key>
    <dict/>
</dict>
</plist>
`, binaryPath, addr)
}

func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.claude-hooks.daemon.plist")
}

func installLaunchd(binaryPath, addr string) error {
	plistPath := launchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("cannot create LaunchAgents dir: %w", err)
	}
	content := GeneratePlistContent(binaryPath, addr)
	if err := os.WriteFile(plistPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("cannot write plist: %w", err)
	}
	// launchctl load starts the daemon immediately and at login.
	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load failed: %v\n%s", err, out)
	}
	fmt.Printf("claude-hooks daemon loaded via launchd: %s\n", plistPath)
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
