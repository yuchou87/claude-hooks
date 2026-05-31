package hooks

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// Config is the top-level structure of ~/.claude-hooks/config.yaml.
type Config struct {
	Rules []YAMLRule `yaml:"rules"`
}

// YAMLRule is a single declarative rule entry in config.yaml.
type YAMLRule struct {
	Name     string     `yaml:"name"`
	Event    string     `yaml:"event"`
	When     WhenClause `yaml:"when"`
	Decision string     `yaml:"decision"` // "allow" | "deny"
	Reason   string     `yaml:"reason"`
}

// WhenClause holds match conditions for a YAML rule.
// All non-empty conditions must match (AND logic within a rule).
type WhenClause struct {
	Tool     []string `yaml:"tool"`      // exact tool name match (OR within list)
	FilePath []string `yaml:"file_path"` // glob patterns for tool_input.file_path
	Cwd      []string `yaml:"cwd"`       // glob patterns for event cwd
}

// LoadConfig reads a YAML rule file and returns the resulting []Rule.
// Returns an empty slice (no error) if the file does not exist.
func LoadConfig(configPath string) ([]Rule, error) {
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	rules := make([]Rule, 0, len(cfg.Rules))
	for _, yr := range cfg.Rules {
		r, err := yamlRuleToRule(yr)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", yr.Name, err)
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func yamlRuleToRule(yr YAMLRule) (Rule, error) {
	if yr.Name == "" {
		return Rule{}, fmt.Errorf("name is required")
	}
	if yr.Event == "" {
		return Rule{}, fmt.Errorf("event is required")
	}
	if yr.Decision != "allow" && yr.Decision != "deny" {
		return Rule{}, fmt.Errorf("decision must be 'allow' or 'deny', got %q", yr.Decision)
	}

	when := yr.When
	decision := yr.Decision
	reason := yr.Reason

	return Rule{
		Name:   yr.Name,
		Events: []string{yr.Event},
		Run: func(ev Input) *Output {
			// Check tool match (OR within list)
			if len(when.Tool) > 0 {
				matched := false
				for _, t := range when.Tool {
					if t == ev.ToolName {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}

			// Check file_path glob (OR within list)
			if len(when.FilePath) > 0 {
				fp, _ := ev.ToolInput["file_path"].(string)
				if fp == "" {
					return nil
				}
				matched := false
				for _, pattern := range when.FilePath {
					if matchGlob(pattern, fp) {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}

			// Check cwd glob (OR within list)
			if len(when.Cwd) > 0 {
				matched := false
				for _, pattern := range when.Cwd {
					if matchGlob(pattern, ev.Cwd) {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}

			switch decision {
			case "deny":
				return Deny(reason)
			case "allow":
				return Allow()
			default:
				return nil
			}
		},
	}, nil
}

// matchGlob matches a path against a glob pattern supporting ** (any depth).
// Pattern examples: "**/.env", "**/secrets/**", "/prod/**"
func matchGlob(pattern, target string) bool {
	// Normalize to forward slashes
	pattern = filepath.ToSlash(pattern)
	target = filepath.ToSlash(target)

	// Expand ~ in target to home dir
	if strings.HasPrefix(target, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			target = filepath.ToSlash(filepath.Join(home, target[2:]))
		}
	}

	return matchGlobParts(strings.Split(pattern, "/"), strings.Split(target, "/"))
}

// matchGlobParts recursively matches pattern parts against target parts.
func matchGlobParts(pp, tp []string) bool {
	for len(pp) > 0 {
		p := pp[0]
		if p == "**" {
			// When ** is the last segment, require at least one path component.
			start := 0
			if len(pp[1:]) == 0 {
				start = 1
			}
			for i := start; i <= len(tp); i++ {
				if matchGlobParts(pp[1:], tp[i:]) {
					return true
				}
			}
			return false
		}
		if len(tp) == 0 {
			return false
		}
		ok, _ := path.Match(p, tp[0])
		if !ok {
			return false
		}
		pp = pp[1:]
		tp = tp[1:]
	}
	return len(tp) == 0
}
