# Python SDK

The original RFCOMM implementation that gave the project its name, kept
as the **reference SDK** in the new architecture:

- It's the canonical place to look up "what should an RFCOMM client/server
  *do*?" while we port behaviour to the Go daemon.
- It's a working escape hatch on hosts where a native Go driver isn't
  ready yet — the Go daemon could shell out to a Python SDK process to
  exercise PyBluez paths it doesn't natively support.
- Its tests double as protocol-conformance specs: any new Go driver that
  claims to implement RFCOMM should pass equivalent cases.

The new control plane (`ubtd`) does **not** depend on this SDK at
runtime; it lives in its own process and only talks to drivers that
implement the [`transport.Driver`](../go/pkg/transport/driver.go) Go
interface. See [`/README.md`](../../README.md) for the full layered
picture.

## Layout

- `bluetooth_service/` — reusable package: server/client orchestration,
  serializers, storage adapters, socket facades, structured exceptions.
- `run_server.py`, `run_client.py` — batteries-included entry points.
- `scripts/install_dependencies.sh` — installs BlueZ and PyBluez (the
  latter from the GitHub source, since the PyPI build has been broken
  on Python 3.10+ for years).
- `tests/` — pytest suite with transport stubs.

## Quick start

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
> sudo python3 -m pip install git+https://github.com/pybluez/pybluez.git
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

