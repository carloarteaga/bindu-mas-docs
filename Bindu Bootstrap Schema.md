# Bindu Bootstrap Schema (bootstrap.json)

This document defines the schema for `bootstrap.json`.

Bootstrap is **not** mesh configuration. It is a **temporary hint file** used only to establish the first connection.

## Purpose

`bootstrap.json` provides the node with:

- its own identity (`node_id`)
- the intended parent (`parent_id`)
- a set of **explicit endpoint candidates** (`signing_candidates`, `mqtt_candidates`)
- legacy hint addresses (`candidate_parent_addrs`, optional)
- the root signing endpoint (`signing_url`, optional fallback)
- a time-to-live (`ttl_seconds`)

Addresses in this file are **hints only**. They must never be written to `mesh.json`.
Mesh IDs are **identity only** and are not used to build network URLs.
In production, name resolution should be handled by DNS or mDNS if needed.
`/etc/hosts` is acceptable only for local development.

## Schema Location

```
public/console/schema/bootstrapSchema.json
```

## Canonical Fields

Required:

- `node_id` (string)
- `parent_id` (string)
- `candidate_parent_addrs` (array of strings)
- `ttl_seconds` (integer)

Optional:

- `signing_candidates` (array of HTTPS URLs)
- `mqtt_candidates` (array of MQTT endpoints, `tcp://host:port`)
- `signing_url` (string)
- `expires_at` (RFC3339 timestamp)

## Example (Valid)

```json
{
  "node_id": "node-1",
  "parent_id": "root",
  "signing_candidates": [
    "https://192.168.12.173:8444/api/v1/tls/sign-node"
  ],
  "mqtt_candidates": [
    "tcp://192.168.12.173:1884"
  ],
  "candidate_parent_addrs": [
    "root.local:1884",
    "root:1884"
  ],
  "signing_url": "https://192.168.12.173:1887/api/v1/tls/sign-node",
  "ttl_seconds": 604800
}
```

This example validates against the schema.

## Notes

- `bootstrap.json` is **not a permanent truth source**.
- Presence messages are separate and occur only after a successful connection.
- Bootstrap does not replace DNS. It only provides first‑contact hints.
- Installers must prefer `signing_candidates` / `mqtt_candidates` when present.
