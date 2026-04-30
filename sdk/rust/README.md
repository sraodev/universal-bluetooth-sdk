# Rust SDK (planned)

Reserved for a Rust implementation of the same wire protocol the Go
daemon and Python SDK already speak. The shape is identical — same
methods, same error codes — so a Rust client/server is **another set
of inbound or outbound adapters**, not a fork.

## Where it fits

Two ways a Rust crate plugs into the architecture:

1. **As a client (inbound adapter).** A `ubt-client` crate that dials
   `ubtd` over UDS and exposes the methods from
   [`common/protocol/v1.proto`](../../common/protocol/v1.proto) as
   strongly-typed async functions. Useful for embedded gateways and
   for any Rust app that wants to drive a daemon running elsewhere.
2. **As a driver (outbound adapter).** Eventually, an alternative
   daemon or a constrained-platform implementation might want a Rust
   driver for the BlueZ or WinRT side. Those land alongside the Go
   `transport.Driver` implementations under their own `sdk/rust/transport/`.

The point is that *neither* role requires reimplementing the
**hexagon** — only the protocol bindings on whichever side they sit.

## Anticipated workspace layout

```
sdk/rust/
├── Cargo.toml          # workspace manifest
├── crates/
│   ├── ubt-protocol/   # generated wire types (Serde) + length-prefix codec
│   ├── ubt-client/     # async client over UDS / TCP
│   └── ubt-driver/     # transport-driver scaffolding (Linux / macOS / Windows)
└── examples/
    └── ping/           # smallest end-to-end demo
```

## Required upstream pieces

- The wire protocol is stable enough to bind against (it is, today, for v1).
- gRPC v2 (roadmap) lets Rust generate stubs from the same `.proto` the
  Go daemon will use, dropping the hand-rolled length-prefix codec.

Status: **planned**. Open an issue to coordinate; the initial scope
should be `ubt-protocol` + `ubt-client` only — Rust drivers can wait
until after the Go side ships CoreBluetooth and WinRT.
