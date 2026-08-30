# Can this SDK power BLE chat, Bitchat, or local AI chat?

**Direction: yes. Complete app foundation today: not yet.** The Go daemon exposes
one-shot sending, not receiving, listening, or persistent peer sessions. The
local AI example is a draft helper, not a messaging network.

The recommended product focus is **a small SDK for nearby messaging apps with
optional local AI**. This is a project recommendation, not a claim of market
validation. Keep app UI, identity, model execution, and transport independently
replaceable without adding another service or framework now.

## Responsibilities

```text
App UI / history / explicit consent
                |
       message/session API (planned)
                |
       ubtd + transport driver
         /                 \
  RFCOMM (send today)    BLE GATT (planned)

Optional local model -> draft -> human review -> app send
```

The model runs on a phone, laptop, or Raspberry Pi with enough resources. Bluetooth
carries message bytes; it does not provide inference or distribute model weights.
Keeping the model on the same device avoids making a gateway a hidden dependency.
A future gateway must disclose who can read the text and when internet is needed.

## Choices informed by upstream projects

Research checked 2026-08-30. Upstream capabilities may change; pin versions when
implementing an adapter.

| Project / API | What to learn or reuse | Boundary |
|---|---|---|
| [Bitchat protocol](https://github.com/permissionlesstech/bitchat/blob/main/WHITEPAPER.md) | Existing BLE messaging, identity/security, routing and delivery semantics | An application protocol to implement and test, not a drop-in RFCOMM peer. No affiliation or compatibility claim. |
| [TinyGo Bluetooth](https://github.com/tinygo-org/bluetooth) | Evaluate its Go API and per-platform central/peripheral support before writing OS bindings | Platform roles and build requirements differ; a hardware spike must establish fit. |
| [BlueZ GATT API](https://bluez.readthedocs.io/en/latest/gatt-api/) | Linux GATT services, characteristics, and registration | Requires an explicit profile, permissions, lifecycle and notifications. RFCOMM is a different transport. |
| [Bleak](https://bleak.readthedocs.io/en/latest/) | Python GATT client and test-peer tooling | A client alone does not provide both sides of a peer-to-peer chat app. |
| [Ollama generate API](https://docs.ollama.com/api/generate) | Optional local draft generation via the native HTTP API | Model/server setup is separate; never let generated text execute tools or send without user action. |

## Smallest strong demo

1. Two Linux machines exchange text via an explicit session API, with receipts and
   disconnect handling. Show actual receiver-side text, not only bytes written.
2. One machine drafts a reply with a preloaded local model. The user sees and
   approves it before sending. Demonstrate with internet disconnected.
3. Replace the transport with a documented BLE profile and repeat on a tested
   Linux/mobile pair. Measure actual usable payload sizes and reconnect behavior.
4. Only then evaluate Bitchat interoperability, mesh routing, and additional OSes.

This sequence tests the product's central promise before increasing platform scope.

## Security and protocol gates

- Specify framing independently of stream reads and GATT notification boundaries.
- Bound frames, queues, peers, reassembly memory, and timeouts; test exhaustion.
- Define message IDs, acknowledgements, ordering, duplicate handling, and what
  “delivered” means. Do not silently imply exactly-once delivery.
- Define identities, trust establishment, key storage and rotation before encrypted
  chat. Do not invent cryptography or describe pairing alone as end-to-end security.
- Treat nearby device names and received text as untrusted input, including when
  a model reads them. Human confirmation is an execution boundary, not a prompt.
- Distinguish our future BLE profile from Bluetooth Mesh and from Bitchat's protocol.

See the [milestone gates](../ROADMAP.md), [security limitations](../SECURITY.md), and
[hardware evidence template](HARDWARE_TESTING.md).
