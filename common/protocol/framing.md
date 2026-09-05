# Wire framing — v1

The control-plane connection between `ubt` and `ubtd` is a length-prefixed
stream of JSON envelopes over a local Unix domain socket. TCP and gRPC servers
are not implemented. The Go types are the implemented schema; `v1.proto` is a
design counterpart and does not generate the current JSON bindings.

## Frame

```
+---------------------+----------------------------+
| 4 bytes, big-endian | JSON body (UTF-8, no BOM)  |
| body length (uint32)|                            |
+---------------------+----------------------------+
```

Maximum body size is 16 MiB; the frame reader returns `frame_too_large` for larger frames. The daemon closes
the connection on framing errors; it does not send a structured RPC error for them.

## Envelope

```jsonc
{
  "id": "uuid-v4",          // correlation id, echoed in responses/events
  "kind": "request" |
          "response" |
          "event",
  "method": "Discover",     // request/event only
  "params": { ... },        // request/event only
  "result": { ... },        // response only, on success
  "error":  {               // response only, on failure
    "code": "not_implemented",
    "message": "...",
    "details": { ... }
  }
}
```

Streaming responses (e.g., `Discover`) are delivered as a sequence of `event`
frames sharing the originating request's `id`, terminated by a final
`response` frame (empty `result`) or an `error`.

## Methods (v1)

| Method         | Streaming | Purpose                              |
|----------------|-----------|--------------------------------------|
| `Ping`         | no        | Liveness + clock skew check          |
| `Version`      | no        | Daemon + protocol version            |
| `Capabilities` | no        | Per-transport feature matrix         |
| `Discover`     | server    | Emit `Device` events until timeout   |
| `Send`         | no        | One-shot payload to a peer           |
| `Status`       | no        | Daemon health + active sessions      |

## Error codes

Stable, machine-readable. Add new codes; never repurpose existing ones.

| Code                | Meaning                                       |
|---------------------|-----------------------------------------------|
| `not_implemented`   | Method known to schema, no driver supports it |
| `unknown_method`    | Method not in schema                          |
| `invalid_params`    | Schema mismatch / required field missing      |
| `transport_error`   | Underlying driver failure (BlueZ, WinRT, ...) |
| `not_found`         | Address/session not known                     |
| `frame_too_large`   | Payload exceeded 16 MiB cap                   |
| `internal`          | Unexpected daemon error                       |
