# File-transfer example (planned)

Reliable binary file transfer (images, firmware, logs) over RFCOMM,
chunked with sequence numbers and integrity hashes. Demonstrates how a
non-trivial application layer composes on top of the
[wire protocol](../../common/protocol/) without daemon changes.

## Where it fits

Like [`examples/chat`](../chat), this is an inbound adapter that
consumes the planned `Listen` / `Reply` RPCs. The chunk envelope schema
will land in [`common/message-schema/`](../../common/message-schema/) so
both sides agree on framing.

## Anticipated CLI shape

```bash
# Receiver
ubt file recv --channel 24 --output ./inbox/

# Sender
ubt file send --address AA:BB:CC:DD:EE:01 --channel 24 firmware.bin
```

## Anticipated chunk shape

(Lands in `common/message-schema/proto/file-chunk.proto`.)

```proto
message FileChunk {
  string transfer_id = 1;
  uint32 sequence    = 2;
  uint32 total       = 3;
  bytes  data        = 4;
  bytes  sha256      = 5;  // chunk hash; whole-file hash on the final chunk
}
```

## Required upstream pieces

- Phase 2 of the [roadmap](../../README.md#roadmap) — `Listen` / `Reply` / `CloseSession`.
- A small chunker / reassembler in [`sdk/go/pkg/`](../../sdk/go/pkg) (likely `pkg/filechunks/`).
- The chunk schema in `common/message-schema/`.

Status: **planned**.
