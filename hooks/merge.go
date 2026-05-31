package hooks

// outputPriority returns the priority of an Output for merge conflict resolution.
// Higher priority wins: deny=3, ask=2, allow/other=1, nil=0.
func outputPriority(o *Output) int {
	if o == nil {
		return 0
	}
	if o.IsDeny() {
		return 3
	}
	if o.IsAsk() {
		return 2
	}
	return 1
}

// Merge combines multiple rule outputs into a single Output.
// deny > ask > allow > nil (strictest wins); continue:false (GlobalStop) wins everything.
// ev is reserved for future per-event-type merge semantics (Plan 3).
func Merge(_ Input, outputs []*Output) *Output {
	var result *Output

	for _, o := range outputs {
		if o == nil {
			continue
		}
		// GlobalStop (continue:false) wins unconditionally.
		if o.Continue != nil && !*o.Continue {
			return o
		}
		if outputPriority(o) > outputPriority(result) {
			result = o
		}
	}
	return result
}
