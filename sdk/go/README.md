# Go SDK

Shared Go packages used by `ubtd` (the daemon) and `ubtctl` (the CLI).

```
sdk/go/pkg/
├── protocol/    # wire envelope + length-prefixed JSON codec (see common/protocol/framing.md)
├── sockaddr/    # default Unix socket location, single source of truth
└── transport/   # Driver interface + Registry + reference stub driver
```

## Driver contract

Every transport adapter (RFCOMM-BlueZ, BLE-CoreBluetooth, WinRT, …) implements
`transport.Driver`:

```go
type Driver interface {
    Name() string
    Capability() protocol.Capability
    Discover(ctx, params, out chan<- Device) error
    Send(ctx, params) (SendResult, error)
    Close() error
}
```

`stub.New()` is the in-memory reference driver — used for tests, CI, and any
host without Bluetooth hardware. Real drivers land alongside it in
`pkg/transport/<name>/`.

## Status

- Protocol + codec: implemented, wire v1.
- Stub driver: implemented.
- BlueZ (Linux) / CoreBluetooth (macOS) / WinRT (Windows) drivers: TODO.
- gRPC transport: TODO (the JSON-over-UDS framing is the v1 wire; gRPC will be
  v2, generated from `common/protocol/v1.proto`).
