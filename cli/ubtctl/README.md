# ubtctl + ubtd

This directory ships **two binaries** that share one architecture:

- **`ubtd`** — the long-lived control-plane daemon. The only process that
  touches the radio. Loads exactly one transport driver at startup
  (`stub` for hardware-free dev, `linuxrfcomm` on Linux with BlueZ).
- **`ubtctl`** — the Universal Bluetooth CLI. Five inbound adapters in
  one binary: typed verbs, AI planner (`ask`), MCP server (`mcp`),
  plan record/replay (`plan`), and the daemon client they share.

Neither binary contains hardware-specific code; they speak the
[wire protocol](../../common/protocol/framing.md) to each other and let the
loaded driver do the radio work.

```
ubtctl  (typed CLI / ask / mcp / plan)
   │
   ▼     length-prefixed JSON over UDS
 ubtd
   │
   ▼     transport.Driver  (+ Listener, planned)
 stub │ linuxrfcomm │ corebluetooth (TODO) │ winrt (TODO)
```

The package layout under `cli/ubtctl/` mirrors this:

| Sub-package | Role |
|---|---|
| `client/` | Daemon client (length-prefixed JSON codec, request/streaming helpers). Used by every other sub-package. |
| `commands/` | Typed verb registry. Each command parses its flags then calls into `client/`. Adding a verb here registers it for the root command automatically. |
| `tools/` | **Neutral tool registry.** Specs declared once via reflection-derived JSON Schema, consumed by both `ai/` and `mcp/`. The `Mutating` flag gates write operations during plan replay. |
| `ai/` | Claude Opus 4.7 planner: builds Specs, adapts them to `anthropic.BetaTool`, runs the streaming agentic loop, captures the trace into a Plan. |
| `mcp/` | JSON-RPC 2.0 MCP server over stdio: same Specs from `tools/`, exposed as `tools/list` + `tools/call`. |

Adding a daemon RPC mechanically extends every front-end: declare the
spec in `tools/`, dispatch it in `cli/ubtd/server/`, and the typed
CLI, AI planner, MCP server, and plan replay all see it at the same
time. That is the architectural promise the package layout is designed
to keep.

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
| `mcp`           | Serve the tool registry over MCP on stdio            |
| `plan`          | Show / replay a saved AI plan (no LLM)               |

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

## Plan record / replay (`ubtctl ask --save` + `ubtctl plan`)

Every `ubtctl ask` run can capture its tool-call trace into a JSON file.
The saved plan is human-readable, version-controllable, and replayable
against the daemon **without going back to the LLM** — useful for
auditing, runbooks, CI smoke-tests, and "I want to do that exact thing
again."

```bash
# 1. Run the AI once and save the trace.
ubtctl ask --save /tmp/morning-check.plan.json \
  "ping the daemon, list capabilities, and discover devices"

# 2. Pretty-print the captured plan.
ubtctl plan show /tmp/morning-check.plan.json

# 3. Replay it. Read-only steps run by default; mutating steps require --yes.
ubtctl plan run /tmp/morning-check.plan.json           # blocked if the plan
                                                       # contains send_payload
ubtctl plan run --yes /tmp/morning-check.plan.json     # allowed
ubtctl plan run --dry-run /tmp/morning-check.plan.json # never contacts ubtd
```

Plans are versioned (`format_version: 1`); replay refuses unknown versions
rather than mis-execute. Mutating steps carry a `mutating: true` flag in
the JSON so reviewers can diff for side-effects before approving a script.

## MCP server (`ubtctl mcp`)

Exposes the same tool registry the in-process AI planner uses, but over the
[Model Context Protocol](https://modelcontextprotocol.io/) on stdio. Any
MCP-aware client (Claude Desktop, Cursor, Zed, custom agents) can drive
ubtd by launching the binary directly.

Example client config:

```json
{
  "mcpServers": {
    "ubtctl": {
      "command": "/usr/local/bin/ubtctl",
      "args": ["mcp", "--socket", "/tmp/ubtd.sock"],
      "env": {}
    }
  }
}
```

The server speaks JSON-RPC 2.0 with newline-delimited frames, supports
`initialize`, `ping`, `tools/list`, `tools/call`, and exposes the same five
tools (`ping_daemon`, `get_status`, `get_capabilities`, `discover_devices`,
`send_payload`) with auto-derived JSON Schemas. Logs go to stderr only —
stdout is reserved for the MCP stream.

Quick smoke test from the shell:

```bash
{
  echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}'
  echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
  echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_status","arguments":{}}}'
} | ubtctl mcp --socket /tmp/ubtd.sock
```

## Configuration

| Variable             | Default                                    | Notes                                      |
|----------------------|--------------------------------------------|--------------------------------------------|
| `UBTD_SOCKET`        | `$XDG_RUNTIME_DIR/ubtd.sock` or `/tmp/ubtd.sock` | Override with `--socket`              |
| `UBTD_LOG_LEVEL`     | `info`                                     | `debug`/`info`/`warn`/`error`              |
| `ANTHROPIC_API_KEY`  | —                                          | Required for `ubtctl ask`                  |

## Status

Phase 5: typed CLI + AI planner + Linux RFCOMM driver + MCP server, all
sharing the same tool registry. Adding a daemon RPC mechanically extends
the typed CLI, the AI planner's tool set, and the MCP surface in lockstep.
