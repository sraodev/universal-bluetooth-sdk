"""Configuration objects for the Bluetooth client SDK."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Optional


@dataclass(frozen=True)
class ClientSettings:
    """Client-specific configuration."""

    service_uuid: str = "94f39d29-7d6d-437d-973b-fba39e49d4ee"
    target_address: Optional[str] = None
    json_file: str = "text.json"
    buffer_size: int = 1024
    discovery_retries: int = 3
    discovery_backoff_seconds: float = 0.5
    # Bluetooth connect() is flaky in practice — retry the connect step
    # with a short backoff before giving up. (Adapted from PR #1.)
    connect_retries: int = 5
    connect_backoff_seconds: float = 1.0
    resend_empty_message: str = "EmptyBufferResend"
    resend_corrupt_message: str = "CorruptedBufferResend"
    delimiter_missing_message: str = "DelimiterMissingBufferResend"
    acknowledge_message: str = "DataReceived"
    connect_timeout: Optional[float] = None
    receive_timeout: Optional[float] = None
    logging_config_path: str = "configLogger.json"
    log_env_key: str = "LOG_CFG"

