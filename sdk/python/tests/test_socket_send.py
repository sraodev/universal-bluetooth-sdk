"""Regression tests for issue #28: a short write must not truncate a sent frame.

Both ``SocketManager.send`` (socket_manager.py) and ``ClientSocketManager.send``
(client_socket.py) import the ``bluetooth`` (PyBluez) package. The focused tests
use the real package when available and otherwise install a scoped fallback with
the names required by the modules under test.
"""

from __future__ import annotations

import sys
import types

import pytest


def _install_bluetooth_stub() -> None:
    stub = types.ModuleType("bluetooth")
    stub.BluetoothError = type("BluetoothError", (OSError,), {})
    stub.BluetoothSocket = object
    stub.RFCOMM = 3
    stub.PORT_ANY = 0
    stub.SERIAL_PORT_CLASS = "1101"
    stub.SERIAL_PORT_PROFILE = "1101"
    stub.advertise_service = lambda *args, **kwargs: None
    stub.find_service = lambda *args, **kwargs: []
    sys.modules["bluetooth"] = stub


@pytest.fixture(scope="module")
def manager_types():
    modules_before = set(sys.modules)
    using_stub = False
    try:
        import bluetooth  # noqa: F401
    except ModuleNotFoundError as exc:
        if exc.name != "bluetooth":
            raise
        _install_bluetooth_stub()
        using_stub = True

    try:
        from bluetooth_service.client_config import ClientSettings
        from bluetooth_service.client_socket import ClientSocketManager
        from bluetooth_service.exceptions import BluetoothServerError
        from bluetooth_service.socket_manager import SocketManager

        yield ClientSettings, ClientSocketManager, BluetoothServerError, SocketManager
    finally:
        if using_stub:
            for module_name in set(sys.modules) - modules_before:
                if module_name == "bluetooth" or module_name.startswith("bluetooth_service"):
                    sys.modules.pop(module_name, None)


class ShortWriteSocket:
    """A fake socket whose ``send`` only ever writes a bounded prefix.

    ``sendall`` is implemented the way the real ``socket.sendall`` behaves: it
    loops over (possibly short) ``send`` calls until the whole buffer is
    written. This lets the same short-write ``send`` faithfully expose the bug
    (a single unchecked ``send`` call silently drops the tail of the buffer)
    and the fix (``sendall`` keeps writing until nothing is left).
    """

    def __init__(self, chunk_size: int = 4) -> None:
        self.chunk_size = chunk_size
        self.sent = bytearray()

    def send(self, data: bytes) -> int:
        chunk = data[: self.chunk_size]
        self.sent.extend(chunk)
        return len(chunk)

    def sendall(self, data: bytes) -> None:
        view = memoryview(data)
        while view:
            written = self.send(view.tobytes())
            view = view[written:]


class RaisingSocket:
    """A fake socket whose ``sendall`` always fails like a dropped connection."""

    def __init__(self, exc: BaseException) -> None:
        self._exc = exc

    def send(self, data: bytes) -> int:
        raise self._exc

    def sendall(self, data: bytes) -> None:
        raise self._exc


PAYLOAD = b"a payload longer than one short-write chunk"


def test_socket_manager_send_delivers_full_payload_through_a_short_writer(manager_types) -> None:
    _, _, _, SocketManager = manager_types
    manager = SocketManager()
    fake_socket = ShortWriteSocket(chunk_size=4)
    manager._client_socket = fake_socket  # noqa: SLF001 - direct fixture wiring, no live connect available

    manager.send(PAYLOAD)

    assert bytes(fake_socket.sent) == PAYLOAD


def test_socket_manager_send_wraps_failure_and_keeps_the_cause(manager_types) -> None:
    _, _, BluetoothServerError, SocketManager = manager_types
    manager = SocketManager()
    original = OSError("connection reset by peer")
    manager._client_socket = RaisingSocket(original)  # noqa: SLF001

    with pytest.raises(BluetoothServerError) as exc_info:
        manager.send(PAYLOAD)

    assert exc_info.value.__cause__ is original


def test_client_socket_manager_send_delivers_full_payload_through_a_short_writer(manager_types) -> None:
    ClientSettings, ClientSocketManager, _, _ = manager_types
    client = ClientSocketManager(ClientSettings())
    fake_socket = ShortWriteSocket(chunk_size=4)
    client._socket = fake_socket  # noqa: SLF001 - direct fixture wiring, no live connect available

    client.send(PAYLOAD)

    assert bytes(fake_socket.sent) == PAYLOAD


def test_client_socket_manager_send_wraps_failure_and_keeps_the_cause(manager_types) -> None:
    ClientSettings, ClientSocketManager, BluetoothServerError, _ = manager_types
    client = ClientSocketManager(ClientSettings())
    original = OSError("connection reset by peer")
    client._socket = RaisingSocket(original)  # noqa: SLF001

    with pytest.raises(BluetoothServerError) as exc_info:
        client.send(PAYLOAD)

    assert exc_info.value.__cause__ is original
