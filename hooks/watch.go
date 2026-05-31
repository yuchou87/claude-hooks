package hooks

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watchDebounce = 200 * time.Millisecond

// StartWatcher watches configPath (file) and scriptsDir (directory) for changes.
// Calls onChange after a 200ms debounce when a write/create/remove event fires.
// Returns a stop function to clean up the watcher goroutine.
// configPath or scriptsDir may be empty string to skip watching that path.
func StartWatcher(configPath, scriptsDir string, onChange func()) (stop func(), err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if configPath != "" {
		_ = watcher.Add(configPath)
	}
	if scriptsDir != "" {
		_ = watcher.Add(scriptsDir)
	}

	var (
		mu    sync.Mutex
		timer *time.Timer
		once  sync.Once
	)

	done := make(chan struct{})
	go func() {
		defer watcher.Close()
		defer func() {
			mu.Lock()
			if timer != nil {
				timer.Stop()
			}
			mu.Unlock()
		}()
		for {
			select {
			case <-done:
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) {
					mu.Lock()
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(watchDebounce, onChange)
					mu.Unlock()
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// watcher errors are non-fatal; continue watching
			}
		}
	}()

	return func() { once.Do(func() { close(done) }) }, nil
}
