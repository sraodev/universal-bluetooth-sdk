package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/tools"
)

// Plan is the persistent record of a single AI run: goal, execution
// mode, and the ordered list of tool calls Claude actually made.
// Saved by `ubtctl ask --save`, replayed by `ubtctl plan run`.
type Plan struct {
	Goal      string    `json:"goal"`
	Mode      string    `json:"mode"`
	Model     string    `json:"model,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Steps     []Step    `json:"steps"`
}

// Step records one tool invocation: which tool, what arguments, what
// the tool returned at record time. Mutating distinguishes steps with
// side effects on real hardware from read-only ones.
type Step struct {
	Tool       string          `json:"tool"`
	Mutating   bool            `json:"mutating"`
	Arguments  json.RawMessage `json:"arguments,omitempty"`
	Result     string          `json:"result,omitempty"`
	IsError    bool            `json:"is_error,omitempty"`
	DurationMs int64           `json:"duration_ms,omitempty"`
}

// recorder accumulates Steps as Spec handlers run. The BetaToolRunner
// invokes tools sequentially, so no synchronisation is needed.
type recorder struct {
	steps []Step
}

// wrapWithRecorder returns a new Registry where every Spec's Handler
// records a Step into rec before delegating to the original handler.
func wrapWithRecorder(reg *tools.Registry, rec *recorder) *tools.Registry {
	out := tools.NewRegistry()
	for _, s := range reg.All() {
		spec := s
		original := spec.Handler
		spec.Handler = func(ctx context.Context, raw json.RawMessage) (tools.Result, error) {
			start := time.Now()
			res, err := original(ctx, raw)
			rec.steps = append(rec.steps, Step{
				Tool:       spec.Name,
				Mutating:   spec.Mutating,
				Arguments:  raw,
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

// SavePlan writes a plan to path as pretty-printed JSON, with mode 0o600
// — plans may carry payloads or device addresses worth keeping private.
func SavePlan(path string, p *Plan) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// LoadPlan reads a plan from disk.
func LoadPlan(path string) (*Plan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plan
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &p, nil
}

// ReplayOptions configures plan execution.
type ReplayOptions struct {
	// AllowMutating must be true to execute tools marked Mutating by the registry.
	// Otherwise such steps abort the run with ErrMutatingNotAllowed.
	AllowMutating bool
	DryRun        bool
	Out           io.Writer
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
	// The registry is authoritative: saved plans are editable input.
	// Validate the entire plan before executing any step.
	for i, step := range p.Steps {
		spec, ok := reg.Get(step.Tool)
		if !ok {
			return fmt.Errorf("step %d: tool %q not registered in this build", i, step.Tool)
		}
		if (spec.Mutating || step.Mutating) && !opts.AllowMutating && !opts.DryRun {
			return ErrMutatingNotAllowed
		}
	}

	for i, step := range p.Steps {
		spec, _ := reg.Get(step.Tool) // validated during preflight
		args := string(step.Arguments)
		if args == "" {
			args = "{}"
		}
		if opts.DryRun {
			fmt.Fprintf(opts.Out, "[%d] %s (dry-run) args=%s\n", i, step.Tool, args)
			continue
		}
		res, err := spec.Handler(ctx, step.Arguments)
		if err != nil {
			return fmt.Errorf("step %d (%s): %w", i, step.Tool, err)
		}
		if res.IsError {
			return fmt.Errorf("step %d (%s): %s", i, step.Tool, res.Text)
		}
		fmt.Fprintf(opts.Out, "[%d] %s [ok] %s\n", i, step.Tool, res.Text)
	}
	return nil
}
