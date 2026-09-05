# Local AI draft → review → explicit send

A small working building block for a future chat app. **This is not two-way
Bluetooth chat.** The daemon has no receive/session API yet. This example creates
a local text draft, which you review and can send with the existing one-shot CLI.
It needs Python 3.10+ but no pip packages. Tests use a fake HTTP server, not an LLM.

## 1. Prepare a local model

Install [Ollama](https://docs.ollama.com/quickstart), download a model suitable for
your hardware, and run it locally. Model downloads initially require internet.
Configure the server with `OLLAMA_NO_CLOUD=1` and restart it before using private
text; use an installed local model shown by `ollama list`. See the
[Ollama local-only instructions](https://docs.ollama.com/faq#how-do-i-disable-ollamas-cloud-features).

The example connects only to `127.0.0.1:11434/api/generate`, disables streaming,
and has no tools, shell execution, radio code, or auto-send. A loopback URL alone
does not prove the server avoids cloud services; you control that server setup.

## 2. Draft and review

From the repository root, replace `YOUR_LOCAL_MODEL` with its exact installed name:

```bash
python3 examples/chat/local_draft.py --model YOUR_LOCAL_MODEL \
  --output draft.txt 'Write a short message asking where we should meet.'
cat draft.txt
```

The output file is created privately and is never overwritten. Keep drafts out of
Git. The helper caps prompt/response size, rejects incomplete responses and control
characters, and fails rather than silently truncating an oversized draft. It does
not validate the meaning or safety of the model's text. Review it yourself.

## 3. Send only if you choose

With the [stub quick start](../../README.md#quick-start-without-bluetooth-hardware)
running, explicitly send the reviewed file:

```bash
./bin/ubt send --address AA:BB:CC:DD:EE:01 --file draft.txt
```

This simulates sending; nothing reaches another device. Real RFCOMM requires Linux,
a compatible receiver, pairing/permissions, and its actual address/channel.
Do not pipe model output directly into radio commands.

## Test without Ollama or Bluetooth

```bash
python3 -m unittest discover -s examples/chat -p 'test_*.py' -v
```

[Chat / BLE design and prerequisites](../../docs/CHAT_AND_LOCAL_AI.md).
The [Ollama generate API](https://docs.ollama.com/api/generate) is its native API,
not the separate OpenAI-compatible endpoint.
