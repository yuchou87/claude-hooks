package hooks_test

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestStartWatcher_FileChange_TriggersOnChange(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("rules: []"), 0644); err != nil {
		t.Fatal(err)
	}

	changed := make(chan struct{}, 1)
	stop, err := hooks.StartWatcher(configPath, "", func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatalf("StartWatcher: %v", err)
	}
	defer stop()

	// Modify the file — watcher should fire within 500ms (200ms debounce + margin)
	time.Sleep(50 * time.Millisecond) // let watcher settle
	if err := os.WriteFile(configPath, []byte("rules: []"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-changed:
		// success
	case <-time.After(1 * time.Second):
		t.Error("onChange was not called after file modification")
	}
}

func TestStartWatcher_Stop_StopsWatching(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("rules: []"), 0644); err != nil {
		t.Fatal(err)
	}

	var callCount atomic.Int64
	stop, err := hooks.StartWatcher(configPath, "", func() { callCount.Add(1) })
	if err != nil {
		t.Fatalf("StartWatcher: %v", err)
	}

	stop() // stop before any changes

	if err := os.WriteFile(configPath, []byte("rules: []"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)

	if callCount.Load() > 0 {
		t.Errorf("onChange called %d times after stop", callCount.Load())
	}
}

func TestStartWatcher_NewScriptFile_TriggersOnChange(t *testing.T) {
	dir := t.TempDir()
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	changed := make(chan struct{}, 1)
	stop, err := hooks.StartWatcher("", scriptsDir, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatalf("StartWatcher: %v", err)
	}
	defer stop()

	time.Sleep(50 * time.Millisecond)
	// Create a new script file — should trigger onChange
	if err := os.WriteFile(filepath.Join(scriptsDir, "new.js"), []byte(`
export const events = ["PreToolUse"];
export function decide(e) { return null; }
`), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-changed:
		// success
	case <-time.After(1 * time.Second):
		t.Error("onChange was not called after new script file created")
	}
}
