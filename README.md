# Universal Bluetooth SDK

A cross-language, cross-platform Bluetooth toolkit organised around **one
long-lived daemon** that owns the radio and a **versioned wire protocol**
that anything else can speak to it. Typed CLI, AI planner, MCP server,
microservices, and language SDKs are all just adapters on the same surface.

```
                  Inbound adapters
   ┌─────────┬─────────┬─────────┬───────────────┬─────────────┐
   │ ubtctl  │ ubtctl  │ ubtctl  │  ubtctl plan  │ MCP clients │
   │ (typed) │  ask    │  mcp    │  show / run   │ Editors/IDE │
   └────┬────┴────┬────┴────┬────┴───────┬───────┴──────┬──────┘
        │         │         │            │              │
        └─────────┴────┬────┴────────────┴──────────────┘
                       │   v1 wire protocol  (length-prefixed JSON / UDS)
                       ▼
              ┌──────────────────┐
              │      ubtd        │   Go daemon: dispatcher, session manager,
              │   control plane  │   structured logs, audit, signal shutdown
              └────────┬─────────┘
                       │   transport.Driver  +  transport.Listener
                       ▼
   ┌──────────┬──────────────┬────────────────┬──────────────────┐
   │   stub   │ linuxrfcomm  │ corebluetooth  │      winrt       │
   │ (any OS) │  (Linux)     │  (macOS, TODO) │  (Windows, TODO) │
   └──────────┴──────────────┴────────────────┴──────────────────┘
                  Outbound adapters → OS Bluetooth stack
```

## Why this shape

| Decision | Pros | Cons | Status |
|---|---|---|---|
| Long-lived daemon owns the radio | One process holds RFCOMM sockets, BLE GATT sessions, and AI context across CLI invocations. Auditing and policy live in one place. | One more process to deploy and supervise. | **Shipped** as `ubtd`. |
| Hexagonal / ports-and-adapters | Adding an OS backend is implementing one Go interface; adding a new way to drive the daemon (web UI, editor, microservice) is another consumer of the wire protocol. | Slight indirection cost vs. monolithic CLIs. | **Shipped** — see `transport.Driver`. |
| JSON-over-UDS for the v1 wire | Debuggable with `nc`, zero codegen step, dumpable as plain text. | No streaming framing for binary; no native cross-language stubs. | **Shipped**. gRPC v2 is on the roadmap. |
| Go for daemon + CLI | Single static binary; cross-compiles to every host that runs Go; great `slog` / signal story; no runtime. | Bluetooth bindings are thinner than Python's. | **Shipped**. |
| Python kept as reference SDK | Production-tested, working RFCOMM client/server; useful as a fallback transport via shell-out for hosts where a native driver isn't ready. | Adds a runtime dependency if you opt into the bridge. | **Shipped** at `sdk/python`. |
| AI planner uses Claude Opus 4.7 with adaptive thinking | Highest-quality plans; one tool surface drives both AI runs and replayable scripts. | Requires `ANTHROPIC_API_KEY`; spend per call. | **Shipped** as `ubtctl ask`. |
| Single tool registry, multiple presentations | One Go file declares each tool; the typed CLI, the AI planner, the MCP server, and the plan replayer all see the same surface. Adding a daemon RPC mechanically extends them all. | Tools are bound to Go reflection for now; non-Go SDKs would re-derive their own. | **Shipped** at `cli/ubtctl/tools`. |
| MCP server (stdio JSON-RPC 2.0) | Any MCP-aware editor/agent (Claude Desktop, Cursor, Zed) can drive ubtd with no glue. | Stdio only; requires the client to exec the binary. | **Shipped** as `ubtctl mcp`. |
| Plan record / replay with `--save` + `ubtctl plan run` | AI runs become version-controllable, reviewable, replayable scripts; mutating steps gated behind explicit confirmation. | One bool flag per tool isn't enough for fine-grained policy at scale. | **Shipped**. |

## Repository layout

Every directory maps to exactly one layer of the hexagon:

```
.
├── common/
│   ├── protocol/              # v1.proto IDL + framing.md (the hex's "ports")
│   └── message-schema/        # reserved for future cross-language schemas
├── sdk/
│   ├── go/pkg/
│   │   ├── protocol/          # wire envelope + length-prefixed JSON codec
│   │   ├── sockaddr/          # default UDS path (single source of truth)
│   │   └── transport/         # outbound adapter contracts
│   │       ├── stub/             • in-memory reference driver (any OS)
│   │       └── linuxrfcomm/      • BlueZ-backed RFCOMM (Linux only)
│   ├── python/                # production-ready PyBluez client/server (reference impl)
│   └── rust/                  # planned native Rust SDK (same wire protocol)
├── cli/
│   ├── ubtd/                  # daemon binary (control plane)
│   └── ubtctl/                # CLI binary — five inbound adapters in one binary:
│       ├── commands/          •   typed verbs:  ping/status/.../send
│       ├── ai/                •   AI planner:   ubtctl ask (Claude Opus 4.7)
│       ├── mcp/               •   MCP server:   ubtctl mcp (stdio JSON-RPC)
│       ├── tools/             •   neutral tool registry shared by ai/ + mcp/
│       └── client/            •   shared daemon client (length-prefixed JSON)
├── microservices/
│   ├── grpc-server/           # planned: remote gRPC facade for ubtd
│   └── rest-server/           # planned: remote REST facade for ubtd
├── examples/
│   ├── chat/                  # planned: offline Bluetooth chat with local AI
│   ├── file-transfer/         # planned: chunked binary transfer over RFCOMM
│   └── sensor-stream/         # planned: streamed telemetry → time-series sink
└── docs/                      # architecture notes (this README is canonical)
```

## Quick start

### Path A — pure Go, no hardware required

Ships a working end-to-end loop in under a minute on any Linux/macOS/Windows
host. Uses the in-memory **stub** driver so the radio isn't touched.

```bash
git clone https://github.com/sraodev/bluetooth-service-rfcomm-python
cd bluetooth-service-rfcomm-python

go build -o bin/ubtd  ./cli/ubtd
go build -o bin/ubtctl ./cli/ubtctl

./bin/ubtd --socket /tmp/ubtd.sock --driver stub &

# Typed CLI
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl status
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl discover --scan-timeout 3
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl capabilities

# AI planner (requires ANTHROPIC_API_KEY)
ANTHROPIC_API_KEY=... ./bin/ubtctl ask \
  --save /tmp/last.plan.json \
  "show me the daemon status and list any nearby devices"

# Replay the saved plan against the same daemon — no LLM, no spend
./bin/ubtctl plan show /tmp/last.plan.json
./bin/ubtctl plan run  /tmp/last.plan.json

# Expose the same tool registry over MCP for editors / external agents
./bin/ubtctl mcp --socket /tmp/ubtd.sock
```

### Path B — Linux with real hardware

Same daemon, swap the driver:

```bash
sudo apt install bluez bluez-tools libbluetooth-dev
sudo ./bin/ubtd --socket /tmp/ubtd.sock --driver linuxrfcomm &
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl discover
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl send \
  --address AA:BB:CC:DD:EE:FF --port 22 --data 'hello'
```

Discovery uses `bluetoothctl devices` (BlueZ); send opens an
`AF_BLUETOOTH / SOCK_STREAM / BTPROTO_RFCOMM` socket directly.

### Path C — Python SDK (the reference implementation)

The Python SDK in `sdk/python` is production-ready PyBluez code; useful as the
reference for what RFCOMM should do, and as an immediate fallback on hosts
where a native Go driver isn't ready. See [`sdk/python/README.md`](sdk/python/README.md)
for setup; the `install_dependencies.sh` script handles BlueZ, PyBluez, and
their well-known PyPI breakage.

## Using each entry point

### 1. Typed CLI

| Command | Purpose |
|---|---|
| `ubtctl ping` | Liveness + clock-skew check |
| `ubtctl version` | CLI + daemon + protocol versions |
| `ubtctl status` | Daemon health, sessions, registered drivers |
| `ubtctl capabilities` | Per-transport feature matrix |
| `ubtctl discover` | Stream device scan results until timeout |
| `ubtctl send` | One-shot payload to a peer (`--data`, `--file`, or stdin) |
| `ubtctl ask` | Natural-language goal → AI planner → daemon RPCs |
| `ubtctl mcp` | Serve the tool registry over MCP on stdio |
| `ubtctl plan show` | Pretty-print a saved AI plan |
| `ubtctl plan run` | Replay a saved AI plan against ubtd (no LLM) |

`ubtctl <command> -h` prints per-command flags.

### 2. AI planner

```bash
export ANTHROPIC_API_KEY=sk-ant-...

ubtctl ask "is the daemon healthy and what drivers are loaded?"
ubtctl ask --dry-run "send /tmp/payload.bin to the first device you find"
ubtctl ask --yes     "send 'hello' to AA:BB:CC:DD:EE:01"
ubtctl ask --save runbooks/morning-check.plan.json \
  "ping the daemon, list capabilities, and discover devices"
```

Defaults: `claude-opus-4-7`, adaptive thinking, prompt-caching on the system
prompt + tool list. Five tools are auto-registered from the daemon's RPC
surface; the only mutator (`send_payload`) honours `--dry-run` and `--yes`.

### 3. MCP server

Exposes the same tool registry to any MCP-aware client.

```jsonc
{
  "mcpServers": {
    "ubtctl": {
      "command": "/usr/local/bin/ubtctl",
      "args": ["mcp", "--socket", "/tmp/ubtd.sock"]
    }
  }
}
```

The server speaks JSON-RPC 2.0 with newline-delimited frames over stdio,
implements `initialize`, `ping`, `tools/list`, `tools/call`, and
auto-derives JSON Schemas from Go struct tags. Logs go to stderr only —
stdout is reserved for the protocol stream.

### 4. Plan record / replay

Every `ubtctl ask` run can capture its tool-call trace into JSON. The
saved file is human-readable, version-controllable, and replayable
**without going back to the LLM** — useful for auditing, runbooks, CI
smoke-tests, and "I want exactly that again."

```bash
ubtctl ask --save /tmp/p.json "discover devices then ping the daemon"
ubtctl plan show /tmp/p.json                # pretty-print
ubtctl plan run  /tmp/p.json                # read-only steps run unattended
ubtctl plan run --dry-run /tmp/p.json       # never contacts ubtd
ubtctl plan run --yes     /tmp/p.json       # required for steps tagged mutating: true
```

Plans carry a `format_version`; replay rejects unknown versions cleanly.

### 5. Daemon

```
ubtd [flags]
  --socket <path>        Unix domain socket path
                         (default: $XDG_RUNTIME_DIR/ubtd.sock or /tmp/ubtd.sock)
  --driver {stub|linuxrfcomm}
                         which transport to register at startup
  --bluetoothctl <path>  override bluetoothctl path (linuxrfcomm only)
  --log-json             emit JSON-structured logs
```

`UBTD_LOG_LEVEL` (`debug`/`info`/`warn`/`error`) controls verbosity.

## Wire protocol

| | |
|---|---|
| Source of truth | [`common/protocol/v1.proto`](common/protocol/v1.proto) |
| Wire format | [`common/protocol/framing.md`](common/protocol/framing.md) |
| Go bindings | [`sdk/go/pkg/protocol`](sdk/go/pkg/protocol) |
| Wire today | length-prefixed JSON envelope over Unix domain socket |
| Wire tomorrow | gRPC v2 generated from the same `.proto` |

Methods today: `Ping`, `Version`, `Capabilities`, `Discover`, `Send`,
`Status`. Error codes are stable strings (`unknown_method`,
`not_implemented`, `invalid_params`, `transport_error`, `not_found`,
`frame_too_large`, `internal`).

## Roadmap

| Phase | Item | Why it matters |
|-------|------|---|
| 1 | **CoreBluetooth (macOS) and WinRT (Windows) drivers** | Same `Driver` interface as `linuxrfcomm`. Daemon already advertises capabilities at runtime, so the rest of the stack picks them up automatically. |
| 2 | **`Listen` / `Reply` / `CloseSession` RPCs + `Listener` interface** | Bidirectional sessions for chat, telemetry, file transfer. Foundation for the offline chat-app example with a local-AI assist (Ollama). |
| 3 | **`ubtctl chat` + `examples/chat`** | Two devices over RFCOMM, optional local-LLM piping for translation / smart reply / summarisation. No internet required. |
| 4 | **gRPC v2 wire** | Generated stubs for every supported language, alongside the JSON wire during migration. |
| 5 | **Native Go and Rust SDKs** | Same `protocol` package + `transport.Driver` shape; ships when an embedded use case demands it. |
| 6 | **`microservices/{grpc,rest}-server`** | Remote control planes that proxy the same RPC surface to clients that can't (or shouldn't) hold a UDS into `ubtd`. |
| 7 | **Stronger policy** | Per-tool RBAC and rate limiting, replacing today's single `mutating` boolean. |

## Project history

The repo was originally a PyBluez-only RFCOMM client/server (still
preserved in `sdk/python`). Everything else — Go daemon, multi-driver
architecture, AI planner, MCP server, plan replay — was added as a
ground-up second life of the project; the existing Python SDK is kept
both as the reference implementation and as a working escape hatch on
hosts where the native Go drivers aren't ready.

## Contributing

Issues and PRs welcome. Read the
[contribution guidelines](https://github.com/sraodev/super-opensource-cheat-sheets/blob/master/contributing.md)
first.

## References

- [Bluetooth Programming with Python 3](http://blog.kevindoran.co/bluetooth-programming-with-python-3)
- [Bluetooth Programming with Python — PyBluez](https://people.csail.mit.edu/albert/bluez-intro/x232.html)
- [Bluetooth for Programmers](http://people.csail.mit.edu/rudolph/Teaching/Articles/PartOfBTBook.pdf)
- [PyBluez](https://github.com/pybluez/pybluez)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [BlueZ](http://www.bluez.org/)
