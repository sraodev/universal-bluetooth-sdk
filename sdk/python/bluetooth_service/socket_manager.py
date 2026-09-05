"""Encapsulation of the Bluetooth socket lifecycle."""

from __future__ import annotations

import logging
from typing import Optional, Tuple

from bluetooth import (
    BluetoothError,
    BluetoothSocket,
    PORT_ANY,
    RFCOMM,
    SERIAL_PORT_CLASS,
    SERIAL_PORT_PROFILE,
    advertise_service,
)

from .exceptions import BluetoothServerError

logger = logging.getLogger(__name__)


class SocketManager:
    """Facade over low-level Bluetooth socket operations (Facade pattern)."""

    def __init__(self) -> None:
        self._server_socket: Optional[BluetoothSocket] = None
        self._client_socket: Optional[BluetoothSocket] = None

    @property
    def server_socket(self) -> BluetoothSocket:
        if self._server_socket is None:
            raise BluetoothServerError("Server socket not initialized")
        return self._server_socket

    @property
    def client_socket(self) -> BluetoothSocket:
        if self._client_socket is None:
            raise BluetoothServerError("Client socket not connected")
        return self._client_socket

    def open_server(self) -> None:
        try:
            self._server_socket = BluetoothSocket(RFCOMM)
            logger.info("RFCOMM server socket created")
        except (BluetoothError, OSError) as exc:
            raise BluetoothServerError("Unable to create server socket", cause=exc)

    def bind_and_listen(self, host: str, backlog: int, port: Optional[int] = None) -> int:
        try:
            bind_port = PORT_ANY if port is None else port
            self.server_socket.bind((host, bind_port))
            self.server_socket.listen(backlog)
            port = self.server_socket.getsockname()[1]
            logger.info("Listening for connections on RFCOMM channel %s", port)
            return port
        except (BluetoothError, OSError) as exc:
            raise BluetoothServerError("Unable to bind or listen", cause=exc)

    def advertise(
        self,
        service_name: str,
        service_id: str,
        advertise_profile: bool = True,
    ) -> None:
        if not advertise_profile:
            return
        try:
            advertise_service(
                self.server_socket,
                service_name,
                service_id=service_id,
                service_classes=[service_id, SERIAL_PORT_CLASS],
                profiles=[SERIAL_PORT_PROFILE],
            )
            logger.info("%s advertised successfully", service_name)
        except (BluetoothError, OSError) as exc:
            raise BluetoothServerError("Unable to advertise service", cause=exc)

    def accept(
        self,
        timeout: Optional[float] = None,
    ) -> Tuple[BluetoothSocket, Tuple[str, int]]:
        try:
            if timeout is not None:
                self.server_socket.settimeout(timeout)
            self._client_socket, client_info = self.server_socket.accept()
            logger.info("Accepted connection from %s", client_info)
            return self.client_socket, client_info
        except (BluetoothError, OSError) as exc:
            raise BluetoothServerError("Unable to accept connection", cause=exc)
        finally:
            # Restore blocking mode for subsequent operations
            try:
                self.server_socket.settimeout(None)
            except (BluetoothError, OSError):
                # Non-fatal; closing will be attempted later.
                pass

    def receive(self, buffer_size: int, timeout: Optional[float] = None) -> bytes:
        try:
            if timeout is not None:
                self.client_socket.settimeout(timeout)
            return self.client_socket.recv(buffer_size)
        except (BluetoothError, OSError) as exc:
            raise BluetoothServerError("Unable to receive data", cause=exc)
        finally:
            # Return to blocking mode to avoid surprising callers
            try:
                self.client_socket.settimeout(None)
            except (BluetoothError, OSError):
                pass

    def send(self, payload: str | bytes) -> None:
        try:
            buffer = payload.encode("utf-8") if isinstance(payload, str) else payload
            self.client_socket.sendall(buffer)
        except (BluetoothError, OSError) as exc:
            raise BluetoothServerError("Unable to send data", cause=exc)

    def close(self) -> None:
        for sock in (self._client_socket, self._server_socket):
            if sock is None:
                continue
            try:
                sock.close()
            except BluetoothError as exc:
                logger.warning("Failed to close socket cleanly: %s", exc)

