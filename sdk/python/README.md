# Legacy Python RFCOMM implementation

> **Not production-ready.** Default serialization uses pickle, which can execute
> code when receiving untrusted payloads. Use only isolated, trusted lab peers.
> The stream framing also assumes more about read boundaries than a robust
> transport should. See [security guidance](../../SECURITY.md).

This historical PyBluez client/server is separate from the Go daemon. It is not
an automatic cross-platform fallback or a Go wire-protocol client. No Python
bridge is implemented. Its tests cover orchestration with stubs, not hardware or
wire conformance. Both peers must use compatible framing and serializers.

## Layout

- `bluetooth_service/` — reusable package: server/client orchestration,
  serializers, storage adapters, socket facades, structured exceptions.
- `run_server.py`, `run_client.py` — batteries-included entry points.
- `scripts/install_dependencies.sh` — installs BlueZ and PyBluez (the
  latter from the GitHub source, since the PyPI build has been broken
  on Python 3.10+ for years).
- `tests/` — pytest suite with transport stubs.

## Historical lab setup (review before running)

Review the installer first: it changes system Bluetooth packages and Python setup.
Prefer an isolated virtual environment for Python packages; do not blindly run
the script on a workstation. This setup was not hardware-validated in this update.

```bash
sudo ./scripts/install_dependencies.sh
python3 run_server.py      # Raspberry Pi / receiver
python3 run_client.py      # sender
```

Run from within `sdk/python` or adjust your `PYTHONPATH` if launching elsewhere.

> **PyBluez install note.** PyPI's `pybluez` package hasn't been updated
> in years and no longer builds on Python 3.10+. Install from the GitHub
> source instead (the helper script does this for you):
>
> ```bash
> python3 -m pip install git+https://github.com/pybluez/pybluez.git
> ```

## Tests

```bash
python3 -m pip install pytest
pytest tests/
```

## Customization

- Update `bluetooth_service/config.py` (`ServerSettings`) for server behavior.
- Update `bluetooth_service/client_config.py` (`ClientSettings`) for discovery,
  retries (both `discovery_retries` and `connect_retries`), and payload source.
- Swap serializers/sinks/sources by injecting your own implementations when
  constructing `BluetoothServer` / `BluetoothClient`.

