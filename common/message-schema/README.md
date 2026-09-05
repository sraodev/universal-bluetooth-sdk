# Message schemas — `common/message-schema/`

Reserved for **payload-shape** schemas that travel inside the Universal
Bluetooth wire protocol but aren't part of the protocol itself —
sensor-stream records, file-transfer chunk envelopes, chat message
shapes, etc. The protocol layer ([`common/protocol/`](../protocol/))
carries opaque `payload []byte`; this directory will define the JSON
Schema / proto shape of what's inside that payload, per use case.

## Status

Empty today. Schemas land here as the corresponding examples ship:

| Use case | Likely schema location | Status |
|---|---|---|
| Chat | `common/message-schema/json/chat-message.schema.json` | planned (alongside [`examples/chat`](../../examples/chat)) |
| File transfer | `common/message-schema/proto/file-chunk.proto` | planned (alongside [`examples/file-transfer`](../../examples/file-transfer)) |
| Sensor telemetry | `common/message-schema/json/sensor-record.schema.json` | planned (alongside [`examples/sensor-stream`](../../examples/sensor-stream)) |

## Where this fits in the architecture

```
ubt ──► ubtd ──► driver ──► RFCOMM peer
            │
            └─ Send.payload   = bytes
                              = JSON / proto encoded against a common/message-schema/* doc
```

Keeping payload schemas in one place means the AI planner, the typed CLI,
and any future microservice all encode/decode the same shape. Until a
schema lands here, callers are responsible for their own serialization
(the Python SDK at `sdk/python` uses `pickle`; the Go CLI's `send` verb
ships UTF-8 strings or raw bytes from a file).
