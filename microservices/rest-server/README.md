# REST server (planned)

Sibling of [`microservices/grpc-server`](../grpc-server) — same role,
different transport. Exposes `ubtd` to web and mobile clients that
prefer plain HTTP/JSON over gRPC.

## Where it fits

```
browser / mobile ── HTTPS ──►  microservices/rest-server  ── UDS ──►  ubtd
                                  (this directory)
```

Yet another **inbound adapter**. The same daemon that the typed CLI, the
AI planner, and the MCP server talk to.

## When this ships

After the gRPC server, since two of the three implementation paths reuse
gRPC artefacts:

| Path | Effort | Notes |
|---|---|---|
| Hand-written REST controller | Lowest | Calls back into `ubtd` over UDS using the same client the CLI uses. |
| `grpc-gateway` in front of `microservices/grpc-server` | Medium | Auto-generated REST routes from `common/protocol/v1.proto`. |
| OpenAPI document generated from `v1.proto` + a Go HTTP handler | Medium | Provides client codegen for web/mobile out of the box. |

## Authentication / policy

Owned by this service, not the daemon. Recommended baseline:

- TLS on the public port; mTLS optional.
- Bearer-token auth (RBAC mapped to per-tool policy).
- Reuse the same `mutating: true` flag the [tool registry](../../cli/ubtctl/tools/) carries today as the gate for write operations.
- Emit access logs to whatever the deploy environment captures (stdout JSON via slog is the obvious default).

Status: **planned**. The hex architecture means this can ship without
touching the daemon at all.
