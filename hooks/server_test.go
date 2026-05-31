package hooks_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yuchou87/claude-hooks/hooks"
)

// newTestServer creates a Server with a mock approver and returns an httptest server.
func newTestServer(t *testing.T, approver *hooks.Approver) (*hooks.Server, *httptest.Server) {
	t.Helper()
	srv := hooks.NewServer("127.0.0.1:0", approver)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func TestServer_LocalDenyRule_SkipsDialog(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	hooks.Register(hooks.Rule{
		Name:   "server-test-deny",
		Events: []string{"PreToolUse"},
		Run: func(in hooks.Input) *hooks.Output {
			if in.ToolName == "Bash" {
				return hooks.Deny("blocked by local rule")
			}
			return nil
		},
	})

	dialogCalled := false
	approver := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			dialogCalled = true
			return true, nil
		},
		Timeout: 2 * time.Second,
	}
	_, ts := newTestServer(t, approver)

	payload := `{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Bash","tool_input":{"command":"ls"}}`
	resp, err := http.Post(ts.URL+"/hook", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if dialogCalled {
		t.Error("dialog must not be called when local rule decides")
	}
	if !strings.Contains(string(body), "permissionDecision") {
		t.Errorf("want deny JSON in body, got %q", body)
	}
}

func TestServer_NilDispatch_CallsApprover(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	// No rules registered → dispatch returns nil → approver is called.

	approverCalled := false
	approver := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			approverCalled = true
			return true, nil // approve
		},
		Timeout: 2 * time.Second,
	}
	_, ts := newTestServer(t, approver)

	payload := `{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{"file_path":"/tmp/x"}}`
	resp, err := http.Post(ts.URL+"/hook", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if !approverCalled {
		t.Error("approver must be called when dispatch returns nil for PreToolUse")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
}

func TestServer_NonPreToolUse_SkipsApprover(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	approver := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			return true, nil
		},
		Timeout: 2 * time.Second,
	}
	_, ts := newTestServer(t, approver)

	// SessionEnd → no rules, no approval → empty 200
	payload := `{"hook_event_name":"SessionEnd","session_id":"s","transcript_path":"/t","cwd":"/"}`
	resp, err := http.Post(ts.URL+"/hook", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", resp.StatusCode)
	}
	// nil decision → empty body
	if strings.TrimSpace(string(body)) != "" {
		t.Errorf("non-PreToolUse with nil dispatch must return empty body, got %q", body)
	}
}

func TestServer_BadJSON_Returns200_EmptyBody(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)
	approver := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			return false, nil
		},
		Timeout: 2 * time.Second,
	}
	_, ts := newTestServer(t, approver)

	resp, err := http.Post(ts.URL+"/hook", "application/json", strings.NewReader("{bad json"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("want 200 (fail-open), got %d", resp.StatusCode)
	}
	if strings.TrimSpace(string(body)) != "" {
		t.Errorf("bad JSON must return empty body (fail-open), got %q", body)
	}
}

func TestServer_WrongMethod_Returns405(t *testing.T) {
	approver := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) { return true, nil },
		Timeout:  2 * time.Second,
	}
	_, ts := newTestServer(t, approver)

	resp, err := http.Get(ts.URL + "/hook")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", resp.StatusCode)
	}
}

func TestServer_Shutdown_CancelsInFlightApproval(t *testing.T) {
	t.Cleanup(hooks.ResetRegistryForTest)

	dialogStarted := make(chan struct{})
	approver := &hooks.Approver{
		DialogFn: func(ctx context.Context, ev hooks.Input) (bool, error) {
			close(dialogStarted)
			<-ctx.Done() // blocks until shutdown cancels
			return false, ctx.Err()
		},
		Timeout: 30 * time.Second,
	}
	srv, ts := newTestServer(t, approver)

	// Start a long-running approval request
	reqDone := make(chan *http.Response, 1)
	go func() {
		payload := `{"hook_event_name":"PreToolUse","session_id":"s","transcript_path":"/t","cwd":"/","tool_name":"Write","tool_input":{}}`
		resp, _ := http.Post(ts.URL+"/hook", "application/json", strings.NewReader(payload))
		reqDone <- resp
	}()

	<-dialogStarted // dialog goroutine is now running

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown error: %v", err)
	}

	select {
	case resp := <-reqDone:
		if resp != nil {
			resp.Body.Close()
		}
		// Success: request completed after shutdown
	case <-time.After(5 * time.Second):
		t.Error("in-flight request did not complete after Shutdown")
	}
}
