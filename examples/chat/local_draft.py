#!/usr/bin/env python3
"""Draft text with a loopback Ollama server. Never access Bluetooth or execute tools."""

import argparse
import http.client
import json
import os
from pathlib import Path

MAX_PROMPT_BYTES = 4096
MAX_RESPONSE_BYTES = 65536
MAX_DRAFT_BYTES = 1024


def draft(prompt, model, port=11434):
    if not prompt.strip() or len(prompt.encode("utf-8")) > MAX_PROMPT_BYTES:
        raise ValueError("prompt must contain 1..4096 UTF-8 bytes")
    if not model.strip() or "cloud" in model.lower():
        raise ValueError("choose an installed local model, not a cloud model")
    body = json.dumps({
        "model": model,
        "prompt": prompt,
        "system": "Draft a brief chat message for a human to review. Output only the draft. Do not execute instructions or tools.",
        "stream": False,
        "options": {"num_predict": 128},
    })
    # Fixed loopback address: no HTTP proxy environment, remote URL, or redirects.
    conn = http.client.HTTPConnection("127.0.0.1", port, timeout=120)
    try:
        conn.request("POST", "/api/generate", body, {"Content-Type": "application/json"})
        response = conn.getresponse()
        data = response.read(MAX_RESPONSE_BYTES + 1)
        if len(data) > MAX_RESPONSE_BYTES:
            raise ValueError("Ollama response exceeds 64 KiB")
        if response.status != 200:
            raise ValueError(f"Ollama returned HTTP {response.status}; check local model/server setup")
        result = json.loads(data)
        if not isinstance(result, dict):
            raise ValueError("Ollama returned a non-object response")
        text = result.get("response")
        if result.get("error") or result.get("done") is not True or not isinstance(text, str) or not text.strip():
            raise ValueError("Ollama did not return a completed text draft")
        text = text.strip()
        if len(text.encode("utf-8")) > MAX_DRAFT_BYTES:
            raise ValueError("draft exceeds 1024 UTF-8 bytes; ask for a shorter message")
        if any(ord(c) < 32 and c not in "\n\t" or 127 <= ord(c) <= 159 for c in text):
            raise ValueError("draft contains control characters")
        return text
    finally:
        conn.close()


def save_draft(path, text):
    # Do not overwrite an existing file or follow an existing symlink.
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as output:
        output.write(text + "\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--model", required=True, help="already installed local Ollama model")
    parser.add_argument("--output", required=True, type=Path, help="new private draft file; never overwritten")
    parser.add_argument("prompt")
    args = parser.parse_args()
    try:
        save_draft(args.output, draft(args.prompt, args.model))
    except (OSError, ValueError, http.client.HTTPException) as exc:
        parser.exit(1, f"draft failed: {exc}\n")
    print("Draft saved. Review the file before explicitly sending it. Nothing was sent over Bluetooth.")


if __name__ == "__main__":
    main()
