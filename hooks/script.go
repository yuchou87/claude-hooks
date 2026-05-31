package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

const scriptTimeout = 100 * time.Millisecond

// ScriptEngine compiles a TS/JS script once and creates a fresh goja.Runtime
// for each event call (safe for concurrent use, correct isolation).
type ScriptEngine struct {
	path   string
	prog   *goja.Program
	events []string
}

// NewScriptEngine reads, esbuild-transforms, and compiles a TS/JS script.
// Extracts the exported `events` array for rule registration.
// Returns error on transform/compile failure or missing `events` export.
func NewScriptEngine(scriptPath string) (*ScriptEngine, error) {
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("read script %s: %w", scriptPath, err)
	}

	loader := api.LoaderJS
	switch strings.ToLower(filepath.Ext(scriptPath)) {
	case ".ts", ".tsx":
		loader = api.LoaderTS
	case ".jsx":
		loader = api.LoaderJSX
	}

	result := api.Transform(string(src), api.TransformOptions{
		Loader: loader,
		Format: api.FormatCommonJS,
		Target: api.ES2017,
	})
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("esbuild %s: %s", scriptPath, result.Errors[0].Text)
	}

	prog, err := goja.Compile(scriptPath, string(result.Code), false)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", scriptPath, err)
	}

	events, err := extractScriptEvents(prog)
	if err != nil {
		return nil, fmt.Errorf("events from %s: %w", scriptPath, err)
	}

	return &ScriptEngine{
		path:   scriptPath,
		prog:   prog,
		events: events,
	}, nil
}

// AsRule wraps the ScriptEngine as a hooks.Rule for registration via StoreDynamic.
func (e *ScriptEngine) AsRule() Rule {
	return Rule{
		Name:   "script:" + filepath.Base(e.path),
		Events: e.events,
		Run: func(ev Input) *Output {
			return e.run(context.Background(), ev)
		},
	}
}

// run executes the script's decide() function in a fresh runtime.
// Returns nil (fail-open) on timeout, throw, or unexpected output.
func (e *ScriptEngine) run(_ context.Context, ev Input) (out *Output) {
	rt := goja.New()
	setupCJS(rt)

	// Hard 100ms timeout via Interrupt.
	timer := time.AfterFunc(scriptTimeout, func() {
		rt.Interrupt("script timeout")
	})
	defer timer.Stop()

	defer func() {
		if rec := recover(); rec != nil {
			out = nil // fail-open on panic
		}
	}()

	if _, err := rt.RunProgram(e.prog); err != nil {
		LogError(Input{}, "script load: "+e.path, err)
		return nil // fail-open
	}

	exportsObj := getModuleExports(rt)
	decide, ok := goja.AssertFunction(exportsObj.Get("decide"))
	if !ok {
		LogError(Input{}, "script missing decide(): "+e.path, nil)
		return nil // fail-open
	}

	// Pass event as plain JS object (via JSON round-trip for clean conversion).
	evJSON, _ := json.Marshal(ev)
	var evMap map[string]any
	if err := json.Unmarshal(evJSON, &evMap); err != nil {
		LogError(ev, "marshal event: "+e.path, err)
		return nil // fail-open
	}
	// Merge Extra fields so scripts see unknown future Claude Code event fields.
	for k, v := range ev.Extra {
		if _, exists := evMap[k]; !exists {
			evMap[k] = v
		}
	}
	evVal := rt.ToValue(evMap)

	result, err := decide(goja.Undefined(), evVal)
	if err != nil {
		LogError(ev, "script decide() error: "+e.path, err)
		return nil // fail-open
	}
	if goja.IsNull(result) || goja.IsUndefined(result) {
		return nil // no opinion
	}

	return parseScriptResult(result.Export())
}

// parseScriptResult maps the script's return value to an *Output.
// Handles {permissionDecision: "deny"|"allow", permissionDecisionReason: "..."}.
func parseScriptResult(v any) *Output {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	decision, _ := m["permissionDecision"].(string)
	reason, _ := m["permissionDecisionReason"].(string)
	switch decision {
	case "deny":
		return Deny(reason)
	case "allow":
		return Allow()
	default:
		return nil
	}
}

// extractScriptEvents runs the compiled program in a throw-away runtime
// and reads the exported `events` string array.
func extractScriptEvents(prog *goja.Program) ([]string, error) {
	rt := goja.New()
	setupCJS(rt)

	timer := time.AfterFunc(scriptTimeout, func() {
		rt.Interrupt("script timeout")
	})
	defer timer.Stop()

	if _, err := rt.RunProgram(prog); err != nil {
		return nil, fmt.Errorf("run: %w", err)
	}
	// esbuild CJS output does `module.exports = __toCommonJS(...)`, which
	// replaces module.exports. Read it back via module.exports, not the
	// original exports variable we installed.
	eventsVal := getModuleExports(rt).Get("events")
	if eventsVal == nil || goja.IsUndefined(eventsVal) || goja.IsNull(eventsVal) {
		return nil, fmt.Errorf("script must export 'events' array")
	}
	var events []string
	if err := rt.ExportTo(eventsVal, &events); err != nil {
		return nil, fmt.Errorf("'events' must be a string array: %w", err)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("'events' array must not be empty")
	}
	return events, nil
}

// getModuleExports returns the current module.exports object after script execution.
// esbuild replaces module.exports with a new object, so we must read it back.
func getModuleExports(rt *goja.Runtime) *goja.Object {
	moduleVal := rt.Get("module")
	if moduleVal == nil || goja.IsUndefined(moduleVal) || goja.IsNull(moduleVal) {
		return rt.NewObject()
	}
	moduleObj := moduleVal.ToObject(rt)
	exportsVal := moduleObj.Get("exports")
	if exportsVal == nil || goja.IsUndefined(exportsVal) || goja.IsNull(exportsVal) {
		return rt.NewObject()
	}
	return exportsVal.ToObject(rt)
}

// setupCJS installs CommonJS shims (module, exports, require) into a runtime.
// Required before running esbuild CJS-formatted output.
func setupCJS(rt *goja.Runtime) {
	moduleObj := rt.NewObject()
	exportsObj := rt.NewObject()
	moduleObj.Set("exports", exportsObj) //nolint:errcheck
	rt.Set("module", moduleObj)          //nolint:errcheck
	rt.Set("exports", exportsObj)        //nolint:errcheck
	rt.Set("require", func(goja.FunctionCall) goja.Value {
		return goja.Undefined()
	}) //nolint:errcheck
}
