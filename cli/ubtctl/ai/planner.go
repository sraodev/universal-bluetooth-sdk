package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
)

// DefaultModel is Claude Opus 4.7 — most capable model for agentic planning.
// The SDK predates the constant for 4.7 but accepts the bare model ID.
const DefaultModel = "claude-opus-4-7"

// Plan configures one planner run.
type Plan struct {
	Goal       string       // user's natural-language request
	Model      string       // empty → DefaultModel
	MaxTokens  int64
	Mode       ExecMode
	SocketPath string       // surfaced in the system prompt for the model
	Daemon     *client.Client
	Out        io.Writer    // where final text + per-step notices go
}

// Run executes the plan against Claude with the daemon-backed tool set.
//
// The runner streams tokens as they arrive so the user sees progress on
// long planning steps. We rely on the SDK's BetaToolRunnerStreaming to
// handle the agentic loop (call → tool result → call → …) and only step
// in to surface text deltas to the user.
func Run(ctx context.Context, p Plan) error {
	if p.Goal == "" {
		return errors.New("plan: empty goal")
	}
	if p.Daemon == nil {
		return errors.New("plan: daemon client required")
	}
	if p.Out == nil {
		p.Out = os.Stdout
	}
	if p.Model == "" {
		p.Model = DefaultModel
	}
	if p.MaxTokens == 0 {
		// Generous ceiling — adaptive thinking + tool calls need headroom on Opus 4.7.
		p.MaxTokens = 16000
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return errors.New("ANTHROPIC_API_KEY is not set; export it or pass it via your secrets manager")
	}
	llm := anthropic.NewClient(option.WithAPIKey(apiKey))

	tools, err := BuildTools(p.Daemon, p.Mode)
	if err != nil {
		return fmt.Errorf("build tools: %w", err)
	}

	system := []anthropic.BetaTextBlockParam{{
		Text:         SystemPrompt(p.SocketPath, p.Mode),
		// Cache the system prompt + tool list — both are stable per binary,
		// so every subsequent `ubtctl ask` invocation reads the prefix
		// instead of paying for it. See shared/prompt-caching.md.
		CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
	}}

	runner := llm.Beta.Messages.NewToolRunnerStreaming(tools, anthropic.BetaToolRunnerParams{
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

	// AllStreaming yields one stream per agentic turn. We render only text
	// deltas — earlier turns are tool-use orchestration the runner handles
	// for us. Resetting `final` per turn keeps the assistant's last reply
	// (the one that does NOT trigger another tool call) as the output.
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
	return nil
}
