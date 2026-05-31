package hooks

import (
	"fmt"
	"sync/atomic"
)

// Rule defines a hook rule. Register it with Register(); it runs for every
// matching event via Dispatch().
type Rule struct {
	Name   string
	Events []string // hook_event_name values this rule cares about
	Match  string   // optional: tool name matcher (used by install for `if` pushdown)
	Run    func(Input) *Output
}

// ruleSet is an immutable snapshot of registered rules.
// Replaced atomically on hot-reload (Plan 3).
type ruleSet struct {
	rules []Rule
}

var active atomic.Pointer[ruleSet]

func init() {
	active.Store(&ruleSet{})
}

// Register adds a rule to the registry. Call from init() in rules/*.go.
// Panics if a rule with the same Name is already registered (startup-time check).
func Register(r Rule) {
	for {
		old := active.Load()
		for _, existing := range old.rules {
			if existing.Name == r.Name {
				panic("claude-hooks: duplicate rule name: " + r.Name)
			}
		}
		next := &ruleSet{rules: append(append([]Rule{}, old.rules...), r)}
		if active.CompareAndSwap(old, next) {
			return
		}
	}
}

// Dispatch parses raw JSON, runs all matching rules, and merges results.
// Always returns nil on JSON parse error (fail-open).
func Dispatch(raw []byte) *Output {
	ev, err := ParseInput(raw)
	if err != nil {
		LogError(Input{}, "parse error", err)
		return nil // fail-open
	}

	rs := active.Load()
	outputs := make([]*Output, 0, len(rs.rules))
	for _, r := range rs.rules {
		if !eventMatches(r, ev) {
			continue
		}
		out := safeRun(r, ev)
		outputs = append(outputs, out)
		if out != nil {
			LogDecision(ev, out, r.Name)
		}
	}

	return Merge(ev, outputs)
}

// safeRun calls r.Run and recovers from any panic, returning nil on panic.
func safeRun(r Rule, ev Input) (out *Output) {
	defer func() {
		if rec := recover(); rec != nil {
			LogError(ev, "rule panic: "+r.Name, fmt.Errorf("%v", rec))
			out = nil // fail-open
		}
	}()
	return r.Run(ev)
}

// eventMatches reports whether rule r should run for event ev.
func eventMatches(r Rule, ev Input) bool {
	for _, e := range r.Events {
		if e == ev.HookEventName {
			return true
		}
	}
	return false
}

// ResetRegistryForTest clears all registered rules. Call from t.Cleanup in tests
// that call Register, to prevent rule accumulation across -count=2 runs.
func ResetRegistryForTest() {
	active.Store(&ruleSet{})
}
