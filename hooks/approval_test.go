package hooks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yuchou87/claude-hooks/hooks"
)

func TestApprover_Approved_ReturnsAllow(t *testing.T) {
	a := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			return true, nil
		},
		Timeout: 2 * time.Second,
	}
	out := a.Handle(context.Background(), hooks.Input{ToolName: "Bash"})
	if out == nil || out.IsDeny() {
		t.Errorf("approved dialog must return Allow, got %+v", out)
	}
}

func TestApprover_Rejected_ReturnsDeny(t *testing.T) {
	a := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			return false, nil
		},
		Timeout: 2 * time.Second,
	}
	out := a.Handle(context.Background(), hooks.Input{ToolName: "Write"})
	if out == nil || !out.IsDeny() {
		t.Errorf("rejected dialog must return Deny, got %+v", out)
	}
}

func TestApprover_Timeout_ReturnsNil(t *testing.T) {
	a := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			<-ctx.Done() // block until timeout fires
			return false, ctx.Err()
		},
		Timeout: 50 * time.Millisecond,
	}
	out := a.Handle(context.Background(), hooks.Input{ToolName: "Bash"})
	if out != nil {
		t.Errorf("timeout must return nil (defer/fail-open), got %+v", out)
	}
}

func TestApprover_DialogError_ReturnsNil(t *testing.T) {
	a := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			return false, errors.New("osascript not available")
		},
		Timeout: 2 * time.Second,
	}
	out := a.Handle(context.Background(), hooks.Input{ToolName: "Bash"})
	if out != nil {
		t.Errorf("dialog error must return nil (fail-open), got %+v", out)
	}
}

func TestApprover_ContextCancel_ReturnsNil(t *testing.T) {
	// Simulates SIGTERM cancelling the shutCtx passed by Server.
	dialogStarted := make(chan struct{})
	a := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			close(dialogStarted)
			<-ctx.Done()
			return false, ctx.Err()
		},
		Timeout: 10 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan *hooks.Output, 1)
	go func() { resultCh <- a.Handle(ctx, hooks.Input{ToolName: "Bash"}) }()
	<-dialogStarted
	cancel() // simulate SIGTERM
	out := <-resultCh
	if out != nil {
		t.Errorf("cancelled context must return nil (defer), got %+v", out)
	}
}

func TestApprover_Concurrent_NoRace(t *testing.T) {
	// Run 5 concurrent approvals — race detector must pass.
	unblock := make(chan struct{})
	a := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			<-unblock
			return true, nil
		},
		Timeout: 5 * time.Second,
	}
	done := make(chan struct{}, 5)
	for i := 0; i < 5; i++ {
		go func() {
			a.Handle(context.Background(), hooks.Input{ToolName: "Read"})
			done <- struct{}{}
		}()
	}
	close(unblock)
	for i := 0; i < 5; i++ {
		<-done
	}
}
