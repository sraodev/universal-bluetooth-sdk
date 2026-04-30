# Universal Bluetooth SDK

Cross-language, cross-platform toolkit for building Bluetooth solutions. The
control plane is a single long-lived daemon (`ubtd`) that owns every radio
session; everything else — the typed CLI, the AI planner, the MCP server, the
language SDKs — talks to it through one versioned wire format.

```
ubtctl  ──┐                                           ┌── BlueZ (Linux)
ubtctl ask│  ──UDS── ubtd ─── TransportDriver ────────┤
ubtctl mcp│           │                               ├── CoreBluetooth (TODO)
MCP client┘           └── audit / policy / sessions   └── WinRT (TODO)
                                  │
                                  └── Python / Go / Rust SDKs (same protocol)
```

## What's working today

- **`ubtd`** — Go daemon, Unix-socket control plane, pluggable transport
  drivers, structured slog, signal-driven shutdown.
  - `--driver stub` — in-memory reference driver (any host).
  - `--driver linuxrfcomm` — BlueZ-backed RFCOMM via the kernel's
    `AF_BLUETOOTH` socket; `Discover` enumerates known peers via
    `bluetoothctl devices`.
- **`ubtctl`** — Go CLI. One binary, several front-ends:
  - Typed verbs: `ping`, `version`, `status`, `capabilities`,
    `discover`, `send`.
  - **AI planner**: `ubtctl ask "<goal>"` runs Claude Opus 4.7 with
    adaptive thinking against a tool registry that is 1:1 with the
    daemon's RPC surface.
  - **MCP server**: `ubtctl mcp` exposes the same tool registry over
    JSON-RPC 2.0 on stdio, so any MCP-aware editor or agent (Claude
    Desktop, Cursor, Zed, …) can drive ubtd directly.
  - **Plan record/replay**: `ubtctl ask --save plan.json …` captures
    every tool call; `ubtctl plan show / run` replay it later without
    going back to the LLM. Mutating steps are gated behind `--yes`.
- **Python SDK (`sdk/python`)** — production-ready PyBluez SDK kept around
  as the reference implementation and as a path the daemon can shell out to
  on hosts where a native driver isn't ready yet.
- **Common contract (`common/protocol/`)** — `v1.proto` IDL (future
  gRPC) plus `framing.md` describing the v1 wire (length-prefixed JSON
  over UDS).

## Repository layout

```
.
├── cli/
│   ├── ubtd/              # Go daemon (UDS server, dispatcher)
│   └── ubtctl/            # Go CLI (typed verbs, ai planner, mcp server)
│       ├── ai/            # Claude tool runner + plan record/replay
│       ├── client/        # daemon client (length-prefixed JSON codec)
│       ├── commands/      # subcommand registry (ping/status/.../ask/mcp/plan)
│       ├── mcp/           # JSON-RPC 2.0 MCP server (stdio)
│       └── tools/         # neutral Spec/Registry shared by ai + mcp
├── sdk/
│   ├── go/pkg/
│   │   ├── protocol/      # wire envelope + codec
│   │   ├── sockaddr/      # default socket location
│   │   └── transport/     # Driver port + Registry
│   │       ├── stub/         # in-memory reference driver
│   │       └── linuxrfcomm/  # BlueZ-backed RFCOMM (Linux only)
│   ├── python/            # production-ready PyBluez SDK
│   └── rust/              # planned
├── microservices/
│   ├── grpc-server/       # planned (REST/gRPC façade for remote callers)
│   └── rest-server/
├── common/
│   ├── protocol/          # v1.proto + framing.md
│   └── message-schema/
├── examples/              # scenario samples (chat, sensor stream, file xfer)
└── docs/
```

## Quick start (Go)

```bash
go build -o bin/ubtd  ./cli/ubtd
go build -o bin/ubtctl ./cli/ubtctl

# 1. Start the daemon. Use `stub` for a hardware-free dev loop;
#    on Linux, switch to linuxrfcomm to talk to real radios.
./bin/ubtd --socket /tmp/ubtd.sock --driver stub &

# 2. Drive it from the typed CLI.
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl status
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl discover --scan-timeout 3

# 3. Or drive it from natural language (requires ANTHROPIC_API_KEY).
ANTHROPIC_API_KEY=... ./bin/ubtctl ask \
  --save /tmp/last.plan.json \
  "show me the daemon status and list any nearby devices"

# 4. Replay the captured plan against the same daemon — no LLM, no spend.
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl plan show /tmp/last.plan.json
UBTD_SOCKET=/tmp/ubtd.sock ./bin/ubtctl plan run  /tmp/last.plan.json

# 5. Or expose the same tool registry over MCP for editors / external agents.
./bin/ubtctl mcp --socket /tmp/ubtd.sock      # speaks JSON-RPC on stdio
```

For per-command flags and MCP client config, see
[`cli/ubtctl/README.md`](cli/ubtctl/README.md).

## Python SDK (sdk/python)

Production-ready PyBluez RFCOMM client/server. Tested on Raspberry Pi OS and
Ubuntu. Detailed setup, troubleshooting, and the full PyBluez fix-up notes
live in [`sdk/python/README.md`](sdk/python/README.md).

```bash
cd sdk/python
sudo ./scripts/install_dependencies.sh
sudo python3 run_server.py
sudo python3 run_client.py
```

## Roadmap

- **CoreBluetooth (macOS) and WinRT (Windows) drivers** — same `Driver`
  interface as `linuxrfcomm`; the daemon already advertises the capability
  matrix at runtime.
- **`Listen` / `Reply` RPCs** — bidirectional RFCOMM sessions for chat /
  long-lived data streams (foundation for an offline Bluetooth chat app
  with a local-AI assist).
- **gRPC v2 wire** — generated from `common/protocol/v1.proto`,
  alongside the JSON-over-UDS v1 wire during migration.
- **Native Go and Rust SDKs** — same `protocol` package the daemon and CLI
  already use.
- **`microservices/{grpc,rest}-server`** — remote control planes that
  re-export the daemon surface.

## Contributing

Issues and PRs welcome. Read the
[contribution guidelines](https://github.com/sraodev/super-opensource-cheat-sheets/blob/master/contributing.md)
first.

## References

- [Bluetooth Programming with Python 3](http://blog.kevindoran.co/bluetooth-programming-with-python-3)
- [Bluetooth Programming with Python — PyBluez](https://people.csail.mit.edu/albert/bluez-intro/x232.html)
- [Bluetooth for Programmers](http://people.csail.mit.edu/rudolph/Teaching/Articles/PartOfBTBook.pdf)
- [PyBluez](https://github.com/karulis/pybluez)
- [Model Context Protocol](https://modelcontextprotocol.io/)
