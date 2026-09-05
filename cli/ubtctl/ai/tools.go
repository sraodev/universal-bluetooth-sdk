package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/toolrunner"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/tools"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

// BuildSpecs returns the tool registry ubt exposes to any tool-using
// front-end (the Anthropic planner, the MCP server, etc).
//
// Every spec funnels through `c` — the same daemon client a typed CLI
// command would use. dryRun=true stubs the only mutating tool
// (send_payload) so the model can finish planning without side effects.
func BuildSpecs(c *client.Client, dryRun bool) *tools.Registry {
	send := tools.New[SendInput](
		"send_payload",
		"MUTATING: deliver a UTF-8 payload to a peer. In dry-run mode this does not contact the daemon and returns a synthetic result instead.",
		func(ctx context.Context, in SendInput) (tools.Result, error) {
			if dryRun {
				return jsonResult(map[string]any{
					"dry_run": true,
					"would_send": map[string]any{
						"address":   in.Address,
						"transport": in.Transport,
						"bytes":     len(in.Payload),
					},
				})
			}
			return rpcResult(c.Call(ctx, protocol.MethodSend, protocol.SendParams{
				Address:   in.Address,
				Transport: orDefault(in.Transport, "rfcomm"),
				Payload:   []byte(in.Payload),
				UUIDPort:  in.UUIDPort,
			}))
		},
	)
	send.Mutating = true

	reg := tools.NewRegistry()
	reg.Add(tools.New[struct{}](
		"ping_daemon",
		"Round-trip a Ping to ubtd. Useful as a liveness check or to estimate clock skew before doing anything time-sensitive.",
		func(ctx context.Context, _ struct{}) (tools.Result, error) {
			return rpcResult(c.Call(ctx, protocol.MethodPing, nil))
		},
	))
	reg.Add(tools.New[struct{}](
		"get_status",
		"Return ubtd's current state, active session count, and the list of registered transport drivers.",
		func(ctx context.Context, _ struct{}) (tools.Result, error) {
			return rpcResult(c.Call(ctx, protocol.MethodStatus, nil))
		},
	))
	reg.Add(tools.New[struct{}](
		"get_capabilities",
		"List per-transport capabilities (discover, pair, stream, max MTU) advertised by every driver currently registered with ubtd.",
		func(ctx context.Context, _ struct{}) (tools.Result, error) {
			return rpcResult(c.Call(ctx, protocol.MethodCapabilities, nil))
		},
	))
	reg.Add(tools.New[DiscoverInput](
		"discover_devices",
		"Scan for nearby Bluetooth devices on the given transport and return everything seen until the timeout elapses.",
		func(ctx context.Context, in DiscoverInput) (tools.Result, error) {
			if in.TimeoutSeconds <= 0 {
				in.TimeoutSeconds = 5
			}
			devices := []protocol.Device{}
			err := c.Stream(ctx, protocol.MethodDiscover, protocol.DiscoverParams{
				Transport:      in.Transport,
				TimeoutSeconds: in.TimeoutSeconds,
			}, func(ev map[string]any) {
				var d protocol.Device
				if e := client.Decode(ev, &d); e == nil {
					devices = append(devices, d)
				}
			})
			if err != nil {
				return errResult(err), nil
			}
			return jsonResult(map[string]any{"devices": devices, "count": len(devices)})
		},
	))
	reg.Add(send)
	return reg
}

// AsBetaTools adapts a tools.Registry into the SDK's BetaTool slice so
// the BetaToolRunner can invoke the same handlers Claude-via-MCP would.
func AsBetaTools(reg *tools.Registry) ([]anthropic.BetaTool, error) {
	out := make([]anthropic.BetaTool, 0, len(reg.All()))
	for _, s := range reg.All() {
		spec := s
		t, err := toolrunner.NewBetaToolFromBytes[json.RawMessage](
			spec.Name, spec.Description, spec.Schema,
			func(ctx context.Context, raw json.RawMessage) (anthropic.BetaToolResultBlockParamContentUnion, error) {
				res, err := spec.Handler(ctx, raw)
				if err != nil {
					return anthropic.BetaToolResultBlockParamContentUnion{
						OfText: &anthropic.BetaTextBlockParam{Text: err.Error()},
					}, nil
				}
				return anthropic.BetaToolResultBlockParamContentUnion{
					OfText: &anthropic.BetaTextBlockParam{Text: res.Text},
				}, nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf("adapt %s: %w", spec.Name, err)
		}
		out = append(out, t)
	}
	return out, nil
}

// Tool input schemas. Note: invopop/jsonschema parses commas inside
// `jsonschema:"description=..."` as tag separators and silently
// truncates everything after the first comma. Use semicolons or
// slashes instead.

type DiscoverInput struct {
	Transport      string `json:"transport,omitempty" jsonschema:"description=Limit the scan to a single transport such as 'rfcomm' or 'ble'; omit to use the daemon's default driver."`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"description=Scan duration in seconds; defaults to 5 if omitted; never use values above 30."`
}

type SendInput struct {
	Address   string `json:"address" jsonschema:"required,description=Peer Bluetooth address: MAC for RFCOMM / UUID or handle for BLE."`
	Transport string `json:"transport,omitempty" jsonschema:"description=Transport name; defaults to 'rfcomm'."`
	Payload   string `json:"payload" jsonschema:"required,description=UTF-8 string to deliver to the peer."`
	UUIDPort  int    `json:"uuid_port,omitempty" jsonschema:"description=RFCOMM channel or BLE handle; 0 means driver default."`
}

func rpcResult(payload map[string]any, err error) (tools.Result, error) {
	if err != nil {
		return errResult(err), nil
	}
	return jsonResult(payload)
}

func jsonResult(v any) (tools.Result, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return errResult(err), nil
	}
	return tools.Result{Text: string(b)}, nil
}

func errResult(err error) tools.Result {
	return tools.Result{
		Text:    fmt.Sprintf(`{"error":%q}`, err.Error()),
		IsError: true,
	}
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
