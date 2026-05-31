package hooks_test

import (
	"testing"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestMerge_PreToolUse(t *testing.T) {
	tests := []struct {
		name     string
		outputs  []*hooks.Output
		wantDeny bool
		wantNil  bool
	}{
		{
			name:    "all nil → nil (fail-open)",
			outputs: []*hooks.Output{nil, nil},
			wantNil: true,
		},
		{
			name:     "deny wins over allow",
			outputs:  []*hooks.Output{hooks.Allow(), hooks.Deny("bad")},
			wantDeny: true,
		},
		{
			name:     "deny wins over nil",
			outputs:  []*hooks.Output{nil, hooks.Deny("bad")},
			wantDeny: true,
		},
		{
			name:    "allow + nil → allow (not nil)",
			outputs: []*hooks.Output{hooks.Allow(), nil},
			wantNil: false,
		},
		{
			name:    "empty slice → nil",
			outputs: []*hooks.Output{},
			wantNil: true,
		},
		{
			name:     "multiple denies → deny",
			outputs:  []*hooks.Output{hooks.Deny("a"), hooks.Deny("b")},
			wantDeny: true,
		},
	}

	ev := hooks.Input{HookEventName: "PreToolUse"}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hooks.Merge(ev, tc.outputs)
			if tc.wantNil && got != nil {
				t.Errorf("want nil, got %+v", got)
			}
			if !tc.wantNil && got == nil {
				t.Error("want non-nil, got nil")
			}
			if tc.wantDeny && !got.IsDeny() {
				t.Errorf("want deny, got non-deny: %+v", got)
			}
			if !tc.wantDeny && got != nil && got.IsDeny() {
				t.Errorf("want non-deny, got deny")
			}
		})
	}
}

func TestMerge_GlobalStop_WinsOverAll(t *testing.T) {
	ev := hooks.Input{HookEventName: "PreToolUse"}
	outputs := []*hooks.Output{
		hooks.Allow(),
		hooks.GlobalStop("emergency"),
		hooks.Deny("also bad"),
	}
	got := hooks.Merge(ev, outputs)
	if got == nil {
		t.Fatal("want non-nil")
	}
	if got.Continue == nil || *got.Continue != false {
		t.Error("GlobalStop must set Continue=false")
	}
}

func TestMerge_NonPreToolUse_NilIsNil(t *testing.T) {
	ev := hooks.Input{HookEventName: "Stop"}
	got := hooks.Merge(ev, []*hooks.Output{nil, nil})
	if got != nil {
		t.Errorf("all-nil should return nil, got %+v", got)
	}
}
