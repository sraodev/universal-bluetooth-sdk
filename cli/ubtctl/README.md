# ubtctl + ubtd

`ubtctl` is the Universal Bluetooth CLI; `ubtd` is the long-lived control-plane
daemon that owns every Bluetooth session. The CLI never touches the radio
directly — it speaks the wire protocol in
[`common/protocol/framing.md`](../../common/protocol/framing.md) to the daemon.

```
ubtctl ──UDS──> ubtd ──port──> TransportDriver ──> radio
```

The same wire surface is what the AI planner targets, so adding a verb here
extends the AI tool registry automatically.

## Build

```bash
go build -o bin/ubtd  ./cli/ubtd
go build -o bin/ubtctl ./cli/ubtctl
```

## Quick start

```bash
# 1. Start the daemon (registers the in-memory stub driver by default).
./bin/ubtd --socket /tmp/ubtd.sock &

# 2. Talk to it.
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl version
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl status
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl capabilities
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl discover --scan-timeout 3
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl send --address AA:BB:CC:DD:EE:01 --data 'hi'
```

## Commands

| Command         | Purpose                                              |
|-----------------|------------------------------------------------------|
| `ping`          | Liveness + clock-skew check                          |
| `version`       | CLI + daemon + protocol versions                     |
| `status`        | Daemon health and active sessions                    |
| `capabilities`  | Per-transport feature matrix                         |
| `discover`      | Stream device scan results until timeout             |
| `send`          | Push a payload (`--data`, `--file`, or stdin)        |

Run `ubtctl <command> -h` for per-command flags.

## Configuration

| Variable        | Default                                    | Notes                       |
|-----------------|--------------------------------------------|-----------------------------|
| `UBTD_SOCKET`   | `$XDG_RUNTIME_DIR/ubtd.sock` or `/tmp/ubtd.sock` | Override with `--socket` |
| `UBTD_LOG_LEVEL`| `info`                                     | `debug`/`info`/`warn`/`error` |

## Status

Phase 2 of the plan in the repo root README: typed CLI surface working
end-to-end against the stub driver. Native Bluetooth drivers and the AI
planner ride on the same wire format and ship next.
