package hooks

import "encoding/json"

// Input holds a parsed hook event. Known fields are typed; unknown fields
// land in Extra to survive schema drift across Claude Code versions.
type Input struct {
	HookEventName  string         `json:"hook_event_name"`
	SessionID      string         `json:"session_id"`
	TranscriptPath string         `json:"transcript_path"`
	Cwd            string         `json:"cwd"`
	PermissionMode string         `json:"permission_mode"`
	ToolName       string         `json:"tool_name,omitempty"`
	ToolInput      map[string]any `json:"tool_input,omitempty"`
	ToolUseID      string         `json:"tool_use_id,omitempty"`
	ToolResult     string         `json:"tool_result,omitempty"`
	Extra          map[string]any `json:"-"`
}

// ParseInput decodes raw JSON into an Input. Unknown fields are stored in Extra.
func ParseInput(raw []byte) (Input, error) {
	var all map[string]any
	if err := json.Unmarshal(raw, &all); err != nil {
		return Input{}, err
	}

	var in Input
	if err := json.Unmarshal(raw, &in); err != nil {
		return Input{}, err
	}

	known := map[string]bool{
		"hook_event_name": true, "session_id": true, "transcript_path": true,
		"cwd": true, "permission_mode": true, "tool_name": true,
		"tool_input": true, "tool_use_id": true, "tool_result": true,
	}
	in.Extra = make(map[string]any)
	for k, v := range all {
		if !known[k] {
			in.Extra[k] = v
		}
	}
	return in, nil
}
