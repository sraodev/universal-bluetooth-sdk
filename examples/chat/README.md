# Chat example (planned)

The headline use case for the bidirectional-session work in the
[roadmap](../../README.md#roadmap): two devices talking over RFCOMM,
optionally piping messages through a **local** LLM (Ollama) for smart
reply / translation / summarisation. **No internet required.**

## Where it fits

This example consumes the planned `Listen` / `Reply` / `CloseSession`
RPCs (roadmap phase 2). Because they're regular methods on `ubtd`, the
example doesn't talk to the radio directly — it uses `ubtctl chat`,
which is just another inbound adapter on top of the existing wire
protocol.

```
device A:                                    device B:
  ubtctl chat serve   ◄─── RFCOMM ────────►   ubtctl chat connect AA:BB:...
        │                                          │
        ▼                                          ▼
  ubtd (linuxrfcomm)                         ubtd (linuxrfcomm)
        │
        └─ optional: pipe each incoming message through Ollama for
                     translation / smart-reply / summary
```

## Anticipated CLI shape

```bash
# Serve side
ubtctl chat serve --channel 22 \
                  --name "MyChat" \
                  --ai ollama://llama3.2 \
                  --suggest

# Client side
ubtctl chat connect AA:BB:CC:DD:EE:01 --channel 22

# Inside the chat TUI:
#   /summarize         summarise the last 50 messages
#   /translate fr      auto-translate incoming messages
#   /suggest           propose 3 replies, pick with arrow keys
#   /smart-reply on    auto-respond when away (explicit opt-in)
```

## Required upstream pieces

- **Phase 2 of the roadmap** — `Listen` / `Reply` / `CloseSession` RPCs and the
  `transport.Listener` interface they sit on.
- **Phase 3 of the roadmap** — the `ubtctl chat` TUI itself.
- A local LLM that exposes an OpenAI-compatible HTTP API. Ollama is the
  reference choice (`http://127.0.0.1:11434/api/generate`) but the
  adapter will be behind a `Completer` interface, so `llama.cpp`,
  `GPT4All`, or anything else with the same shape is a drop-in.

Status: **planned**. The RFCOMM bits are the next foundation to ship.
