package rules

import (
	"regexp"
	"strings"

	"github.com/yuchou87/claude-hooks/hooks"
)

// rmRfRoot matches rm -rf targeting the filesystem root or home directory,
// but NOT legitimate subpaths like /tmp/build or ~/Documents.
//
// Matches: rm -rf /   rm -rf ~   rm -rf ~/   rm -rf / ; ...
// Allows:  rm -rf /tmp/build    rm -rf ~/Documents
//
// Pattern: after flags, the path must be / OR ~/? (tilde with optional slash),
// followed by end-of-string or a shell metacharacter — not a path component.
var rmRfRoot = regexp.MustCompile(`rm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+(/|~/?)(\s|;|&|\||$)`)

func init() {
	hooks.Register(hooks.Rule{
		Name:   "bash-safety",
		Events: []string{"PreToolUse"},
		Match:  "Bash",
		Run: func(in hooks.Input) *hooks.Output {
			if in.ToolName != "Bash" {
				return nil
			}
			cmd, _ := in.ToolInput["command"].(string)

			if rmRfRoot.MatchString(cmd) {
				return hooks.Deny("dangerous command blocked by bash-safety rule: rm -rf targeting filesystem root")
			}

			exactPatterns := []string{
				":(){ :|:& };:", // fork bomb
				"> /dev/sda",
				"dd if=/dev/zero of=/dev/",
			}
			for _, p := range exactPatterns {
				if strings.Contains(cmd, p) {
					return hooks.Deny("dangerous command blocked by bash-safety rule: " + p)
				}
			}
			return nil
		},
	})
}
