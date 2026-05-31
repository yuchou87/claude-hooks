package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildDynamicRules loads YAML rules from configPath and script rules from all
// .js/.ts/.jsx/.tsx files in scriptsDir. Returns combined []Rule.
// Missing files/directories are silently skipped (not an error).
// Invalid config or scripts return an error.
func BuildDynamicRules(configPath, scriptsDir string) ([]Rule, error) {
	var rules []Rule

	// Layer 2: YAML rules
	if configPath != "" {
		yamlRules, err := LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}
		rules = append(rules, yamlRules...)
	}

	// Layer 3: script rules
	if scriptsDir != "" {
		scriptRules, err := loadScriptRules(scriptsDir)
		if err != nil {
			return nil, fmt.Errorf("load scripts: %w", err)
		}
		rules = append(rules, scriptRules...)
	}

	return rules, nil
}

// loadScriptRules reads all .js/.ts/.jsx/.tsx files in dir and compiles each
// into a Rule via NewScriptEngine. Skips non-existent directory.
func loadScriptRules(dir string) ([]Rule, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read scripts dir: %w", err)
	}

	var rules []Rule
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		switch strings.ToLower(filepath.Ext(name)) {
		case ".js", ".ts", ".jsx", ".tsx":
		default:
			continue
		}
		eng, err := NewScriptEngine(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("script %s: %w", name, err)
		}
		rules = append(rules, eng.AsRule())
	}
	return rules, nil
}
