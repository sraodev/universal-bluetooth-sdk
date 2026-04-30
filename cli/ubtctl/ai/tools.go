package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/toolrunner"

	"github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client"
	"github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol"
)

// BuildTools returns the BetaTool set the planner exposes to Claude.
//
// Every tool funnels through `c` — the same daemon client a typed CLI
// command would use. That keeps the AI surface and the typed CLI on a
// single execution path: there is no second way to talk to the radio.
func BuildTools(c *client.Client, mode ExecMode) ([]anthropic.BetaTool, error) {
	tools := []anthropic.BetaTool{}

	add := func(t anthropic.BetaTool, err error) error {
		if err != nil {
			return err
		}
		tools = append(tools, t)
		return nil
	}

	if err := add(toolrunner.NewBetaToolFromJSONSchema(
		"ping_daemon",
		"Round-trip a Ping to ubtd. Useful as a liveness check or to estimate clock skew before doing anything time-sensitive.",
		func(ctx context.Context, _ struct{}) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			return callDaemon(ctx, c, protocol.MethodPing, nil)
		},
	)); err != nil {
		return nil, err
	}

	if err := add(toolrunner.NewBetaToolFromJSONSchema(
		"get_status",
		"Return ubtd's current state, active session count, and the list of registered transport drivers.",
		func(ctx context.Context, _ struct{}) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			return callDaemon(ctx, c, protocol.MethodStatus, nil)
		},
	)); err != nil {
		return nil, err
	}

	if err := add(toolrunner.NewBetaToolFromJSONSchema(
		"get_capabilities",
		"List per-transport capabilities (discover, pair, stream, max MTU) advertised by every driver currently registered with ubtd.",
		func(ctx context.Context, _ struct{}) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			return callDaemon(ctx, c, protocol.MethodCapabilities, nil)
		},
	)); err != nil {
		return nil, err
	}

	if err := add(toolrunner.NewBetaToolFromJSONSchema(
		"discover_devices",
		"Scan for nearby Bluetooth devices on the given transport and return everything seen until the timeout elapses.",
		func(ctx context.Context, in DiscoverInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			if in.TimeoutSeconds <= 0 {
				in.TimeoutSeconds = 5
			}
			devices := []protocol.Device{}
			err := c.Stream(ctx, protocol.MethodDiscover, protocol.DiscoverParams{
				Transport:      in.Transport,
				TimeoutSeconds: in.TimeoutSeconds,
			}, func(ev map[string]any) {
				var d protocol.Device
				if err := client.Decode(ev, &d); err == nil {
					devices = append(devices, d)
				}
			})
			if err != nil {
				return errorResult(err), nil
			}
			return jsonResult(map[string]any{"devices": devices, "count": len(devices)})
		},
	)); err != nil {
		return nil, err
	}

	if err := add(toolrunner.NewBetaToolFromJSONSchema(
		"send_payload",
		"MUTATING: deliver a UTF-8 payload to a peer. In dry-run mode this does not contact the daemon and returns a synthetic result instead.",
		func(ctx context.Context, in SendInput) (anthropic.BetaToolResultBlockParamContentUnion, error) {
			if mode == ExecModeDryRun {
				return jsonResult(map[string]any{
					"dry_run":   true,
					"would_send": map[string]any{
						"address":   in.Address,
						"transport": in.Transport,
						"bytes":     len(in.Payload),
					},
				})
			}
			return callDaemon(ctx, c, protocol.MethodSend, protocol.SendParams{
				Address:   in.Address,
				Transport: orDefault(in.Transport, "rfcomm"),
				Payload:   []byte(in.Payload),
				UUIDPort:  in.UUIDPort,
			})
		},
	)); err != nil {
		return nil, err
	}

	return tools, nil
}

// ---------------------------------------------------------------------------
// Tool input schemas (jsonschema tags drive the schema sent to Claude)
// ---------------------------------------------------------------------------

type DiscoverInput struct {
	Transport      string `json:"transport,omitempty" jsonschema:"description=Limit the scan to a single transport such as 'rfcomm' or 'ble'. Omit to use the daemon's default driver."`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"description=Scan duration in seconds. Defaults to 5 if omitted; never use values above 30."`
}

type SendInput struct {
	Address   string `json:"address" jsonschema:"required,description=Peer Bluetooth address (MAC for RFCOMM, UUID/handle for BLE)."`
	Transport string `json:"transport,omitempty" jsonschema:"description=Transport name. Defaults to 'rfcomm'."`
	Payload   string `json:"payload" jsonschema:"required,description=UTF-8 string to deliver to the peer."`
	UUIDPort  int    `json:"uuid_port,omitempty" jsonschema:"description=RFCOMM channel or BLE handle. 0 means driver default."`
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func callDaemon(ctx context.Context, c *client.Client, method string, params any) (anthropic.BetaToolResultBlockParamContentUnion, error) {
	res, err := c.Call(ctx, method, params)
	if err != nil {
		return errorResult(err), nil
	}
	return jsonResult(res)
}

func jsonResult(v any) (anthropic.BetaToolResultBlockParamContentUnion, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return errorResult(err), nil
	}
	return anthropic.BetaToolResultBlockParamContentUnion{
		OfText: &anthropic.BetaTextBlockParam{Text: string(b)},
	}, nil
}

func errorResult(err error) anthropic.BetaToolResultBlockParamContentUnion {
	return anthropic.BetaToolResultBlockParamContentUnion{
		OfText: &anthropic.BetaTextBlockParam{Text: fmt.Sprintf(`{"error":%q}`, err.Error())},
	}
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
