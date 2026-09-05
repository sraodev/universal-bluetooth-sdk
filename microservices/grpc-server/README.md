# gRPC server (planned)

A future **remote facade** in front of `ubtd`. The hexagonal architecture
already has every primitive this service needs — same wire protocol,
same tool registry — so this directory becomes a thin gRPC translator
once the v2 wire ships.

## Where it fits

```
remote client ── gRPC ──►  microservices/grpc-server  ── UDS ──►  ubtd
                              (this directory)
```

The microservice is **another inbound adapter**. It does not reach into
hardware; it reaches into `ubtd` exactly the way `ubt` does today.

## When this ships

Right after the **v2 gRPC wire** lands (see the
[roadmap](../../README.md#roadmap)). The work plan:

1. Generate Go server stubs from [`common/protocol/v1.proto`](../../common/protocol/v1.proto).
2. Implement them by translating each request to a length-prefixed JSON call into `ubtd` (or, once v2 lands, by passing the request through directly).
3. Reuse the existing tool registry to expose the AI planner verbs (`Ask`, `PlanRun`) as RPCs.
4. Ship deployment recipes: `Dockerfile`, `systemd` unit, K8s manifest.

## What it is *not*

- It does not own the radio. Only `ubtd` does.
- It does not reimplement Bluetooth logic — it calls back into the daemon.
- It does not own state. Sessions, plans, audit all live in `ubtd`.

Status: **planned**. Open an issue if you want to bootstrap it.
