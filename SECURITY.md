# Security

This is experimental developer tooling, not a secure messenger. There are no
supported stable security releases or promised response times yet. Use only
hardware and peers you own or have permission to test.

## Known boundaries

- The legacy Python defaults deserialize **pickle** from the peer. Untrusted
  pickle can execute code. Do not expose that receiver to untrusted devices.
  JSON-safe defaults and bounded stream parsing need a reviewed migration.
- Go RFCOMM sending provides no application identity verification, message
  encryption, replay protection, or delivery acknowledgement. Bluetooth pairing
  does not establish all of those application guarantees.
- Daemon access grants radio authority. Use a private 0700 parent directory and
  a 0600 socket, no public TCP proxy, and no root execution by default. Startup
  currently removes an existing socket path; do not share it between instances.
- The cloud planner can send unless `--dry-run` is supplied. That flag still
  permits cloud requests and daemon reads. MCP also exposes the send tool without
  a server-enforced approval gate. Treat device names and tool output as untrusted.
- Saved plans contain sensitive data; keep them private. Replay uses the live tool
  registry to enforce its `--yes` mutation gate, but is not a sandbox.
- Socket frame size is bounded; connection counts, idle clients, and Linux radio
  I/O deadlines still need hardening. Do not expose the daemon to hostile clients.

## Reporting

Do not post exploit details, credentials, or private payloads in public issues.
Use GitHub's **Security → Report a vulnerability** if enabled. If it is unavailable,
open an issue containing only a request for a private reporting channel; a
maintainer must arrange that channel before you send details. No private email
address or security response SLA is implied here.

Ordinary crashes with non-sensitive, synthetic inputs belong in the bug form.
Any future secure-chat claim requires a threat model, vetted cryptography,
negative tests, and independent review.
