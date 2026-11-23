# Bindu Presence Model

Role of this document:

- Supporting deep-dive reference for presence semantics
- Best used alongside the routing guide when debugging lane wakeups and delivery recovery

For canonical routing behavior, start with
`public/console/docs/Bindu Routing & Mesh Delivery — User Guid.md`.

## Overview

Presence in Bindu is a minimal publication trigger.

It is not:

- a transport validator
- a lease
- a dial authority
- a connection state machine

The active runtime path is:

1. enqueue to the durable queue for the destination
2. if the lane is not waiting for presence, try to publish immediately
3. resolve the endpoint structurally from mesh/config
4. connect
5. publish the queued batch
6. close the connection
7. if retries are exhausted, stop and wait for a new presence message

## Presence Messages

There are only two presence messages:

- `NodePresence` (`node -> root`)
- `ParentPresence` (`root -> node`)

Operationally:

- `NodePresence` re-opens the `root -> child` lane for that node
- `ParentPresence` re-opens the `node -> root` lane

Emission policy:

- `NodePresence` on nodes still acts as a periodic heartbeat and may also be emitted on startup or explicit recovery triggers
- `ParentPresence` on root is emitted on startup, on explicit recovery triggers, or when the root presence snapshot changes
- `ParentPresence` is not intended to behave as a noisy periodic heartbeat when nothing changed

On nodes, if no `ParentPresence` exists locally yet (fresh install or wiped DB), the runtime may seed a minimal `ParentPresence` from installer hints (for example `runtime/runtime-info/parent/last_parent.json`) to open the first publication attempt deterministically.

These messages are triggers.
They do not mark a connection valid or invalid.

## Normal vs Waiting State

In normal operation:

- a new outgoing message is enqueued
- the scheduler attempts publication immediately

After a failed publish cycle exhausts retries:

- the lane enters `waiting for presence`
- no background reconnect loop continues
- queued messages stay durable
- the next fresh presence message re-enables another publish attempt

This is the entire control loop:

`Queue -> Attempt -> Retry -> Stop -> Wait for Presence -> Attempt again`

## Endpoint Authority & IP Resolution

The active scheduler resolves destinations dynamically:

- **Topology & Ports:** `mesh.json` defines parent-child relationships and listening channel ports (e.g., `1883`, `1885`). `mesh.json` **NEVER** contains IP addresses, hostnames, or URLs.
- **Dialing Targets (IP:port):** Resolved dynamically from the latest presence records:
  - Root resolves a child's dial address from advertised IP addresses in `NodePresence`.
  - A child resolves Root's dial address from advertised IP addresses in `ParentPresence` (or `last_parent.json` during first installation).
- **Authentication:** TLS certificates authenticate the immutable `node_id` (via TLS ServerName). Transport IP addresses can change freely without invalidating identity.

## Network Migration & IP Drift Recovery

When a mesh is relocated to a different network (e.g., moving from a development staging environment to a production site):
1. **The Drift:** Both Root and child nodes receive new IP addresses.
2. **The Stalemate:** 
   - Children attempt to dial Root at its stale lab IP; retries fail and lanes enter `waiting for presence`.
   - Root attempts to dial children at their stale lab IPs recorded in the ledger database.
   - Neither can reach the other automatically.
3. **Why `--all` Broadcast Does Not Solve Subnet Drift:**
   `rescue-parent --all` causes Root to emit `ParentPresence` using the child IPs recorded in its ledger. If the children also moved to new IPs, Root dials stale addresses and fails. `--all` is only effective when child IPs are unchanged and only Root's IP changed.
4. **Resolution via Targeted & Batch Rescue:**
   Operators supply the new IP addresses to Root via Bindu Studio:
   - **Targeted:** `bindu-studio mesh rescue-parent --node-id <id> --node-ip <new-ip> --auth <user:pass>`
   - **Batch:** `bindu-studio mesh rescue-all --nodes "node-1=10.0.5.51,node-2=10.0.5.52" --auth <user:pass>` or `--inventory <file>`
   
   Executing rescue seeds Root's router presence cache with the new IP and dispatches a fresh `ParentPresence` over mTLS, instantly awakening the node's publication lane and restoring bidirectional flow.

## Management Plane Trigger vs Data Plane Delivery

A critical distinction exists between how recovery is requested and how it is executed:

- **Management Plane (HTTP REST):** Operators run `bindu-studio mesh rescue-parent` or `bindu-studio mesh node-presence`. Studio talks to the local Root engine via authenticated HTTP REST (`bindurestapi` on port `1887`).
- **Mesh Data Plane (mTLS MQTT):** Edge nodes run **no HTTP servers**. Root constructs the native `ParentPresence` MQTT message and dispatches it over mutual TLS on port `1888` (`bindu/in`) to the target node's MQTT listener.

## Operational CLI Inspection

Operators can inspect presence state in real time using Bindu Studio:

- `bindu-studio mesh node-presence <node-id>` (alias: `mesh presence`): Inspects arrival timestamp, TTL, freshness (Active vs Expired), advertised network endpoints, and lane readiness.
- `bindu-studio mesh health`: Displays presence status across the entire mesh.

## TTL and Stored Metadata

`ttl_seconds`, `expires_at`, and cached presence records may still appear in stored messages or legacy metadata.

They are informational only.

They do not:

- block publication
- invalidate routing
- define whether communication is allowed

## Linux Recovery

On Linux, local network changes can still trigger fresh `NodePresence` emission.

That helps after:

- sleep/wake
- interface reset
- DHCP renewal

This is still only a trigger mechanism. It does not promote presence into a transport authority.

## Legacy Notes

Older logs or historical code paths may still mention:

- `source=session`
- `source=presence`
- transport-connected state
- `RouterV2`

Those are legacy references.

The active runtime path uses:

- the on-demand publish scheduler
- structural endpoint resolution from mesh/config
- short-lived `connect -> publish -> close` cycles
