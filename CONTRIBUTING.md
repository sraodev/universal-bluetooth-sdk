# Contributing

Welcome. Small, tested changes are useful even without Bluetooth hardware.
The project is experimental; keep claims tied to code and test evidence.

## First contribution

1. Pick an unassigned [good first issue](https://github.com/sraodev/universal-bluetooth-sdk/issues?q=is%3Aissue%20is%3Aopen%20label%3A%22good%20first%20issue%22).
2. Comment with your approach so contributors do not duplicate work. No assignment
   is needed for a small typo fix; discuss protocol/API changes before coding.
3. Fork, create a branch, and keep the diff focused on the issue's acceptance criteria.
4. Run verification and open a PR explaining behavior, tests, and limitations.

## Local development

Go 1.24+ and Python 3.10+ are enough for the Go core and demo checks. No radio,
cloud key, or Python dependency is needed for these commands, from the repo root:

```bash
go test -race ./...
go vet ./...
test -z "$(gofmt -l cli sdk/go)"
go build -o bin/ubtd ./cli/ubtd
go build -o bin/ubt ./cli/ubtctl
go build -o bin/ubtctl ./cli/ubtctl # 0.x compatibility alias
python3 scripts/smoke.py
python3 -m unittest discover -s examples/chat -p 'test_*.py' -v
```

The legacy `sdk/python/tests` suite separately requires pytest and PyBluez and is
not covered by these checks. Never install its system dependencies as root merely
to contribute documentation. [Legacy caveats](sdk/python/README.md).

## Expectations

- Reuse existing interfaces and dependencies; avoid unrelated cleanup.
- Add behavioral regression tests for bug fixes. For transport changes, include
  cancellation, partial reads/writes, bounds, and peer-disconnect cases.
- A stub test is not a hardware test. Record OS, adapter, peer, commands, and
  redacted logs using [the hardware template](docs/HARDWARE_TESTING.md).
- Keep the historical Go import path until a migration is reviewed.
- Do not add custom cryptography or claim secure messaging without review.
- Never commit API keys, real device identifiers, chat logs, or generated plans.
- Cite upstream sources for compatibility claims; do not copy another project's
  code without its license and required notices.

Report ordinary bugs using the issue forms. For sensitive reports, follow
[SECURITY.md](SECURITY.md). Be respectful under the [code of conduct](CODE_OF_CONDUCT.md).
