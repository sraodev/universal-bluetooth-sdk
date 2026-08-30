# Wire protocol — `common/protocol/`

The protocol is the **port** of the hexagon: every inbound adapter
(typed CLI, AI planner, MCP server, future microservices, future
language SDKs) and every outbound adapter (the daemon and its drivers)
agrees on the schema declared here.

## Files

| File | Purpose |
|---|---|
| [`v1.proto`](v1.proto) | Design counterpart for future gRPC stubs; current JSON types are hand-written in Go. |
| [`framing.md`](framing.md) | Wire format used today: 4-byte big-endian length prefix + JSON envelope. Documents methods, error codes, streaming semantics. |

## Status

- **v1 (current)** — length-prefixed JSON over Unix domain socket. No codegen,
  binary length prefix followed by a readable JSON body; use the CLI or a framed client.
- **v2 (planned)** — gRPC generated from `v1.proto`, alongside v1 during
  migration. Same method names, same error codes.

## Adding a new method

1. Add the message types and the `rpc` line to `v1.proto`.
2. Add the field-name twins to [`sdk/go/pkg/protocol/messages.go`](../../sdk/go/pkg/protocol/messages.go) and the method constant.
3. Document it in [`framing.md`](framing.md) (table of methods + any new error codes + payload examples).
4. Add a dispatcher case in [`cli/ubtd/server/dispatcher.go`](../../cli/ubtd/server/dispatcher.go).
5. Add or extend the matching transport-driver method in `sdk/go/pkg/transport/`.
6. Optionally register a tool wrapper in [`cli/ubtctl/ai/tools.go`](../../cli/ubtctl/ai/tools.go) — that change extends the AI planner and MCP server. Add a typed CLI command separately when needed.

## Versioning rules

- The wire protocol has a version returned by `Version`. Saved AI plans are currently unversioned JSON; a plan-version migration is separate work.
- Error codes are append-only: never repurpose an existing code; introduce a new one.
- Method names are append-only: never rename or remove a v1 method without going to v2.
