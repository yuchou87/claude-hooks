package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// MODE controls what decision to send for PermissionRequest.
// Round 1: "deny"  — verify Claude waits + refuses the tool.
// Round 2: "allow" — verify Claude waits + executes the tool without popup.
// Change this constant and restart the server to switch rounds.
const MODE = "deny"

type hookEvent struct {
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolInput     any    `json:"tool_input"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("ERROR read body: %v", err)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
		return
	}

	var ev hookEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		// fail-open: bad JSON → no decision, Claude continues normally
		log.Printf("ERROR parse JSON: %v", err)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
		return
	}

	if ev.HookEventName != "PermissionRequest" {
		// fail-open: unrelated event → no decision
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}")
		return
	}

	recvAt := time.Now()
	log.Printf("RECV  event=%s  tool=%s  recv_at=%s",
		ev.HookEventName, ev.ToolName, recvAt.Format(time.RFC3339Nano))

	time.Sleep(5 * time.Second)

	sendAt := time.Now()
	elapsed := sendAt.Sub(recvAt).Round(time.Millisecond)
	log.Printf("SEND  behavior=%s  send_at=%s  elapsed=%s",
		MODE, sendAt.Format(time.RFC3339Nano), elapsed)

	resp := map[string]any{
		"hookSpecificOutput": map[string]any{
			"decision": map[string]any{
				"behavior": MODE,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("ERROR encode response: %v", err)
	}
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	addr := "127.0.0.1:8787"
	http.HandleFunc("/hook", handler)
	log.Printf("Spike server starting  addr=%s  mode=%s", addr, MODE)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
