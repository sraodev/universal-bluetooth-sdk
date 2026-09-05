#!/usr/bin/env python3
"""Exercise built CLI/daemon binaries over a private socket; no Bluetooth or LLM."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]


def main():
    daemon = ROOT / "bin/ubtd"
    cli = ROOT / "bin/ubt"
    legacy_cli = ROOT / "bin/ubtctl"
    if not daemon.is_file() or not cli.is_file() or not legacy_cli.is_file():
        raise SystemExit("Build bin/ubtd, bin/ubt, and bin/ubtctl first; see README.md")
    # Keep the socket path short enough for macOS's sockaddr_un limit.
    with tempfile.TemporaryDirectory(prefix="ubt-", dir="/tmp") as directory:
        socket = str(Path(directory) / "daemon.sock")
        env = dict(os.environ, UBTD_SOCKET=socket)

        def run(*args, **kwargs):
            return subprocess.run([str(cli), *args], env=env, text=True,
                                  capture_output=True, timeout=15, **kwargs)

        with open(Path(directory) / "daemon.log", "w+") as log:
            process = subprocess.Popen([str(daemon), "--driver", "stub", "--socket", socket],
                                       stdout=log, stderr=log)
            try:
                deadline = time.monotonic() + 10
                while not Path(socket).exists():
                    if process.poll() is not None or time.monotonic() > deadline:
                        log.seek(0)
                        raise RuntimeError("daemon failed to start: " + log.read())
                    time.sleep(0.05)
                run("ping", check=True)
                status = run("status", check=True).stdout
                assert "ready" in status and "stub" in status, status
                peers = run("discover", "--scan-timeout", "1", check=True).stdout
                assert "stub-pi" in peers and "stub-esp32" in peers, peers
                sent = run("send", "--address", "AA:BB:CC:DD:EE:01", "--data", "hello", check=True).stdout
                assert "sent 5 bytes" in sent, sent
                sent = run("send", "--address", "AA:BB:CC:DD:EE:01", "--file", "-",
                           input="stdin", check=True).stdout
                assert "sent 5 bytes" in sent, sent
                plan = Path(directory) / "plan.json"
                plan.write_text(json.dumps({"steps":[{"tool":"send_payload", "mutating":False,
                    "arguments":{"address":"AA:BB:CC:DD:EE:01", "payload":"hello"}}]}))
                refused = run("plan", "run", str(plan))
                assert refused.returncode != 0 and "--yes" in refused.stderr, refused
                run("plan", "run", "--yes", str(plan), check=True)
                requests = [
                    {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}},
                    {"jsonrpc":"2.0","id":2,"method":"tools/list"},
                    {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_status","arguments":{}}},
                ]
                result = run("mcp", "--socket", socket, input="\n".join(map(json.dumps, requests))+"\n", check=True)
                replies = [json.loads(line) for line in result.stdout.splitlines()]
                assert len(replies) == 3 and all("result" in row for row in replies), replies
                assert replies[0]["result"]["serverInfo"]["name"] == "ubt", replies
                assert len(replies[1]["result"]["tools"]) == 5, replies
                assert not replies[2]["result"]["isError"], replies
                legacy_sent = subprocess.run(
                    [str(legacy_cli), "send", "--address", "AA:BB:CC:DD:EE:01", "--file", "-"],
                    env=env, text=True, input="alias", capture_output=True, timeout=15, check=True)
                assert "sent 5 bytes" in legacy_sent.stdout, legacy_sent
                legacy_mcp = subprocess.run(
                    [str(legacy_cli), "mcp", "--socket", socket], env=env, text=True,
                    input=json.dumps(requests[0])+"\n", capture_output=True, timeout=15, check=True)
                legacy_reply = json.loads(legacy_mcp.stdout)
                assert legacy_reply["result"]["serverInfo"]["name"] == "ubtctl", legacy_reply
            finally:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
                    raise RuntimeError("daemon did not stop within five seconds")
            run("plan", "run", "--dry-run", str(plan), check=True)
    print("PASS: ubt/ubtctl stub send + MCP, ping/status/discover, replay, shutdown")


if __name__ == "__main__":
    main()
