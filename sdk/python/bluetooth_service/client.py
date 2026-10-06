"""High-level Bluetooth client implementation mirroring the server SDK."""

from __future__ import annotations

import logging
from typing import Any, Optional

from .client_config import ClientSettings
from .client_socket import ClientSocketManager
from .exceptions import BluetoothServerError
from .interfaces import DataSource, Serializer

logger = logging.getLogger(__name__)


class BluetoothClient:
    """Coordinates discovery, connection, serialization, and sending."""

    def __init__(
        self,
        settings: Optional[ClientSettings] = None,
        *,
        serializer: Serializer,
        source: DataSource,
        socket_manager: Optional[ClientSocketManager] = None,
    ) -> None:
        self.settings = settings or ClientSettings()
        self._serializer = serializer
        self._source = source
        self._socket_manager = socket_manager or ClientSocketManager(self.settings)

    def start(self) -> None:
        logger.debug("Starting Bluetooth client with settings: %s", self.settings)
        self._socket_manager.discover()
        self._socket_manager.connect()

    def send_once(self) -> Any:
        logger.debug("Loading payload from data source")
        obj = self._source.load()
        payload = self._serializer.serialize(obj)
        framed_payload = self._frame_payload(payload)
        logger.debug("Sending framed payload of %s bytes", len(framed_payload))
        self._socket_manager.send(framed_payload)
        self._await_ack(payload)
        return obj

    def stop(self) -> None:
        self._socket_manager.close()

    # Internals -----------------------------------------------------------------
    def _frame_payload(self, payload: bytes) -> bytes:
        prefix = f"{len(payload)}:".encode("utf-8")
        return prefix + payload

    def _await_ack(self, payload: bytes) -> None:
        resend_attempts = 0
        while True:
            response = self._socket_manager.receive(
                self.settings.buffer_size,
                timeout=self.settings.receive_timeout,
            )
            if response.decode("utf-8") in {
                self.settings.resend_empty_message,
                self.settings.resend_corrupt_message,
                self.settings.resend_incomplete_message,
                self.settings.delimiter_missing_message,
            }:
                if resend_attempts >= self.settings.max_resend_attempts:
                    raise BluetoothServerError(
                        f"Server still requested a resend after "
                        f"{self.settings.max_resend_attempts} attempts: {response!r}"
                    )
                resend_attempts += 1
                logger.warning("Server requested retransmit: %s", response)
                self._socket_manager.send(self._frame_payload(payload))
                continue
            if response.decode("utf-8") == self.settings.acknowledge_message:
                logger.info("Server acknowledged payload")
                return
            raise BluetoothServerError(f"Unexpected acknowledgement: {response!r}")

