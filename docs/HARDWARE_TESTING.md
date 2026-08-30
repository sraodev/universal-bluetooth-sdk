# Hardware validation

No physical Bluetooth validation was performed for the repository-maturity update.
Do not treat compilation, stub outputs, or a mocked local-model response as hardware
or inference evidence. Use devices and networks you have permission to test.

Copy this into a hardware-report issue:

```text
Commit:
Host OS / kernel / architecture:
Go version / BlueZ version:
Bluetooth adapter / firmware:
Peer hardware / receiver program and version:
Transport / RFCOMM channel:
Pairing and permissions (do not include keys):
Exact commands (redact device addresses):
Expected sender behavior:
Actual sender output:
Actual receiver bytes and acknowledgement, if any:
Disconnect / unavailable peer / cancellation results:
Repeated runs and observed failures:
Latency method, sample count and distribution (if measured):
Known limitations:
```

For Linux RFCOMM, first verify the stub demo, then use `bluetoothctl` interactively
to inspect and pair a consenting peer. The peer must already run an RFCOMM receiver
on the selected channel. `discover` currently lists BlueZ-known devices, even if
those peers are not presently reachable. `send` is raw bytes, not the legacy Python
framing/pickle protocol. Never use the legacy pickle receiver with untrusted peers.

Record receiver-side evidence; sender byte counts alone are insufficient. Start
with tiny synthetic text. Stop if a process requires unexpected privileges or a
radio call fails to cancel. No range, throughput, background execution, or mobile
compatibility promise should be made without reproducible measurements.
