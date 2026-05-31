package hooks

// ResetLogFileForTest closes and resets the cached log file handle.
// Call from t.Cleanup in tests that change CLAUDE_HOOKS_LOG_DIR.
// Compiled for test builds only.
func ResetLogFileForTest() {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}
