"""Unit tests for the Bluetooth server orchestration."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, List, Optional, Tuple

import pytest

from bluetooth_service.config import ServerSettings
from bluetooth_service.exceptions import BluetoothServerError
from bluetooth_service.server import BluetoothServer


class StubSocketManager:
    """Test double that mimics the SocketManager contract."""

    def __init__(self, payloads: List[bytes]):
        self.payloads = payloads
        self.sent_messages: List[bytes] = []
        self.opened = False
        self.bound: Optional[Tuple[str, int, Optional[int]]] = None
        self.advertised: Optional[Tuple[str, str, bool]] = None
        self.accept_timeout: Optional[float] = None
        self.closed = False

    def open_server(self) -> None:
        self.opened = True

    def bind_and_listen(self, host: str, backlog: int, port: Optional[int] = None) -> int:
        self.bound = (host, backlog, port)
        return 3

    def advertise(self, service_name: str, service_id: str, advertise_profile: bool = True) -> None:
        self.advertised = (service_name, service_id, advertise_profile)

    def accept(self, timeout: Optional[float] = None) -> None:
        self.accept_timeout = timeout

    def receive(self, buffer_size: int, timeout: Optional[float] = None) -> bytes:
        if not self.payloads:
            raise AssertionError("No payloads left to return")
        return self.payloads.pop(0)

    def send(self, payload: bytes) -> None:
        self.sent_messages.append(payload if isinstance(payload, bytes) else payload.encode("utf-8"))

    def close(self) -> None:
        self.closed = True


@dataclass
class StubDeserializer:
    output: Any

    def deserialize(self, payload: bytes) -> Any:
        return self.output


class StubSink:
    def __init__(self) -> None:
        self.persisted: List[Any] = []

    def persist(self, obj: Any) -> None:
        self.persisted.append(obj)


def test_server_receives_and_persists_payload() -> None:
    payload = b"11:hello worldEXTRA"
    socket_manager = StubSocketManager(payloads=[payload])
    sink = StubSink()
    deserializer = StubDeserializer(output={"message": "hello world"})
    server = BluetoothServer(
        ServerSettings(),
        deserializer=deserializer,
        sink=sink,
        socket_manager=socket_manager,
    )

    server.start()
    result = server.receive_once()
    server.stop()

    assert socket_manager.opened
    assert socket_manager.bound == ("", 1, None)
    assert socket_manager.advertised == ("BluetoothServices", "94f39d29-7d6d-437d-973b-fba39e49d4ee", True)
    assert socket_manager.sent_messages[-1] == b"DataReceived"
    assert sink.persisted == [{"message": "hello world"}]
    assert result == {"message": "hello world"}
    assert socket_manager.closed


def test_server_requests_retry_on_corrupt_payload() -> None:
    # First payload smaller than declared size -> should trigger resend request.
    socket_manager = StubSocketManager(
        payloads=[
            b"10:short",
            b"4:data",
        ]
    )
    sink = StubSink()
    server = BluetoothServer(
        ServerSettings(),
        deserializer=StubDeserializer(output={"message": "data"}),
        sink=sink,
        socket_manager=socket_manager,
    )

    server.start()
    server.receive_once()
    server.stop()

    # First send should be a resend request, second should be ack.
    assert socket_manager.sent_messages[0] == b"CorruptedBufferResend"
    assert socket_manager.sent_messages[-1] == b"DataReceived"
    assert sink.persisted == [{"message": "data"}]


@pytest.mark.parametrize(
    "malformed_frame",
    [
        b"no-delimiter-here",
        b":data",
        b"abc:data",
        b"-1:data",
    ],
)
def test_server_requests_resend_on_invalid_length_prefix(
    malformed_frame: bytes,
) -> None:
    socket_manager = StubSocketManager(
        payloads=[
            malformed_frame,
            b"4:data",
        ]
    )
    sink = StubSink()
    server = BluetoothServer(
        ServerSettings(),
        deserializer=StubDeserializer(output={"message": "data"}),
        sink=sink,
        socket_manager=socket_manager,
    )

    server.start()
    result = server.receive_once()
    server.stop()

    assert socket_manager.sent_messages == [
        b"DelimiterMissingBufferResend",
        b"DataReceived",
    ]
    assert sink.persisted == [{"message": "data"}]
    assert result == {"message": "data"}


def test_server_stops_after_bounded_resend_attempts() -> None:
    # A fragmented/desynchronized stream must fail loudly instead of retrying
    # forever and emitting an unbounded number of control messages.
    socket_manager = StubSocketManager(
        payloads=[
            b"800:" + (b"x" * 664),
            b"x" * 136,
            b"x" * 136,
            b"x" * 136,
        ]
    )
    sink = StubSink()
    server = BluetoothServer(
        ServerSettings(max_resend_attempts=3),
        deserializer=StubDeserializer(output={"message": "data"}),
        sink=sink,
        socket_manager=socket_manager,
    )

    server.start()
    with pytest.raises(BluetoothServerError, match="after 3 resend attempts"):
        server.receive_once()
    server.stop()

    assert socket_manager.sent_messages == [
        b"CorruptedBufferResend",
        b"DelimiterMissingBufferResend",
        b"DelimiterMissingBufferResend",
    ]
    assert sink.persisted == []
