# ubtctl and ubtd

Build from the repository root using the [quick start](../../README.md#quick-start-without-bluetooth-hardware).
`ubtd` loads one driver; `ubtctl` calls it through a local Unix socket.

| Command | Purpose |
|---|---|
| `ping`, `version`, `status`, `capabilities` | Inspect the daemon |
| `discover` | Stream driver results; Linux currently lists BlueZ-known peers |
| `send --address ADDRESS --data TEXT` | One-shot send; also accepts `--file` or `--file -` for stdin |
| `ask [flags] "goal"` | Cloud Claude planner; optional, billable |
| `mcp --socket PATH` | MCP tools over stdio |
| `plan show FILE` | Inspect a recorded tool trace |
| `plan run [flags] FILE` | Replay without an LLM |

Flags must come **before positional arguments**. Use `ubtctl COMMAND -h`.

## Planner

```bash
# Assumes ANTHROPIC_API_KEY was set securely outside this command.
./bin/ubtctl ask --dry-run --save /tmp/check.plan.json \
  "check daemon status and list known devices"
./bin/ubtctl plan show /tmp/check.plan.json
./bin/ubtctl plan run --dry-run /tmp/check.plan.json
```

The implementation defaults to `claude-opus-4-7`; `--model` accepts an override.
Model availability and live API behavior are not covered by the hardware-free
tests. Goals, addresses, and tool results can leave the machine. `ask --dry-run`
still calls the model and reads daemon state, but substitutes a synthetic send
result. **`ask` has no `--yes` flag and otherwise permits sends.**

Plans record actual tool calls, not a guaranteed successful workflow. They are
currently unversioned JSON. Replay checks the live registry's mutation policy,
preflights unknown tools, and stops on a tool failure. `plan run --yes FILE`
allows mutating tools; review addresses and payloads first. `plan run --dry-run`
needs neither a running daemon nor credentials. Keep saved plans private.

## MCP

Launch `./bin/ubtctl mcp --socket "$UBTD_SOCKET"` from a compatible client.
The server implements revision `2025-03-26` with `initialize`, `ping`,
`tools/list`, and `tools/call`. This is a limited implementation, not an MCP
conformance certification. Tools are `ping_daemon`, `get_status`,
`get_capabilities`, `discover_devices`, and `send_payload`.

Use an absolute executable and socket path in your client's configuration:

```json
{
  "mcpServers": {
    "ubtctl": {
      "command": "/absolute/path/to/bin/ubtctl",
      "args": ["mcp", "--socket", "/absolute/private/path/daemon.sock"]
    }
  }
}
```

The send tool is enabled; the server does not enforce a per-call approval gate.
Use trusted clients and the stub driver for experiments. Logs use stderr;
stdout is reserved for JSON-RPC.

## Configuration

| Variable / flag | Meaning |
|---|---|
| `UBTD_SOCKET` | Client socket; overridable by `--socket` |
| `UBTD_LOG_LEVEL` | Daemon log level: debug, info, warn, error |
| `ANTHROPIC_API_KEY` | Cloud planner credential only |
| `ubtd --driver stub\|linuxrfcomm` | Transport selection |
| `ubtd --bluetoothctl PATH` | Override Linux discovery executable |
| `ubtd --log-json` | Structured logs |

Default socket lookup is `$XDG_RUNTIME_DIR/ubtd.sock`, otherwise `/tmp/ubtd.sock`.
Prefer an explicit path inside a directory with mode 0700. Do not launch two
daemons with the same path; startup currently removes the old path.
