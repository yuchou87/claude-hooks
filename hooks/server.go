package hooks

import (
	"context"
	"io"
	"net/http"
	"time"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

// Server is the HTTP daemon that receives hook events from Claude Code.
// It binds to 127.0.0.1 only and holds POST /hook connections open
// while waiting for approval decisions.
type Server struct {
	mux      *http.ServeMux
	addr     string
	approver *Approver
	srv      *http.Server
	shutdown context.CancelFunc // cancels shutCtx → aborts in-flight dialogs
	shutCtx  context.Context
}

// NewServer creates a Server. addr must be "127.0.0.1:<port>".
func NewServer(addr string, approver *Approver) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		mux:      http.NewServeMux(),
		addr:     addr,
		approver: approver,
		shutdown: cancel,
		shutCtx:  ctx,
	}
	s.mux.HandleFunc("/hook", s.hookHandler)
	s.srv = &http.Server{
		Addr:        addr,
		Handler:     s.mux,
		ReadTimeout: 10 * time.Second,  // request body must arrive within 10s
		WriteTimeout: 60 * time.Second, // approval dialog can hold the response for up to 55s
	}
	return s
}

// Handler returns the HTTP handler. Use with httptest.NewServer in tests.
func (s *Server) Handler() http.Handler { return s.mux }

// ListenAndServe starts the HTTP server. Blocks until Shutdown or error.
func (s *Server) ListenAndServe() error { return s.srv.ListenAndServe() }

// Shutdown cancels all in-flight dialogs (they return defer) then drains HTTP.
func (s *Server) Shutdown(ctx context.Context) error {
	s.shutdown()               // signal dialogs to abort → they return nil (defer)
	return s.srv.Shutdown(ctx) // wait for active HTTP handlers + drain connections
}

func (s *Server) hookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusOK) // fail-open: oversized or read error
		return
	}

	// Parse once; on error fail-open.
	ev, parseErr := ParseInput(raw)
	if parseErr != nil {
		LogError(Input{}, "parse error", parseErr)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Local rules run first. If they decide (deny/allow), skip dialog.
	out := dispatchParsed(ev)

	if out != nil {
		NotifyCompletion(ev) // fire-and-forget, no-op for non-Stop events
		writeHookResponse(w, out)
		return
	}

	if ev.HookEventName == "PreToolUse" {
		// Pass shutCtx so SIGTERM cancels the dialog → returns defer
		out = s.approver.Handle(s.shutCtx, ev)
	}

	NotifyCompletion(ev) // fire-and-forget, no-op for non-Stop events
	writeHookResponse(w, out)
}

// writeHookResponse writes the JSON decision to the response.
// nil output → empty 200 (defer: Claude continues normally).
func writeHookResponse(w http.ResponseWriter, out *Output) {
	if out == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	b, err := out.JSON()
	if err != nil {
		w.WriteHeader(http.StatusOK) // fail-open
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
