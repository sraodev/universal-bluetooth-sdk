"""High-level Bluetooth server orchestration."""

from __future__ import annotations

import logging
from typing import Any, Optional

from .config import ServerSettings
from .exceptions import BluetoothServerError
from .interfaces import DataSink, Deserializer
from .socket_manager import SocketManager

logger = logging.getLogger(__name__)


class BluetoothServer:
    """
    Coordinates socket lifecycle, data decoding, and persistence.

    The class composes smaller collaborators (Strategy + Facade) to honor SOLID.
    """

    def __init__(
        self,
        settings: Optional[ServerSettings] = None,
        *,
        deserializer: Deserializer,
        sink: DataSink,
        socket_manager: Optional[SocketManager] = None,
    ) -> None:
        self.settings = settings or ServerSettings()
        self._deserializer = deserializer
        self._sink = sink
        self._socket_manager = socket_manager or SocketManager()
        self._connected = False

    def start(self) -> None:
        """Create, bind, and optionally advertise the RFCOMM server."""
        logger.debug("Starting Bluetooth server with settings: %s", self.settings)
        self._socket_manager.open_server()
        port = self._socket_manager.bind_and_listen(
            self.settings.socket_host,
            self.settings.backlog,
            port=self.settings.port,
        )
        logger.info("Server listening on RFCOMM port %s", port)
        self._socket_manager.advertise(
            self.settings.service_name,
            self.settings.uuid,
            advertise_profile=self.settings.advertise,
        )
        self._socket_manager.accept(timeout=self.settings.accept_timeout)
        self._connected = True

    def receive_once(self) -> Any:
        """
        Receive, validate, deserialize, and persist a single payload.

        Returns the deserialized object for further processing by callers.
        """
        if not self._connected:
            raise BluetoothServerError("Server must be started before receiving data")

        raw_payload = self._receive_buffer_with_ack()
        obj = self._deserializer.deserialize(raw_payload)
        self._sink.persist(obj)
        logger.info("Payload persisted successfully")
        return obj

    def stop(self) -> None:
        """Release sockets."""
        self._socket_manager.close()
        self._connected = False
        logger.info("Bluetooth server stopped")

    # Internals -----------------------------------------------------------------

    def _receive_buffer_with_ack(self) -> bytes:
        resend_attempts = 0

        def request_resend(message: str, reason: str) -> None:
            nonlocal resend_attempts
            if resend_attempts >= self.settings.max_resend_attempts:
                raise BluetoothServerError(
                    f"Unable to recover malformed frame after "
                    f"{self.settings.max_resend_attempts} resend attempts"
                )
            resend_attempts += 1
            logger.warning(
                "%s; requesting resend (%s/%s)",
                reason,
                resend_attempts,
                self.settings.max_resend_attempts,
            )
            self._socket_manager.send(message)

        while True:
            data = self._socket_manager.receive(
                self.settings.buffer_size,
                timeout=self.settings.receive_timeout,
            )
            if not data:
                request_resend(self.settings.resend_empty_message, "Empty buffer received")
                continue

            data_size_str, sep, remainder = data.partition(b":")
            if not sep or not data_size_str.isdigit():
                # The prefix must be a bare run of ASCII digits. Besides
                # malformed input, this rejects negative lengths that would
                # otherwise be acknowledged with a truncated payload.
                request_resend(
                    self.settings.delimiter_missing_message,
                    "Missing/invalid length prefix",
                )
                continue
            data_size = int(data_size_str)

            if len(remainder) < data_size:
                request_resend(
                    self.settings.resend_corrupt_message,
                    "Corrupted buffer detected",
                )
                continue

            self._socket_manager.send(self.settings.acknowledge_message)
            logger.debug("Payload of %s bytes acknowledged", data_size)
            return remainder[:data_size]
