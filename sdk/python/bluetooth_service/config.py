"""Configuration objects for the Bluetooth server SDK."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Optional


@dataclass(frozen=True)
class ServerSettings:
    """Value object describing how the server should behave."""

    service_name: str = "BluetoothServices"
    json_file: str = "text.json"
    uuid: str = "94f39d29-7d6d-437d-973b-fba39e49d4ee"
    buffer_size: int = 1024
    backlog: int = 1
    advertise: bool = True
    socket_host: str = ""
    port: Optional[int] = None

    # Acknowledgement / retry protocol messages
    resend_empty_message: str = "EmptyBufferResend"
    resend_corrupt_message: str = "CorruptedBufferResend"
    delimiter_missing_message: str = "DelimiterMissingBufferResend"
    acknowledge_message: str = "DataReceived"
    max_resend_attempts: int = 3

    # Timeouts (seconds). None means blocking behaviour.
    accept_timeout: Optional[float] = None
    receive_timeout: Optional[float] = None

    # Logging configuration
    logging_config_path: str = "configLogger.json"
    log_env_key: str = "LOG_CFG"

    # For future extension (correlation IDs, tenant IDs, etc.)
    extra_metadata: dict[str, str] = field(default_factory=dict)
