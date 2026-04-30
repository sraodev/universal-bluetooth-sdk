# Documentation

Most of the architectural detail lives next to the code it describes.
This directory is the **long-form docs index**; per-package READMEs
remain canonical for the components they live in.

## Canonical entry points

| You want to read about… | Go here |
|---|---|
| The whole project: design, structure, pros/cons, full usage walkthrough | [`/README.md`](../README.md) at the repo root |
| The wire protocol (methods, framing, error codes) | [`common/protocol/`](../common/protocol/) |
| Payload schemas (chat, file-chunk, sensor records) | [`common/message-schema/`](../common/message-schema/) |
| The daemon and CLI (typed verbs, AI planner, MCP, plan replay) | [`cli/ubtctl/`](../cli/ubtctl/) |
| The transport driver port + reference / native implementations | [`sdk/go/`](../sdk/go/) |
| The reference Python SDK (PyBluez RFCOMM client/server) | [`sdk/python/`](../sdk/python/) |
| Future microservice facades (gRPC / REST) | [`microservices/`](../microservices/) |
| Example apps (chat, file transfer, sensor stream) | [`examples/`](../examples/) |

## What lands here over time

- **Design proposals** for breaking changes (protocol v2, multi-peer
  sessions, RBAC policy) — submitted as Markdown PRs, reviewed before
  the corresponding code lands.
- **ADRs** (architecture decision records) capturing decisions whose
  rationale wouldn't survive in a commit message.
- **Onboarding** for new contributors: how to set up a dev loop, how
  to run the smoke tests, how to add a new transport driver.
- **Operations**: deployment recipes for `ubtd` (systemd unit, K8s
  manifest, container image), once they exist.

Until any of those land, this directory is intentionally light — the
top-level [`README.md`](../README.md) is the project's canonical
architecture document.
