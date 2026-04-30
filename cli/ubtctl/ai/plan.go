package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/tools"
)

// PlanFormatVersion is the schema version of the plan JSON file. Bump it
// when the Plan / Step shape changes; replay refuses unknown versions.
const PlanFormatVersion = 1

// Plan is the persistent record of a single AI run: the goal, the
// execution mode, and the ordered list of tool calls Claude actually
// made (with their arguments and observed results). Saved to disk by
// `ubtctl ask --save`, replayed by `ubtctl plan run`.
type Plan struct {
	FormatVersion int       `json:"format_version"`
	Goal          string    `json:"goal"`
	Mode          string    `json:"mode"`
	Model         string    `json:"model,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	Steps         []Step    `json:"steps"`
}

// Step records one tool invocation: which tool, what arguments,
// and what the tool returned at record time. Mutating distinguishes
// steps that have side-effects on real hardware from read-only ones.
type Step struct {
	Tool      string          `json:"tool"`
	Mutating  bool            `json:"mutating"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	Result    string          `json:"result,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	DurationMs int64          `json:"duration_ms,omitempty"`
}

// recorder accumulates Steps while Spec handlers run.
type recorder struct {
	mu    sync.Mutex
	steps []Step
}

func (r *recorder) record(s Step) {
	r.mu.Lock()
	r.steps = append(r.steps, s)
	r.mu.Unlock()
}

func (r *recorder) snapshot() []Step {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Step, len(r.steps))
	copy(out, r.steps)
	return out
}

// wrapWithRecorder returns a new Registry where every Spec's Handler is
// instrumented to append a Step to rec before delegating to the original
// handler. The original Registry is left untouched so AI runs and MCP
// callers don't accidentally share recording state.
func wrapWithRecorder(reg *tools.Registry, rec *recorder) *tools.Registry {
	out := tools.NewRegistry()
	for _, s := range reg.All() {
		spec := s
		original := spec.Handler
		spec.Handler = func(ctx context.Context, raw json.RawMessage) (tools.Result, error) {
			start := time.Now()
			res, err := original(ctx, raw)
			rec.record(Step{
				Tool:       spec.Name,
				Mutating:   spec.Mutating,
				Arguments:  cloneRawMessage(raw),
				Result:     res.Text,
				IsError:    res.IsError,
				DurationMs: time.Since(start).Milliseconds(),
			})
			return res, err
		}
		out.Add(spec)
	}
	return out
}

func cloneRawMessage(r json.RawMessage) json.RawMessage {
	if len(r) == 0 {
		return nil
	}
	c := make(json.RawMessage, len(r))
	copy(c, r)
	return c
}

// SavePlan writes a plan to path as pretty-printed JSON, with mode 0o600
// (plans may contain payloads or device addresses worth keeping private).
func SavePlan(path string, p *Plan) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// LoadPlan reads a plan and validates its FormatVersion.
func LoadPlan(path string) (*Plan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plan
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if p.FormatVersion != PlanFormatVersion {
		return nil, fmt.Errorf("plan format v%d not supported (this build expects v%d)", p.FormatVersion, PlanFormatVersion)
	}
	return &p, nil
}

// ReplayOptions configures plan execution.
type ReplayOptions struct {
	// AllowMutating must be true to execute steps tagged Mutating. Otherwise
	// such steps abort the run with ErrMutatingNotAllowed.
	AllowMutating bool
	// DryRun prints what would run but skips every Handler call.
	DryRun bool
	// Out receives a one-line summary per step ("[i] tool -> result").
	Out io.Writer
}

// ErrMutatingNotAllowed is returned by Replay when a plan contains a
// mutating step and AllowMutating is false.
var ErrMutatingNotAllowed = errors.New("plan contains mutating steps; pass --yes to confirm")

// Replay executes the plan's steps against reg in order, writing one
// summary line per step to opts.Out.
func Replay(ctx context.Context, p *Plan, reg *tools.Registry, opts ReplayOptions) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	// Pre-flight: refuse to start if any step is mutating and the caller
	// hasn't explicitly opted in. Fail fast — don't run half a plan.
	if !opts.AllowMutating && !opts.DryRun {
		for _, s := range p.Steps {
			if s.Mutating {
				return ErrMutatingNotAllowed
			}
		}
	}

	for i, step := range p.Steps {
		spec, ok := reg.Get(step.Tool)
		if !ok {
			return fmt.Errorf("step %d: tool %q not registered in this build", i, step.Tool)
		}
		if opts.DryRun {
			fmt.Fprintf(opts.Out, "[%d] %s (dry-run) args=%s\n", i, step.Tool, compactJSON(step.Arguments))
			continue
		}
		res, err := spec.Handler(ctx, step.Arguments)
		if err != nil {
			return fmt.Errorf("step %d (%s): %w", i, step.Tool, err)
		}
		marker := "ok"
		if res.IsError {
			marker = "err"
		}
		fmt.Fprintf(opts.Out, "[%d] %s [%s] %s\n", i, step.Tool, marker, res.Text)
	}
	return nil
}

func compactJSON(r json.RawMessage) string {
	if len(r) == 0 {
		return "{}"
	}
	return string(r)
}
