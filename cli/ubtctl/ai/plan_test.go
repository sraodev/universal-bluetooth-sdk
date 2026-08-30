package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/tools"
)

func TestReplayPreflight(t *testing.T) {
	for _, tc := range []struct {
		name       string
		steps      []Step
		allow, dry bool
		wantCalls  int
		wantError  bool
	}{
		{name: "tampered mutation flag", steps: []Step{{Tool: "read"}, {Tool: "send", Mutating: false}}, wantError: true},
		{name: "unknown later tool", steps: []Step{{Tool: "read"}, {Tool: "missing"}}, wantError: true},
		{name: "explicit consent", steps: []Step{{Tool: "send"}}, allow: true, wantCalls: 1},
		{name: "offline preview", steps: []Step{{Tool: "send"}}, dry: true},
		{name: "read only", steps: []Step{{Tool: "read"}}, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			handler := func(context.Context, json.RawMessage) (tools.Result, error) {
				calls++
				return tools.Result{Text: "ok"}, nil
			}
			reg := tools.NewRegistry()
			reg.Add(tools.Spec{Name: "read", Handler: handler})
			reg.Add(tools.Spec{Name: "send", Handler: handler, Mutating: true})
			err := Replay(context.Background(), &Plan{Steps: tc.steps}, reg, ReplayOptions{AllowMutating: tc.allow, DryRun: tc.dry, Out: io.Discard})
			if (err != nil) != tc.wantError || calls != tc.wantCalls {
				t.Fatalf("err=%v calls=%d; want error=%v calls=%d", err, calls, tc.wantError, tc.wantCalls)
			}
			if tc.name == "tampered mutation flag" && !errors.Is(err, ErrMutatingNotAllowed) {
				t.Fatalf("wrong error: %v", err)
			}
		})
	}
}

func TestReplayStopsOnToolFailure(t *testing.T) {
	calls := 0
	reg := tools.NewRegistry()
	reg.Add(tools.Spec{Name: "fail", Handler: func(context.Context, json.RawMessage) (tools.Result, error) {
		calls++
		return tools.Result{Text: "peer unavailable", IsError: true}, nil
	}})
	err := Replay(context.Background(), &Plan{Steps: []Step{{Tool: "fail"}, {Tool: "fail"}}}, reg, ReplayOptions{Out: io.Discard})
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
