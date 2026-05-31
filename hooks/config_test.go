package hooks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestLoadConfig_Missing_ReturnsEmpty(t *testing.T) {
	rules, err := hooks.LoadConfig("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("missing file must not error, got: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("want 0 rules, got %d", len(rules))
	}
}

func TestLoadConfig_DenyRule_FiresOnToolMatch(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	yaml := `
rules:
  - name: deny-write
    event: PreToolUse
    when:
      tool: [Write, Edit]
    decision: deny
    reason: "no writes"
`
	path := writeTemp(t, "config.yaml", yaml)
	rules, err := hooks.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rules))
	}

	hooks.StoreDynamic(rules)

	// Write tool → deny
	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{}}`))
	if out == nil || !out.IsDeny() {
		t.Errorf("Write must be denied, got %+v", out)
	}

	// Bash tool → nil (no match)
	out = hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash","tool_input":{}}`))
	if out != nil {
		t.Errorf("Bash must not be denied, got %+v", out)
	}
}

func TestLoadConfig_AllowRule(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	yaml := `
rules:
  - name: allow-read
    event: PreToolUse
    when:
      tool: [Read]
    decision: allow
`
	path := writeTemp(t, "config.yaml", yaml)
	rules, _ := hooks.LoadConfig(path)
	hooks.StoreDynamic(rules)

	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Read","tool_input":{}}`))
	if out == nil || out.IsDeny() {
		t.Errorf("Read must be allowed (non-nil, non-deny), got %+v", out)
	}
}

func TestLoadConfig_FilePath_GlobMatch(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	yaml := `
rules:
  - name: protect-env
    event: PreToolUse
    when:
      tool: [Write]
      file_path: ["**/.env", "**/secrets/**"]
    decision: deny
    reason: "no secrets"
`
	path := writeTemp(t, "config.yaml", yaml)
	rules, _ := hooks.LoadConfig(path)
	hooks.StoreDynamic(rules)

	// .env file → deny
	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{"file_path":"/home/user/project/.env"}}`))
	if out == nil || !out.IsDeny() {
		t.Errorf(".env write must be denied, got %+v", out)
	}

	// normal file → nil
	out = hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{"file_path":"/home/user/project/main.go"}}`))
	if out != nil {
		t.Errorf("main.go write must not be denied, got %+v", out)
	}
}

func TestLoadConfig_Cwd_GlobMatch(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	yaml := `
rules:
  - name: prod-guard
    event: PreToolUse
    when:
      tool: [Bash]
      cwd: ["/prod/**"]
    decision: deny
    reason: "no bash in prod"
`
	path := writeTemp(t, "config.yaml", yaml)
	rules, _ := hooks.LoadConfig(path)
	hooks.StoreDynamic(rules)

	// prod cwd → deny
	out := hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/prod/app","tool_name":"Bash","tool_input":{}}`))
	if out == nil || !out.IsDeny() {
		t.Errorf("Bash in /prod/** must be denied, got %+v", out)
	}

	// dev cwd → nil
	out = hooks.Dispatch([]byte(`{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/dev/app","tool_name":"Bash","tool_input":{}}`))
	if out != nil {
		t.Errorf("Bash in /dev/** must not be denied, got %+v", out)
	}
}

func TestLoadConfig_MultipleRules(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	yaml := `
rules:
  - name: rule-one
    event: PreToolUse
    when:
      tool: [Write]
    decision: deny
    reason: "rule one"
  - name: rule-two
    event: PreToolUse
    when:
      tool: [Edit]
    decision: deny
    reason: "rule two"
`
	path := writeTemp(t, "config.yaml", yaml)
	rules, err := hooks.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("want 2 rules, got %d", len(rules))
	}
}

// writeTemp writes content to a temp file and returns the path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
