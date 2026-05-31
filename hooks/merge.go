package hooks

// Merge combines multiple rule outputs into a single Output.
// deny > allow > nil (strictest wins); continue:false (GlobalStop) wins everything.
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
		if result == nil {
			result = o
			continue
		}
		// deny beats everything else for PreToolUse.
		if o.IsDeny() && !result.IsDeny() {
			result = o
		}
	}
	return result
}
