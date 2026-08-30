# Roadmap

Build a useful nearby-messaging SDK in testable stages. There are no promised
release dates. Completed source code, mock tests, and hardware evidence are
separate milestones; this repository does not claim production maturity yet.

## 0 — Trustworthy developer foundation (current work)

- Accurate README, runnable stub demo, contribution/security guidance and CI.
- Replay consent based on registered tools, offline preview, stop-on-error behavior.
- Local AI draft example with explicit review/send separation.
- Before a stable release: safe daemon lifecycle, bounded RFCOMM I/O, Python
  serialization/framing migration, API compatibility, and physical-device results.

## 1 — Two Linux devices exchanging messages

Add receive/session semantics through the existing daemon and transport boundary.
Acceptance: two actual Linux/Raspberry Pi peers exchange ordered UTF-8 messages;
partial frames, disconnects, cancellation, limits, and slow readers are tested;
resources close predictably; a reproducible hardware report is checked in.
A local write is not advertised as remote delivery. Define acknowledgements first.

## 2 — A minimal BLE chat profile

Prototype **Linux first**, evaluating maintained bindings before adding dependencies.
Specify service/characteristic UUIDs, version negotiation, framing, chunk/reassembly
limits, write/notify flow control, and central/peripheral roles. Do not adapt the
current integer RFCOMM channel field into an undocumented GATT protocol.
Acceptance: two real peers, at least one mobile BLE test client, negotiated payload
limits, malformed/duplicate/missing fragment tests, reconnects, and documented
foreground/background constraints. BLE support does not imply mesh support.

## 3 — Optional local AI in a chat app

Reuse the draft/review flow. Add receive history only after session semantics exist.
Acceptance: opt-in inference, installed local model with cloud features disabled,
o automatic sends, bounded history, clear latency/cancellation, and deterministic
fake-model tests. Explicitly document model licensing and hardware requirements.

## 4 — Compatibility and security

Evaluate Bitchat interoperability against a pinned upstream protocol/version and
test vectors. Use reviewed cryptographic libraries; require peer identity,
replay/duplicate handling, hostile input tests, and independent security review.
Do not promise interoperable or secure messaging until those gates pass.

## Later, only with a demonstrated need

macOS/Windows radio backends, mobile bindings, Rust SDK, resumable file transfer,
mesh/store-and-forward, REST/gRPC facades. Placeholder directories are not releases.
Avoid building more services before one chat path works on real hardware.

[Open contributor work](docs/ISSUES.md) · [Architecture rationale and sources](docs/CHAT_AND_LOCAL_AI.md)
