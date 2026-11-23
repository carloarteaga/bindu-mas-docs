# Bindu Runtime Logging Model

> [!NOTE]
> **HUMAN OPERATOR / SYSADMIN REFERENCE ONLY**
> This document is intended for human system administrators and platform infrastructure operators. Autonomous AI Agents building Multi-Agent Systems (MAS) should NOT use this document and must strictly follow the Whitelisted Curriculum defined in [AGENTS.md](./AGENTS.md).

Role of this document:

- Canonical public guide for runtime logging behavior on root and node
- Best first stop for understanding stdout/stderr, journald, retention ownership, and debugging workflow

## Purpose

This document explains how runtime logs work in Bindu today.

The key rule is:

- Bindu logs to the platform by default, not to files

For normal operation, the canonical model is:

- `stdout`
- `stderr`
- collected and persisted by `systemd` / `journald`

## Logging Contract

Bindu runtime logging is declared in the central `binduConfig.json` under `"logging"`, which is uniform across Root and all edge nodes:

- `level`: Log verbosity (`DEBUG`, `INFO`, `WARN`, `ERROR`; default: `INFO`). Supports dynamic in-memory level switching via `slog.LevelVar` without process restarts.
- `format`: Structured serialization (`json` or `text`; default: `json`).
- `retention_days`: Maximum time window in days retained by the host platform (default: `7`).
- `max_size_mb`: Total disk quota ceiling in megabytes allocated to logs (default: `500`).
- `output`: Stream destination (`stdout` or `stderr`; default: `stdout`).

```json
"logging": {
  "format": "json",
  "level": "INFO",
  "retention_days": 7,
  "max_size_mb": 500
}
```

Bindu does **not** use file logging as the normal runtime model for root or node. The engine emits structured logs to stdout/stderr, where Linux `systemd` / `journald` collects, indexes, and enforces retention.

## Ownership & Maintenance Boundary

In normal operation:

- **Bindu Engine**:
  - Defines the log configuration contract (`level`, `format`, `output`, `retention_days`, `max_size_mb`).
  - **Update Compatibility & In-Memory Defaults**: In existing deployments where `runtime/config/binduConfig.json` is preserved across updates, `LoadConfigFromFile` automatically applies the production defaults (`retention_days: 7`, `max_size_mb: 500`) in memory whenever these fields are missing or unset. This ensures immediate, consistent enforcement without causing disk configuration drift against the accepted baseline. (To persist these values explicitly to disk, operators can use `bindu-studio system-config set`).
  - **Automated Retention Enforcement**: During the daily maintenance schedule (and on-demand console maintenance triggers), the engine executes `EnforceLogRetention`:
    - **Linux Systemd Service Mode**: Executes `journalctl --vacuum-time=<retention_days>d --vacuum-size=<max_size_mb>M -u <service>` to actively vacuum the journal according to policy.
    - **Standalone / File Logging Mode**: Inspects `runtime/logs/`, automatically deleting log files older than `retention_days` and trimming the oldest logs if total size exceeds `max_size_mb`.
- **Linux Infrastructure** (`systemd` / `journald`):
  - Ingests structured stdout/stderr streams.
  - Manages real-time log indexing, local queries, and disk storage.

## Standard Commands

### Root

```bash
systemctl status bindu-root
journalctl -u bindu-root -f
```

### Node

```bash
systemctl status bindu-node
journalctl -u bindu-node -f
```

## Time-Filtered Debugging

These filters are especially useful when debugging a recent failure.

### Last 2 hours

```bash
journalctl -u bindu-root --since "2 hours ago" --no-pager
journalctl -u bindu-node --since "2 hours ago" --no-pager
```

### Last 5 minutes

```bash
journalctl -u bindu-root --since "5 minutes ago" --no-pager
journalctl -u bindu-node --since "5 minutes ago" --no-pager
```

### Exact time window

```bash
journalctl -u bindu-root --since "2026-03-13 10:00:00" --until "2026-03-13 10:15:00" --no-pager
journalctl -u bindu-node --since "2026-03-13 10:00:00" --until "2026-03-13 10:15:00" --no-pager
```

### Follow only recent logs

```bash
journalctl -u bindu-root -n 200 -f
journalctl -u bindu-node -n 200 -f
```

## Mesh-Integrated Log Viewer & Diagnostic Queries

Bindu features a native, message-driven logging protocol (`LogsRequest` and `LogsResponse`) that allows operators to inspect real-time runtime and systemd logs across all gateway nodes without opening separate SSH sessions:

- **Local Root Fast-Path**: When querying the local root gateway via the Web Console, logs are retrieved directly from the host system (`local-disk`), avoiding unnecessary network hops or ACL delays.
- **Remote Gateway Mesh Routing**: For remote edge nodes (e.g. `node-1`, `bindu-pi3-uno`), the Console UI issues a correlated `LogsRequest` message across MQTT (`tx_id`) with authenticated user credentials. The remote node's `MeshLogsPlugin` fetches the requested journal window and replies with a structured `LogsResponse`.
- **Interactive UI Modal**: Accessible via the **Mesh Config** panel by clicking the `<logs>` action next to any node. Supports log-level filtering (`INFO`, `WARN`, `ERROR`, `DEBUG`), search terms, limit configuration, and one-click manual refresh.

## Recommended Retention Policy

These are operational recommendations, not hard-coded Bindu guarantees.

### Root

Recommended baseline:

- retain about 14 days of logs
- or use a bounded total journal budget of about 1-2 GB

### Node

Recommended baseline:

- retain about 3-7 days of logs
- or use a bounded total journal budget of about 200-500 MB

Nodes usually need less retained history than root because they are more storage-sensitive and are often debugged for recent events.

## Debugging Guidance

Use this sequence when investigating a recent runtime problem:

1. check service health with `systemctl status`
2. inspect the last 5 minutes of logs
3. widen to 2 hours if the event started earlier
4. use `-f` only while reproducing live behavior

For remote troubleshooting, prefer bounded time windows over unfiltered full journal dumps.

## Operational Boundaries & Non-Promises

This document notes the following operational boundaries:

- File-backed runtime logging is not enabled by default; Bindu relies on structured `stdout`/`stderr` streams.
- Retention is enforced declaratively: Bindu coordinates journal vacuuming (`journalctl --vacuum-*`) and directory hygiene (`runtime/logs/`) via its maintenance engine based on `retention_days` and `max_size_mb`.
- Any external debug-only file capture mechanisms should remain disabled by default, size-bounded, and temporary.
