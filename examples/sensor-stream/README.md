# Sensor-stream example (planned)

Stream structured telemetry (temperature, humidity, IMU readings) from
an embedded device to a gateway over RFCOMM, into a time-series sink.
Demonstrates the **continuous** flavour of bidirectional sessions —
the device writes forever, the gateway listens forever.

## Where it fits

```
sensor (Pi/ESP32)              gateway
  ubtctl sensor stream    ────► ubtd (linuxrfcomm)  ──► storage adapter
                                      │                  (e.g. InfluxDB,
                                      │                   timescaledb,
                                      │                   sqlite, S3 parquet)
                                      └─ optional: AI summariser via ubtctl ask
```

The storage step reuses the `Storage` interface that already exists in
[`sdk/python/bluetooth_service/storage.py`](../../sdk/python/bluetooth_service/storage.py)
(or its Go equivalent under [`sdk/go/pkg/`](../../sdk/go/pkg/) when that
ships).

## Anticipated record shape

(Lands in `common/message-schema/json/sensor-record.schema.json`.)

```jsonc
{
  "device_id": "esp32-livingroom",
  "timestamp_unix_ms": 1719876543210,
  "metric": "temperature_c",
  "value": 21.3,
  "tags": { "location": "livingroom" }
}
```

## Required upstream pieces

- Phase 2 of the [roadmap](../../README.md#roadmap) — `Listen` / `Reply`.
- Record schema in `common/message-schema/`.
- A small `ubtctl sensor` subcommand on the device side.
- A storage-adapter selection mechanism on the gateway (env var or
  flag pointing at a sink URL).

Status: **planned**.
