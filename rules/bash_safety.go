package rules

import (
	"strings"

	"github.com/yuchou87/claude-hooks/hooks"
)

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
			dangerousPatterns := []string{
				"rm -rf /",
				"rm -rf ~",
				":(){ :|:& };:", // fork bomb
				"> /dev/sda",
				"dd if=/dev/zero of=/dev/",
			}
			for _, p := range dangerousPatterns {
				if strings.Contains(cmd, p) {
					return hooks.Deny("dangerous command blocked by bash-safety rule: " + p)
				}
			}
			return nil
		},
	})
}
