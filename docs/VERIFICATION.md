# Verification — 2026-08-30

Checkout: `sraodev/universal-bluetooth-sdk`, branch `codex/repository-maturity`.
Base: `faa8228f167413e95f3167322f9c2108d2e4a54c` (fresh clone; initially clean).
Host: macOS arm64, Go 1.25.5, Python 3.14.3. No other repository was edited.

## Changes verified

Replay checks live tool mutation policy, validates tool existence before running,
stops on tool-reported failure, and previews without connecting to the daemon.
Daemon shutdown closes idle/partial-frame client connections. Tests cover these
regressions. Local AI helper tests use a fake HTTP server; no live LLM request.

## Exact commands and output

```text
$ go test -race ./...
?   	github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl	[no test files]
ok  	github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/ai	(cached)
?   	github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/client	[no test files]
ok  	github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/commands	1.570s
?   	github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/mcp	[no test files]
?   	github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtctl/tools	[no test files]
?   	github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtd	[no test files]
ok  	github.com/sraodev/bluetooth-service-rfcomm-python/cli/ubtd/server	(cached)
ok  	github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/protocol	(cached)
?   	github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/sockaddr	[no test files]
?   	github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/transport	[no test files]
?   	github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/transport/linuxrfcomm	[no test files]
?   	github.com/sraodev/bluetooth-service-rfcomm-python/sdk/go/pkg/transport/stub	[no test files]
exit 0
```

```text
$ go vet ./...
exit 0
```

```text
$ sh -c 'test -z "$(gofmt -l cli sdk/go)"'
exit 0
```

```text
$ go build -o bin/ubtd ./cli/ubtd
exit 0
```

```text
$ go build -o bin/ubtctl ./cli/ubtctl
exit 0
```

```text
$ python3 scripts/smoke.py
PASS: stub ping/status/discover/send, replay consent/offline preview, MCP, shutdown
exit 0
```

```text
$ python3 -m unittest discover -s examples/chat -p 'test_*.py' -v
test_cloud_and_oversized_inputs_rejected_before_connection (test_local_draft.DraftTests.test_cloud_and_oversized_inputs_rejected_before_connection) ... ok
test_completed_draft_uses_nonstreaming_api_without_tools (test_local_draft.DraftTests.test_completed_draft_uses_nonstreaming_api_without_tools) ... ok
test_http_error (test_local_draft.DraftTests.test_http_error) ... ok
test_invalid_responses_are_rejected (test_local_draft.DraftTests.test_invalid_responses_are_rejected) ... ok
test_private_file_and_no_overwrite (test_local_draft.DraftTests.test_private_file_and_no_overwrite) ... ok

----------------------------------------------------------------------
Ran 5 tests in 0.014s

OK
exit 0
```

```text
$ env GOOS=linux GOARCH=amd64 go build ./...
exit 0
```

```text
$ env GOOS=linux GOARCH=amd64 go test -c -o bin/linuxrfcomm.test ./sdk/go/pkg/transport/linuxrfcomm
exit 0
```

```text
$ git diff --check
exit 0
```

## Explicit gaps

- Linux builds and Linux driver tests were **cross-compiled, not executed**.
  Docker was unavailable; no Linux VM or physical adapter was used. The new
  Linux/macOS CI matrix was subsequently executed successfully on GitHub; see below.
- `PYTHONPATH=sdk/python .venv/bin/python -m pytest sdk/python/tests -q` cannot
  collect the legacy suite on this host: `ModuleNotFoundError: No module named
  'bluetooth'` (two collection errors). PyBluez is not installed. The legacy SDK
  is not represented as verified by the Go/demo checks.
- No actual Bluetooth delivery, range, BLE GATT, phone background operation,
  Bitchat compatibility, local-model inference, or cloud planner execution tested.
- Original social preview visually inspected; PNG MIME and 1280×640 dimensions
  checked. GitHub YAML parsed and local Markdown targets checked separately.
- Security limitations in `SECURITY.md` are unresolved readiness gates, especially
  legacy pickle defaults, daemon socket lifecycle, radio deadlines and agent send
  permissions. Do not call this production-ready or security-audited.

## Live GitHub changes

- Active ruleset [Protect master](https://github.com/sraodev/universal-bluetooth-sdk/rules/21847044)
  blocks deletion and non-fast-forward updates; no bypass actors. Read-back
  confirmed `protected: true`. After successful native GitHub runs, both
  `Go (ubuntu-latest)` and `Go (macos-latest)` were made required, tied to the
  GitHub Actions app, with strict up-to-date checking.
- Nine scoped contributor issues plus [tracking issue #14](https://github.com/sraodev/universal-bluetooth-sdk/issues/14)
  were published. Two are marked `good first issue`; nine have `help wanted`.

This records the initial local validation, before publication. The user subsequently
authorized publication; see the pull-request checks and repository rules for live
CI and protection status. About/topics and private vulnerability reporting were
configured as part of that follow-up. Hardware and legacy Python limitations above
remain in effect.

## Native CI follow-up

[GitHub run 33301222523](https://github.com/sraodev/universal-bluetooth-sdk/actions/runs/33301222523)
passed both Linux and macOS jobs for commit
`87088b706279ed1a41e8f68eb91af39d15f4db95`. Each ran Go vet, formatting,
race tests, binary builds, stub/MCP smoke, and local-draft fake-server tests.
This adds native Linux evidence but is still not physical Bluetooth or live model
validation. Subsequent PR checks remain authoritative for later commits.
