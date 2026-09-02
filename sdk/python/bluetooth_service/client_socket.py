"""Encapsulation of Bluetooth client socket responsibilities."""

from __future__ import annotations

import logging
import time
from typing import Optional

from bluetooth import (
    BluetoothError,
    BluetoothSocket,
    RFCOMM,
    find_service,
)

from .client_config import ClientSettings
from .exceptions import BluetoothServerError

logger = logging.getLogger(__name__)


class ClientSocketManager:
    """Handles discovery, socket creation, and send/receive flows."""

    def __init__(self, settings: ClientSettings) -> None:
        self._settings = settings
        self._socket: Optional[BluetoothSocket] = None
        self._service_info: Optional[dict] = None

    def discover(self) -> None:
        retries = max(1, self._settings.discovery_retries)
        for attempt in range(1, retries + 1):
            logger.info("Discovering Bluetooth service (attempt %s/%s)", attempt, retries)
            services = find_service(
                uuid=self._settings.service_uuid,
                address=self._settings.target_address,
            )
            if services:
                self._service_info = services[0]
                logger.info("Found service: %s", self._service_info.get("name"))
                return
            if attempt < retries:
                time.sleep(self._settings.discovery_backoff_seconds)
        raise BluetoothServerError("Unable to discover Bluetooth service")

    def connect(self) -> None:
        if not self._service_info:
            raise BluetoothServerError("Service discovery must run before connect")

        endpoint = (self._service_info["host"], self._service_info["port"])
        retries = max(1, self._settings.connect_retries)
        last_exc: Optional[BaseException] = None

        for attempt in range(1, retries + 1):
            try:
                self._socket = BluetoothSocket(RFCOMM)
                if self._settings.connect_timeout is not None:
                    self._socket.settimeout(self._settings.connect_timeout)
                logger.info(
                    "Connecting to %s:%s (attempt %s/%s)",
                    *endpoint,
                    attempt,
                    retries,
                )
                self._socket.connect(endpoint)
                logger.info("Connected to %s", self._service_info["name"])
                self._socket.settimeout(None)
                return
            except (BluetoothError, OSError) as exc:
                last_exc = exc
                logger.warning(
                    "Connect attempt %s/%s to %s:%s failed: %s",
                    attempt,
                    retries,
                    *endpoint,
                    exc,
                )
                if self._socket is not None:
                    try:
                        self._socket.close()
                    except BluetoothError as close_exc:
                        logger.warning("Failed to close socket after connect error: %s", close_exc)
                    finally:
                        self._socket = None
                if attempt < retries:
                    time.sleep(self._settings.connect_backoff_seconds)

        raise BluetoothServerError("Unable to connect to Bluetooth service", cause=last_exc)

    def send(self, payload: bytes) -> None:
        if not self._socket:
            raise BluetoothServerError("Client socket not initialized")
        try:
            self._socket.sendall(payload)
        except (BluetoothError, OSError) as exc:
            raise BluetoothServerError("Failed to send data", cause=exc)

    def receive(self, buffer_size: int, timeout: Optional[float] = None) -> bytes:
        if not self._socket:
            raise BluetoothServerError("Client socket not initialized")
        try:
            if timeout is not None:
                self._socket.settimeout(timeout)
            return self._socket.recv(buffer_size)
        except (BluetoothError, OSError) as exc:
            raise BluetoothServerError("Failed to receive data", cause=exc)
        finally:
            if self._socket:
                try:
                    self._socket.settimeout(None)
                except (BluetoothError, OSError) as exc:
                    logger.warning("Failed to reset socket timeout: %s", exc)

    def close(self) -> None:
        if not self._socket:
            return
        try:
            self._socket.close()
            logger.info("Bluetooth client socket closed")
        except BluetoothError as exc:
            logger.warning("Failed to close client socket cleanly: %s", exc)
        finally:
            self._socket = None

