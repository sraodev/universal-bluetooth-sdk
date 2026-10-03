"""Unit tests for the Bluetooth client orchestration."""

from __future__ import annotations

from typing import Any, List, Optional

from bluetooth_service.client import BluetoothClient
from bluetooth_service.client_config import ClientSettings


class StubSerializer:
    def __init__(self, payload: bytes) -> None:
        self.payload = payload
        self.called_with: Optional[Any] = None

    def serialize(self, obj: Any) -> bytes:
        self.called_with = obj
        return self.payload


class StubDataSource:
    def __init__(self, data: Any) -> None:
        self.data = data
        self.load_calls = 0

    def load(self) -> Any:
        self.load_calls += 1
        return self.data


class StubClientSocketManager:
    def __init__(self, responses: List[bytes]):
        self.responses = responses
        self.sent_payloads: List[bytes] = []
        self.discovered = False
        self.connected = False
        self.closed = False

    def discover(self) -> None:
        self.discovered = True

    def connect(self) -> None:
        self.connected = True

    def send(self, payload: bytes) -> None:
        self.sent_payloads.append(payload)

    def receive(self, buffer_size: int, timeout: Optional[float] = None) -> bytes:
        if not self.responses:
            raise AssertionError("No responses left")
        return self.responses.pop(0)

    def close(self) -> None:
        self.closed = True


def test_client_sends_payload_and_receives_ack() -> None:
    serializer = StubSerializer(payload=b"payload-bytes")
    source = StubDataSource({"message": "hi"})
    socket_manager = StubClientSocketManager(responses=[b"DataReceived"])
    client = BluetoothClient(
        ClientSettings(),
        serializer=serializer,
        source=source,
        socket_manager=socket_manager,
    )

    client.start()
    result = client.send_once()
    client.stop()

    assert socket_manager.discovered and socket_manager.connected
    assert socket_manager.sent_payloads[0] == b"13:payload-bytes"
    assert serializer.called_with == {"message": "hi"}
    assert result == {"message": "hi"}
    assert socket_manager.closed


def test_client_retries_when_server_requests_resend() -> None:
    serializer = StubSerializer(payload=b"abc")
    source = StubDataSource({"message": "retry"})
    socket_manager = StubClientSocketManager(
        responses=[
            b"EmptyBufferResend",
            b"DataReceived",
        ]
    )
    client = BluetoothClient(
        ClientSettings(),
        serializer=serializer,
        source=source,
        socket_manager=socket_manager,
    )

    client.start()
    client.send_once()
    client.stop()

    # Expect two sends due to retry.
    assert socket_manager.sent_payloads == [b"3:abc", b"3:abc"]


def test_client_retries_when_server_reports_missing_delimiter() -> None:
    serializer = StubSerializer(payload=b"abc")
    source = StubDataSource({"message": "retry"})
    socket_manager = StubClientSocketManager(
        responses=[
            b"DelimiterMissingBufferResend",
            b"DataReceived",
        ]
    )
    client = BluetoothClient(
        ClientSettings(),
        serializer=serializer,
        source=source,
        socket_manager=socket_manager,
    )

    client.start()
    client.send_once()
    client.stop()

    assert socket_manager.sent_payloads == [b"3:abc", b"3:abc"]
