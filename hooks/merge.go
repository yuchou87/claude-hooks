package hooks

// Merge combines multiple rule outputs into a single Output.
// For PreToolUse: deny > allow > nil (strictest wins).
// For all events: continue:false (GlobalStop) wins everything.
func Merge(ev Input, outputs []*Output) *Output {
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
