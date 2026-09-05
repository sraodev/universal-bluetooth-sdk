package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
)

// DefaultModel is Claude Opus 4.7. SDK predates the constant for 4.7
// but accepts the bare model ID.
const DefaultModel = "claude-opus-4-7"

// RunConfig configures one planner run.
type RunConfig struct {
	ProgramName string
	Goal        string
	Model       string // empty → DefaultModel
	MaxTokens   int64
	DryRun      bool
	SocketPath  string
	Daemon      *client.Client
	Out         io.Writer
	// SavePath, if non-empty, captures every tool call into a Plan
	// and writes it to disk after the run completes.
	SavePath string
}

// Run executes the plan against Claude with the daemon-backed tool set.
func Run(ctx context.Context, p RunConfig) error {
	if p.Goal == "" {
		return errors.New("plan: empty goal")
	}
	if p.Daemon == nil {
		return errors.New("plan: daemon client required")
	}
	if p.Out == nil {
		p.Out = os.Stdout
	}
	if p.ProgramName == "" {
		p.ProgramName = "ubt"
	}
	if p.Model == "" {
		p.Model = DefaultModel
	}
	if p.MaxTokens == 0 {
		// Adaptive thinking + tool calls need headroom on Opus 4.7.
		p.MaxTokens = 16000
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return errors.New("ANTHROPIC_API_KEY is not set; export it or pass it via your secrets manager")
	}
	llm := anthropic.NewClient(option.WithAPIKey(apiKey))

	specs := BuildSpecs(p.Daemon, p.DryRun)
	var rec *recorder
	if p.SavePath != "" {
		rec = &recorder{}
		specs = wrapWithRecorder(specs, rec)
	}
	beta, err := AsBetaTools(specs)
	if err != nil {
		return fmt.Errorf("adapt tools: %w", err)
	}

	system := []anthropic.BetaTextBlockParam{{
		Text: SystemPromptFor(p.ProgramName, p.SocketPath, p.DryRun),
		// Cache the system prompt + tool list — both are stable per
		// binary, so subsequent invocations read the prefix instead
		// of paying for it. See shared/prompt-caching.md.
		CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
	}}

	runner := llm.Beta.Messages.NewToolRunnerStreaming(beta, anthropic.BetaToolRunnerParams{
		BetaMessageNewParams: anthropic.BetaMessageNewParams{
			Model:     anthropic.Model(p.Model),
			MaxTokens: p.MaxTokens,
			System:    system,
			Thinking: anthropic.BetaThinkingConfigParamUnion{
				OfAdaptive: &anthropic.BetaThinkingConfigAdaptiveParam{},
			},
			Messages: []anthropic.BetaMessageParam{
				anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(p.Goal)),
			},
		},
		MaxIterations: 8,
	})

	// AllStreaming yields one stream per agentic turn. Resetting `final`
	// per turn keeps the assistant's last reply (the one that does NOT
	// trigger another tool call) as the rendered output.
	final := strings.Builder{}
	for events, err := range runner.AllStreaming(ctx) {
		if err != nil {
			return fmt.Errorf("runner: %w", err)
		}
		final.Reset()
		for event, err := range events {
			if err != nil {
				return fmt.Errorf("stream: %w", err)
			}
			if delta, ok := event.AsAny().(anthropic.BetaRawContentBlockDeltaEvent); ok {
				if td, ok := delta.Delta.AsAny().(anthropic.BetaTextDelta); ok {
					final.WriteString(td.Text)
				}
			}
		}
	}
	if err := runner.Err(); err != nil {
		return fmt.Errorf("runner: %w", err)
	}

	out := strings.TrimRight(final.String(), "\n")
	if out != "" {
		fmt.Fprintln(p.Out, out)
	}
	if last := runner.LastMessage(); last != nil {
		u := last.Usage
		fmt.Fprintf(p.Out, "\n[%d input · %d output · %d cache-read · %d iterations]\n",
			u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, runner.IterationCount())
	}

	if rec != nil {
		mode := "execute"
		if p.DryRun {
			mode = "dry-run"
		}
		plan := &Plan{
			Goal:      p.Goal,
			Mode:      mode,
			Model:     p.Model,
			CreatedAt: time.Now().UTC(),
			Steps:     rec.steps,
		}
		if err := SavePlan(p.SavePath, plan); err != nil {
			return fmt.Errorf("save plan: %w", err)
		}
		fmt.Fprintf(p.Out, "saved plan: %s (%d steps)\n", p.SavePath, len(plan.Steps))
	}
	return nil
}
