# Go SDK

Shared Go packages consumed by `ubtd` (the daemon binary) and `ubtctl`
(the CLI binary). This is the **outbound side of the hexagon**: the
`Driver` interface defined here is the port through which the daemon
talks to whatever Bluetooth stack the host provides.

```
sdk/go/pkg/
├── protocol/      • wire envelope + length-prefixed JSON codec
│                    (twin of common/protocol/framing.md)
├── sockaddr/      • default UDS path; single source of truth
└── transport/
    ├── driver.go        • the Driver interface (the hexagon's port)
    ├── stub/            • in-memory reference driver (any OS, no hardware)
    └── linuxrfcomm/     • BlueZ-backed RFCOMM (Linux only)
        ├── driver_linux.go         AF_BLUETOOTH socket, bluetoothctl devices
        └── driver_other.go         build-tag stub for darwin/windows
```

## Package responsibilities

| Package | Role |
|---|---|
| `protocol` | Wire envelope (`Envelope`, `Error`, method constants), `WriteFrame` / `ReadFrame` length-prefixed codec, request/response/event payload types. Imported by both daemon and CLI. |
| `sockaddr` | One function: `Default()` returns the daemon socket path, honouring `UBTD_SOCKET` → `$XDG_RUNTIME_DIR/ubtd.sock` → `/tmp/ubtd.sock`. Imported anywhere a process talks to or hosts the daemon. |
| `transport` | The `Driver` interface + a `Registry` that maps names ↔ transports ↔ drivers. The daemon creates a registry at startup and registers exactly one driver into it (controlled by `--driver`). |
| `transport/stub` | Reference driver. `Send` succeeds for any well-formed address; `Discover` emits two synthetic devices. Used by every test in this repo and by anyone doing local dev without hardware. |
| `transport/linuxrfcomm` | Real RFCOMM. `Send` opens an `AF_BLUETOOTH / SOCK_STREAM / BTPROTO_RFCOMM` socket via `golang.org/x/sys/unix` and writes directly. `Discover` shells out to `bluetoothctl devices` and parses the output. Build-tagged: non-Linux builds compile a stub that returns `not_implemented`. |

## Driver contract

Every transport adapter implements [`transport.Driver`](pkg/transport/driver.go):

```go
type Driver interface {
    Name() string                                              // short ID, e.g. "linuxrfcomm"
    Capability() protocol.Capability                           // what this driver advertises
    Discover(ctx, params, out chan<- protocol.Device) error    // streaming
    Send(ctx, params) (protocol.SendResult, error)             // one-shot
    Close() error
}
```

Errors returned as `*protocol.Error{Code, Message}` survive the wire
intact — the daemon's dispatcher uses `errors.As` to preserve the
typed error code (`invalid_params`, `transport_error`,
`not_implemented`, etc.) instead of flattening every failure to a
single code.

## Status

| Component | Status |
|---|---|
| `protocol` (codec, types, error codes) | **Implemented** (wire v1). Round-trip tests in `pkg/protocol/codec_test.go`. |
| `transport.Driver` interface + `Registry` | **Implemented** |
| `transport/stub` | **Implemented**. Reference driver used by every test and the default `--driver stub` flag. |
| `transport/linuxrfcomm` | **Implemented** (Send + Discover via bluetoothctl). Tests cover BD address parsing and the `bluetoothctl devices` line parser. |
| CoreBluetooth driver (macOS) | Planned. Same `Driver` contract, `framework CoreBluetooth` via cgo. |
| WinRT driver (Windows) | Planned. Same `Driver` contract, WinRT bindings. |
| `transport.Listener` (bidirectional sessions) | Planned. Phase 2 of the [roadmap](../../README.md#roadmap). |
| gRPC v2 wire | Planned. Will be generated from [`common/protocol/v1.proto`](../../common/protocol/v1.proto) and run alongside the v1 JSON wire during migration. |
