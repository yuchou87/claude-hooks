package hooks

import "fmt"

// Check represents a single validation or diagnostic result.
type Check struct {
	Label  string // short description of what was checked
	OK     bool   // true = pass, false = fail
	Detail string // human-readable detail (may be empty on pass)
}

// ValidateDynamicRules validates config.yaml and all scripts in scriptsDir.
// Returns a slice of Check results and true if all checks passed.
// Missing files/directories are treated as empty (valid).
func ValidateDynamicRules(configPath, scriptsDir string) ([]Check, bool) {
	var checks []Check
	allOK := true

	// Validate YAML config
	if configPath != "" {
		label := fmt.Sprintf("YAML config: %s", configPath)
		_, err := BuildDynamicRules(configPath, "")
		if err != nil {
			checks = append(checks, Check{Label: label, OK: false, Detail: err.Error()})
			allOK = false
		} else {
			checks = append(checks, Check{Label: label, OK: true})
		}
	}

	// Validate scripts
	if scriptsDir != "" {
		label := fmt.Sprintf("scripts dir: %s", scriptsDir)
		_, err := BuildDynamicRules("", scriptsDir)
		if err != nil {
			checks = append(checks, Check{Label: label, OK: false, Detail: err.Error()})
			allOK = false
		} else {
			checks = append(checks, Check{Label: label, OK: true})
		}
	}

	// If both are empty strings, return a single passing check
	if configPath == "" && scriptsDir == "" {
		checks = append(checks, Check{Label: "no config or scripts dir specified", OK: true})
	}

	return checks, allOK
}
