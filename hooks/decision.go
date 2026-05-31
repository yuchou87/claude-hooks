package hooks

import "encoding/json"

type permissionDecision struct {
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason,omitempty"`
}

type blockDecision struct {
	Decision          string `json:"decision"`
	Reason            string `json:"reason,omitempty"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// Output is the unified response sent back to Claude Code.
type Output struct {
	Continue      *bool  `json:"continue,omitempty"`
	StopReason    string `json:"stopReason,omitempty"`
	SystemMessage string `json:"systemMessage,omitempty"`
	HookSpecific  any    `json:"-"` // serialized into hookSpecificOutput on wire
	isDeny        bool
	isAsk         bool
}

// IsDeny reports whether this output represents a denial decision.
func (o *Output) IsDeny() bool { return o != nil && o.isDeny }

// IsAsk reports whether this output forwards to Claude Code's built-in permission dialog.
func (o *Output) IsAsk() bool { return o != nil && o.isAsk }

// JSON encodes the Output into the wire format Claude Code expects.
func (o *Output) JSON() ([]byte, error) {
	if o == nil {
		return []byte("{}"), nil
	}
	wire := map[string]any{}
	if o.Continue != nil {
		wire["continue"] = *o.Continue
	}
	if o.StopReason != "" {
		wire["stopReason"] = o.StopReason
	}
	if o.SystemMessage != "" {
		wire["systemMessage"] = o.SystemMessage
	}
	if o.HookSpecific != nil {
		wire["hookSpecificOutput"] = o.HookSpecific
	}
	return json.Marshal(wire)
}

// Deny returns an Output that denies the PreToolUse tool call.
func Deny(reason string) *Output {
	return &Output{
		HookSpecific: permissionDecision{
			PermissionDecision:       "deny",
			PermissionDecisionReason: reason,
		},
		isDeny: true,
	}
}

// Allow returns an Output that explicitly allows the PreToolUse tool call,
// skipping the permission dialog.
func Allow() *Output {
	return &Output{
		HookSpecific: permissionDecision{
			PermissionDecision: "allow",
		},
	}
}

// Ask returns an Output that forwards the PreToolUse decision to Claude Code's
// built-in permission dialog, optionally providing a reason.
func Ask(reason string) *Output {
	return &Output{
		HookSpecific: permissionDecision{
			PermissionDecision:       "ask",
			PermissionDecisionReason: reason,
		},
		isAsk: true,
	}
}

// Defer returns nil — "no opinion", Claude continues its normal flow.
func Defer() *Output { return nil }

// Block returns an Output that blocks Claude after PostToolUse / Stop.
func Block(reason string) *Output {
	return &Output{
		HookSpecific: blockDecision{Decision: "block", Reason: reason},
	}
}

// GlobalStop returns an Output that immediately stops all of Claude's activity.
func GlobalStop(reason string) *Output {
	f := false
	return &Output{Continue: &f, StopReason: reason}
}
