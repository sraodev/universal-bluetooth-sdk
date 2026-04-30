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
# 1a. Start the daemon with the in-memory stub (works on any host).
./bin/ubtd --socket /tmp/ubtd.sock &

# 1b. Or, on Linux, register the BlueZ-backed RFCOMM driver:
sudo ./bin/ubtd --socket /tmp/ubtd.sock --driver linuxrfcomm &

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
| `ask`           | Natural-language goal → AI planner → daemon RPCs     |

Run `ubtctl <command> -h` for per-command flags.

## AI planner (`ubtctl ask`)

```bash
export ANTHROPIC_API_KEY=sk-ant-...

ubtctl ask "is the daemon healthy and what drivers are loaded?"
ubtctl ask --dry-run "send the contents of /tmp/payload.bin to the first device you find"
ubtctl ask --yes     "send 'hello' to AA:BB:CC:DD:EE:01"
```

The planner uses Claude Opus 4.7 with adaptive thinking. Every tool the model
can call is a 1:1 wrapper around an existing daemon RPC — there is no second
execution path, so AI runs are auditable as plain CLI calls. Read tools
(`get_status`, `get_capabilities`, `discover_devices`, `ping_daemon`) always
run; the only mutator (`send_payload`) honours `--dry-run` and `--yes`.

The system prompt + tool list are kept stable across runs and marked with
`cache_control: ephemeral`, so subsequent invocations read the cached prefix
instead of paying for it.

## Configuration

| Variable             | Default                                    | Notes                                      |
|----------------------|--------------------------------------------|--------------------------------------------|
| `UBTD_SOCKET`        | `$XDG_RUNTIME_DIR/ubtd.sock` or `/tmp/ubtd.sock` | Override with `--socket`              |
| `UBTD_LOG_LEVEL`     | `info`                                     | `debug`/`info`/`warn`/`error`              |
| `ANTHROPIC_API_KEY`  | —                                          | Required for `ubtctl ask`                  |

## Status

Phase 4 of the plan in the repo root README: typed CLI plus AI planner
working end-to-end against the stub driver. Native Bluetooth drivers ride on
the same wire format and ship next.
