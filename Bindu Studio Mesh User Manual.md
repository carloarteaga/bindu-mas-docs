# Bindu Studio Mesh User Manual

Role of this document:

- Canonical public/operator guide for mesh workflows in Bindu Studio
- Best first stop for editing, validating, and applying `mesh.json`

This guide explains how to inspect and manage `mesh.json` using Bindu Studio.
It documents current ASCII-based behavior and provides a safe operator workflow.

---

## Overview

Bindu Studio Mesh exists to make mesh topology explicit, deterministic, and auditable.
It is the authoritative CLI for inspecting and changing mesh configuration.
All outputs are ASCII-first for copy/paste, CI, and SSH usage.

Key principles:
- Studio defines. Console observes.
- No inference from missing fields.
- `mesh apply` is the only command that writes `mesh.json`.

---

## Conceptual Model

### Gateways vs References

- **gateway**: runtime node with `bindu.node_id` and required channels.
- **reference**: organizational node used for hierarchy; still part of the mesh tree.
  Reference nodes must include `bindu.node_id` and channels but **cannot** host plugins.

### Hierarchy

- Mesh is a parent/child tree.
- Gateways can be placed under reference branches for semantic hierarchy.
- A non-root gateway under another gateway branch violates the operational star rule.
- `mesh tree` is the authoritative hierarchy view.

### Operational Star (Connectivity Contract)

Runtime connectivity is modeled as an operational star:
- Exactly one root gateway exists.
- Every non-root gateway must have `root` as operational parent.
- Operational parent is resolved by walking ancestors and skipping `reference` nodes.

This means semantic hierarchy can be deep, but operational connectivity must remain root <-> gateway.

### Identifiers: node.id vs bindu.node_id vs name vs slug

These fields serve different purposes. Keeping them distinct avoids confusion:

- **node.id**: Structural tree identifier used by Studio mesh hierarchy and pickers. **Hyphens only.**
- **bindu.node_id**: Operational runtime identifier used by bundles/runtime. **Allows underscores and hyphens.** Defaults to `node.id` unless you override it.
- **name**: Human-readable label shown in trees and tables.
- **slug (optional)**: URL-friendly alias for display/docs. It does not change runtime identity.

Related runtime routing concepts:
- **identity**: `bindu.node_id` used for mesh identity and routing intent.
- **tls_server_name**: TLS verification name selected at dial time (runtime concern, not a tree field).
- **dial_address**: network destination host:port selected at dial time (runtime concern, not identity).

### In-Memory Changes and Apply

- `mesh add`, `mesh edit`, and `mesh delete` create **in-memory changes**.
- In-memory changes are persisted to `config/mesh.draft.json`
  (or `<mesh>.draft.json` when using `--mesh`).
- All mesh read commands use the draft when it exists.
- `mesh apply` validates and writes `mesh.json`, then clears the draft.

Why drafts exist:
- To let you inspect and validate changes before writing to disk.
- To keep changes visible across commands until applied.

### Management Plane (HTTP REST) vs Mesh Data Plane (mTLS MQTT)

Bindu maintains a strict architectural separation between operator management and mesh data transport:

| Plane | Protocol & Port | Access Audience | Purpose |
| :--- | :--- | :--- | :--- |
| **Management Plane** | HTTP REST (`1887` by default) | Operators, `bindu-studio`, Web Console | Administrative control, topology editing, SQL queries (`/api/v1/messages/sql`), node health inspection, and triggering remote commands. |
| **Mesh Data Plane** | 100% MQTT over mTLS (`1888` / `bindu/in`) | Mesh nodes (`bindu-root` <-> `bindu-node`) | Inter-node message delivery, sensor telemetry streams, heartbeats (`NodePresence`/`ParentPresence`), and control directives. |

**Key Operational Architectural Rules:**
1. **Edge nodes have NO HTTP or WebSocket listeners:** Edge nodes (`bindu-node`) only run the MQTT mesh protocol (`mesh_mqtt` on `1883/1888` and optional `mqtt_integration` on `1885`). They cannot be dialed or queried via HTTP.
2. **How Studio manages edge nodes:** When you execute commands like `bindu-studio mesh node-restart`, `bindu-studio mesh node-refresh`, or `bindu-studio mesh rescue-parent`, the CLI connects to the local `bindu-root` engine via authenticated HTTP REST (`bindurestapi`). The root engine then constructs and dispatches the native MQTT protocol message (`SystemNodeReset`, `EmitMessageStateRequest`, `ParentPresence`) across the encrypted mTLS Data Plane to the edge node.
3. **No HTTP Message Ingestion:** Ingress into Bindu is strictly MQTT. There is NO HTTP POST endpoint for publishing messages (`/api/v1/publish` does not exist). All telemetry enters through `bindu/in` over MQTT.

---

## Getting Started

### Default Paths

```
config/mesh.json
config/mesh.draft.json
```

### Using `--mesh` and `--base-dir`

- `--mesh` overrides the path to `mesh.json`.
- `--base-dir` overrides the Bindu base directory (auto-detected by default).

Example:

```
bindu-studio mesh tree --mesh /path/to/mesh.json
bindu-studio mesh validate --base-dir /opt/bindu
```

---

## Operator Workflow (Recommended)

1) Inspect current topology:

```
bindu-studio mesh tree
```

2) Make changes (draft-only):

```
bindu-studio mesh add
bindu-studio mesh edit
bindu-studio mesh delete
```

3) Verify draft output:

```
bindu-studio mesh tree
bindu-studio mesh gateways
bindu-studio mesh channels
bindu-studio mesh platforms
bindu-studio mesh validate
```

4) Apply to disk:

```
bindu-studio mesh apply
```

5) Restart bindu-root so the runtime loads the new mesh (auth required). Use the normal root restart entrypoint; it will use the configured restart mechanism for the current installation:

```
bindu-studio reset-bindu --auth console:supersecret
```

---

## Commands (ASCII-First)

### `mesh tree`

Shows the mesh hierarchy as an ASCII tree.
Defaults to gateways only; `--all` shows all nodes.

Examples:

```
bindu-studio mesh tree
```

Example output:

```
Mesh Tree (gateways)
root (gateway, node_id=root) - Initial Root
|-- dc-rack-room-1 (gateway, node_id=node-rack-room-1)
`-- dc-ups-room (gateway, node_id=node-ups-room-new)
```

```
bindu-studio mesh tree --all
```

Example output:

```
Mesh Tree (all)
root (gateway, node_id=root) - Initial Root
|-- rack-refs (reference)
|   `-- dc-rack-room-1 (gateway, node_id=node-rack-room-1)
`-- dc-ups-room (gateway, node_id=node-ups-room-new)
```

Notes:
- Line format: `id (connectivity_kind, node_id=...) - name (slug=...)`.
- When a draft is active, a draft notice is printed to stderr.

---

### `mesh add`

Adds a node under a selected parent via a wizard.
The parent picker is a numbered ASCII tree (no manual ID typing required).

Flow:
1) Choose a parent from the tree.
2) Enter node id, name, slug.
3) Choose connectivity kind (gateway or reference).
4) Reference ends immediately.
5) Gateway continues with platform + required channels.

Node platform options for gateway nodes:
- `linux/amd64`
- `linux/arm64`
- `linux/armv7` (Raspberry Pi 32-bit OS)

Node identity rules (important):
- `node.id` is a **slug** (hyphens only). Underscores are not allowed.
  Example: `node-mac1`
- `bindu.node_id` is an **operational id** and allows underscores and hyphens.
  Example: `node_mac1` or `node-mac1`

Examples:

```
bindu-studio mesh add
```

Parent picker example:

```
Select node to add under:
  1) root (gateway, node_id=root)
  2) |-- rack-refs (reference)
  3) |   `-- dc-rack-room-1 (gateway, node_id=node-rack-room-1)
  4) `-- dc-ups-room (gateway, node_id=node-ups-room-new)
Select number or filter
```

Notes:
- You can type a number or a substring filter (e.g., `ups`).
- The wizard prints a warning if insertion would violate operational star.
- If confirmed anyway, validation still blocks the draft write with a `mesh star violation` error.
- Draft changes are written to `config/mesh.draft.json`.

---

### `mesh bulk-add`

Adds multiple nodes in a single, non-interactive command using a pattern range expansion. This is ideal for script-based automation or AI agents registering fleets of hardware nodes.

Usage:
```bash
bindu-studio mesh bulk-add \
  --pattern "node-pi-{1..10}" \
  --platform "linux/arm64" \
  --parent "root" \
  --connectivity "gateway"
```

Supported expansions in `--pattern`:
*   **Numeric Ranges**: e.g., `{1..100}` or `{01..05}` (retains leading zeros).
*   **Alphabetic Ranges**: e.g., `{a..e}` or `{A..C}`.

Defaults:
*   Nodes are registered as children under the specified `--parent` ID.
*   For `gateway` connectivity, the specified `--platform` is applied, and default ports `1883` (`mesh_mqtt`) and `1885` (`mqtt_integration`) are assigned automatically.
*   For `reference` connectivity, standard fallback defaults are applied.

---

### `mesh rescue-parent` & `mesh rescue-all` (Emergency Recovery)

Break-glass recovery for situations where root and edge nodes change IP addresses (such as migrating from a development lab to a production network) and lose connectivity.

See the complete guide in [Emergency Recovery: mesh rescue-parent & mesh rescue-all](#emergency-recovery-mesh-rescue-parent--mesh-rescue-all).

---

## Mesh Apply + Restart Flow (Current Behavior)

### What `mesh apply` does

- Persists the active draft to the live mesh config.
- Writes:
  `~/.bindu/root/config/mesh.json`
- Clears:
  `~/.bindu/root/config/mesh.draft.json`
- Prompts for restart so the runtime can load the new mesh.

Example output (current UX):

```
🕸️ Mesh applied successfully.
✅ Wrote /Users/<you>/.bindu/root/config/mesh.json.
ℹ️ Cleared draft file /Users/<you>/.bindu/root/config/mesh.draft.json.
⚠️ bindu-root needs restart for changes to take effect.
🔄 Restart now? (Y/n)
```

### Restart prompt + authentication

If you answer "Y", restart may fail because restart endpoints require auth:

```
Restart failed: authentication required.
ℹ️ Run: bindu-studio reset-bindu --auth user:pass
```

This is expected. Use `--auth` to authenticate the restart/reset action:

```
bindu-studio reset-bindu --auth console:supersecret
```

### Authentication for Restart/Reset Commands

Commands that restart bindu-root require credentials:

- `bindu-studio reset-bindu`
- Any command that triggers an immediate restart (e.g. restart prompts)

Use:

```
--auth user:pass
```

Example:

```
bindu-studio reset-bindu --auth console:supersecret
```

### Mesh Warnings and Star Errors

Advisory warning (non-blocking):

```
child gateway node "<id>" is deployable. Use gateway for real runtime nodes; use reference for grouping-only nodes.
```

Meaning:
- **gateway** = real runtime node (deployable).
- **reference** = hierarchy/organization only (not a runtime node).

This warning appears when a non-root child is marked as `gateway`. It is guidance, not a failure.

Hard validation error (blocking):

```
mesh star violation at "<path>": gateway "<gateway_id>" has non-root operational parent "<parent_id>" (expected root "<root_id>")
```

This error means runtime star connectivity would be broken. Studio blocks save/apply in this case.

---

### `mesh edit [id]`

Edits a node via a wizard.
If `id` is omitted, a numbered ASCII tree picker is shown.

Examples:

```
bindu-studio mesh edit
bindu-studio mesh edit root
```

Picker example:

```
Select node to edit:
  1) root (gateway, node_id=root)
  2) |-- rack-refs (reference)
  3) |   `-- dc-rack-room-1 (gateway, node_id=node-rack-room-1)
  4) `-- dc-ups-room (gateway, node_id=node-ups-room-new)
Select number or filter
```

Notes:
- You can type a number or a substring filter (e.g., `rack`).
- The wizard prints a warning if the edited node now violates operational star.
- Validation still blocks saving if the final topology violates operational star.
- Draft changes are written to `config/mesh.draft.json`.
- Use `--all` to include reference nodes in the picker.

---

### `mesh delete [id]`

Deletes a node and re-parents its children into its parent.
If `id` is omitted, a numbered ASCII tree picker is shown.

Examples:

```
bindu-studio mesh delete
bindu-studio mesh delete node-1 --yes
```

Notes:
- Root cannot be deleted.
- Confirmation is required unless `--yes` is provided.
- You can type a number or a substring filter (e.g., `ups`).
- Draft changes are written to `config/mesh.draft.json`.

---

### `mesh gateways`

Compares gateway nodes in a table.

Examples:

```
bindu-studio mesh gateways
```

Example output:

```
ID  NODE_ID  NAME/SLUG     KIND     PARENT
root  root   Initial Root  gateway  n/a
```

Notes:
- `-` means the value is not declared.
- `n/a` means the field does not apply for that node kind.

---

### `mesh channels`

Compares channel ports across gateways.

Examples:

```
bindu-studio mesh channels
```

Example output:

```
GATEWAY_ID  MESH_MQTT  MQTT_INTEGRATION  WS_INTEGRATION  REST_QUERY
root        1883       1885              8443            1887
```

Notes:
- `-` means the channel is not declared.

---

### `mesh platforms`

Compares platform targets across gateways.

Examples:

```
bindu-studio mesh platforms
```

Example output:

```
GATEWAY_ID  OS     ARCH
root        linux  amd64
```

Notes:
- `-` means the value is not declared.

---

### `mesh validate`

Validates the current mesh against the schema.
If a draft exists, validation uses the draft.

Examples:

```
bindu-studio mesh validate
```

---

### `mesh apply`

Validates the draft (if present) and writes `mesh.json`.
The draft file is cleared on success.

Examples:

```
bindu-studio mesh apply
```

---

### `mesh move`

Moves an existing mesh node under a different parent while keeping its subtree intact.

If no `from,to` pair is provided, Studio shows the same numbered ASCII tree style used by
`add`, `edit`, and `delete`, then asks for:

- `from,to`
- separated by a comma
- either numbers or IDs

Examples:

```
bindu-studio mesh move
bindu-studio mesh move 3,7
bindu-studio mesh move camera-1,room-1
bindu-studio mesh move camera-1,room-1 --yes
```

Behavior:

- Root cannot be moved.
- A node cannot be moved under itself.
- A node cannot be moved under one of its own descendants.
- The subtree stays attached to the moved node.
- The same topology/platform/channel validation used by `add` is applied before the draft is saved.

Preview:

- Shows `move`, `from`, and `to`.
- Prints explicit move warnings when the new parent would create operational-star concerns.
- Saves to `mesh.draft.json`; you still run `mesh apply` to commit.

---

## Channels Explained

### Required Channels (Root)

Root is always deployable. These channels are **required** and cannot be disabled:

- `mesh_mqtt`
- `mqtt_integration`
- `ws_integration`
- `rest_query`

### Required Channels (Nodes)

Nodes are minimal by default:

- `mesh_mqtt`
- `mqtt_integration`
- `ws_integration` and `rest_query` are not supported on nodes (MQTT-only node contract).

Rules:

- Channels are **listener config only** (port and optional bind).
- Object present = channel exists.
- Object absent = channel does not exist.
- Partial declarations are invalid.

---

## Topology Invariants (Enforced)

Bindu Studio enforces these mesh rules:

- Exactly one root.
- Each node appears exactly once in the tree (single parent).
- No cycles.
- Globally unique node IDs.
- No duplicate child IDs under a parent.
- For every non-root gateway, the operational parent (ignoring reference ancestors) must be `root`.

If any violation is detected, Studio blocks save/apply and prints a concrete error.

---

## Forbidden Networking Fields in mesh.json

`mesh.json` must never contain **network identity** like:

- hosts, IPs, DNS names
- addresses or endpoints
- URLs (http/https/ws/wss)

Examples of forbidden fields (wherever they appear):

- `address`, `host`, `hostname`, `ip`, `dial`, `endpoint`, `url`, `uri`, `advertise`

Channels are the only exception: `channels.*.bind` is allowed **only** for channel listeners.

If Studio detects forbidden networking fields/values, it fails fast with a path‑specific error.

---

## Mesh is a Live Topology (No Inactive Nodes)

Every node listed in `mesh.json` is treated as **active and addressable**.

There is no “inactive node” state inside the mesh:

- If a node is not active or reachable, remove it from the mesh.
- This keeps routing, monitoring, updates, and plugin operations deterministic.

If a node has children, remove it with subtree semantics or reparent first:

```
bindu-studio mesh remove edge-01 --subtree
Removed 4 node(s): edge-01, rack-01, row-a, row-b
```

---

## Live Mesh Diagnostics & Node Operations

Bindu Studio provides commands to monitor, inspect, and manage running nodes in real time without querying or locking SQLite database files directly on disk.

### `mesh health`

Provides a cluster-wide operational overview across all gateway nodes in the mesh.

```bash
bindu-studio mesh health
bindu-studio mesh health --json
```

Output includes:
- **Node ID & Name**: Identity of each gateway node.
- **Presence Status**: `active` (fresh heartbeat), `stale`, or `missing`.
- **Last Seen**: Timestamp of the latest presence heartbeat.
- **Last Message**: Timestamp of the most recent message received from this node.
- **Total Messages**: Historical message volume processed in the ledger.
- **Engine Version**: Active version and configuration checksum.

---

### `mesh ping [node-id]`

Measures end-to-end network round-trip time (RTT) and verifies mTLS lane readiness across the MQTT mesh data plane.

```bash
bindu-studio mesh ping node-1 --auth console:SuperSecret1
bindu-studio mesh ping dell-debian1 --count 5 --timeout 3s --auth console:SuperSecret1
```

The root node sends a `MeshPingRequest` over MQTT topic `bindu/in` to the edge node, which immediately responds with `MeshPingResponse`. Studio calculates exact latency (RTT in ms) and displays any intermediate hops.

---

### `mesh node-presence [node-id]` (Alias: `mesh presence`)

Inspects the real-time presence heartbeat, network endpoints, and lane state for a specific gateway node. This provides 1:1 CLI parity with the Web Console presence inspector.

```bash
bindu-studio mesh node-presence node-1
bindu-studio mesh presence dell-debian1 --json
```

Example Output:
```text
Node Presence: node-1 (Dell Edge Gateway) [role: node]
   Type:          NodePresence (node -> root)
   Seen At:       2026-11-05T15:00:00Z (2m 15s ago)
   Status:        ✅ Fresh (TTL: 1800s, expires in 27m 45s)
   Software:      v2.4.1 (Build: 7f8a1c9)

[Advertised Network Endpoints]
   • mesh_mqtt: 10.0.5.51:1883 (TLS, priority: 1)
   • mqtt_integration: 10.0.5.51:1885 (plain, priority: 2)
   • Raw Addresses: 10.0.5.51, 192.168.1.105

[Lane State]
   Lane:          Active / publish allowed
   Publish state: Queued messages can trigger immediate delivery attempts.
```

Key Diagnostic Fields:
- **Type & Direction**: Distinguishes between bottom-up `NodePresence` (`node -> root`) and top-down `ParentPresence` (`root -> children`).
- **Seen At & Freshness**: Shows whether the node's TTL window is active or expired.
- **Advertised Network Endpoints**: Lists dynamic IP addresses, listening ports, TLS configuration, and priority levels announced by the node.
- **Lane State**: Indicates whether the publication lane is active or paused waiting for a fresh presence trigger.

---

### `mesh node-status [node-id]`

Retrieves deep telemetry from the node's latest `MessageState` record in the ledger.

```bash
bindu-studio mesh node-status node-1
bindu-studio mesh node-status dell-debian1 --json
```

Displays:
- **System Uptime & Resources**: Kernel version, CPU cores and load, RAM usage (total, free, used %), disk storage utilization.
- **BPR Plugins**: List of active OS-level BPR plugins, process IDs, and health status.
- **Reactors**: Active Common Expression Language (CEL), Semantic AI, and REST reactor configurations loaded on the node.
- **Database Statistics**: SQLite ledger size and message counts.

---

### `mesh node-version [node-id]`

Queries component versions and cryptographic build provenance from `SystemVersion`.

```bash
bindu-studio mesh node-version node-1
bindu-studio mesh node-version dell-debian1 --json
```

Displays:
- **Engine Version**: Semantic version, Git commit SHA, and build date.
- **Platform**: Operating system and CPU architecture (e.g., `linux/amd64`, `linux/arm64`).
- **Artifact Hashes**: SHA256 checksums of core binaries, launchers, and assigned plugins.

---

### `mesh node-refresh [node-id]` (Aliases: `refresh-node`, `refresh-state`)

Commands an edge node to immediately audit its internal telemetry and emit a fresh `MessageState` event to the mesh.

```bash
bindu-studio mesh node-refresh node-1 --auth console:SuperSecret1
bindu-studio mesh node-refresh dell-debian1 --reason "Verifying BPR plugin rollout" --auth console:SuperSecret1
```

How it works:
1. `bindu-studio` calls Root's HTTP REST Management Plane.
2. Root transmits an `EmitMessageStateRequest` over the mTLS MQTT Data Plane.
3. The edge node refreshes its resource metrics and publishes a new `MessageState` message.

---

### `mesh node-restart [node-id]`

Triggers a graceful service restart of a remote edge gateway node.

```bash
bindu-studio mesh node-restart node-1 --auth console:SuperSecret1
bindu-studio mesh node-restart dell-debian1 --reason "Applying kernel updates" --yes --auth console:SuperSecret1
```

Flags:
- `--reason "..."`: Explanatory message logged with the restart event.
- `--force`: Force reset even if background workers are active.
- `--yes`: Bypass terminal confirmation prompt.

How it works:
1. Studio sends an authenticated request to Root's REST Management Plane.
2. Root publishes a `SystemNodeReset` message over MQTT topic `bindu/in` targeting the node.
3. The target node flushes pending telemetry, gracefully terminates its process, and allows its service supervisor (`systemd`) to restart it with clean state.

---

## Emergency Recovery: mesh rescue-parent & mesh rescue-all

### The Dynamic IP Discovery vs Static Topology Architecture

In Bindu, network architecture is governed by two immutable rules:
1. **`mesh.json` contains NO IP addresses or hostnames.**  
   The mesh configuration defines only logical topology (parent/child hierarchy), node platforms, and channel ports (e.g. `1883`, `1885`). Network identities (IPs, DNS hostnames, URLs) are strictly forbidden in `mesh.json`.
2. **Dialing is IP/presence-driven; authentication is node_id-driven.**  
   Nodes dynamically announce their network addresses via presence messages. TLS certificates authenticate `node_id` (e.g. `node-1`), which is decoupled from dynamic transport IP addresses.

Dynamic presence flows in two directions:
- **`NodePresence` (`child -> parent`):** Child gateways periodically announce their dynamic IPs and ports to root.
- **`ParentPresence` (`parent -> children`):** The root coordinator announces its dynamic IP and ports to child gateways.

When a publication attempt exhausts retries, the publication lane enters the `waiting for presence` state. No continuous background reconnect loop runs; the lane pauses until awakened by a fresh presence message.

---

### The Network Migration / Subnet Drift Problem

A common scenario occurs when moving a mesh from a staging lab to a production site:
- In the lab, root had IP `192.168.1.100` and `node-1` had IP `192.168.1.105`.
- In production, root is assigned `10.0.5.10` and `node-1` is assigned `10.0.5.51`.

When the systems boot up in production:
1. `node-1` boots and attempts to dial Root at its *old* lab IP (`192.168.1.100`). The connection fails, and `node-1`'s publication lane enters `waiting for presence`.
2. Root boots and attempts to dial `node-1` at its *old* lab IP (`192.168.1.105`). The connection fails.
3. Neither side can contact the other automatically, because each is waiting for the other to announce its new address.

---

### Why `--all` Cannot Rescue Nodes When Both Root and Nodes Move

The `bindu-studio mesh rescue-parent --all` command instructs root to emit a `ParentPresence` run-now trigger to all child nodes in the mesh.

However, to deliver `ParentPresence` to a child, root's dialer must resolve the child's IP from the local ledger database (`v_messages`).
- If the child nodes also changed IP addresses, root's ledger contains their **stale lab IPs**.
- Root dials the stale IPs, the connection fails, and the rescue attempt cannot reach the children.

> [!IMPORTANT]
> **When `--all` works:** `--all` is effective **ONLY** when edge nodes remain at their known IP addresses and only the Root coordinator changed its IP address.  
> **When `--all` fails:** If both Root and edge nodes moved to new network addresses, you **MUST** provide explicit IP addresses using targeted rescue or batch rescue (`rescue-all`).

---

### Targeted Rescue Mode (`--node-id` + `--node-ip`)

Used to recover a single node whose new IP address is known.

```bash
bindu-studio mesh rescue-parent \
  --node-id dell-debian1 \
  --node-ip 10.0.5.52 \
  --auth console:SuperSecret1
```

**What occurs under the hood:**
1. Studio issues an authenticated HTTP REST POST to Root's `/api/v1/mesh/rescue-parent-presence`.
2. Root temporarily seeds its child presence router cache with the provided IP (`10.0.5.52`) and the node's configured MQTT port.
3. Root dials the node at `10.0.5.52:1883` over mTLS, verifies the node's TLS certificate (`node_id`), and delivers a fresh `ParentPresence` containing Root's new IP.
4. Receipt of `ParentPresence` awakens the child node's publication lane, updates its parent endpoint cache with Root's new IP, and prompts the child to immediately reply with a fresh `NodePresence`, fully re-establishing the bidirectional link.

---

### Batch Rescue Mode (`mesh rescue-all` / `--nodes` / `--inventory`)

When recovering an entire fleet of nodes that moved to a new network, use batch rescue. You can invoke this via `bindu-studio mesh rescue-all` or `bindu-studio mesh rescue-parent`.

#### Option A: Inline Node-to-IP Mappings (`--nodes`)
Pass comma-separated `node_id=ip` pairs:

```bash
bindu-studio mesh rescue-all \
  --nodes "node-1=10.0.5.51,dell-debian1=10.0.5.52,bindu-pi3-uno=10.0.5.53" \
  --yes \
  --auth console:SuperSecret1
```

#### Option B: Inventory File (`--inventory`)
Provide a path to an inventory file containing node IP mappings:

```bash
bindu-studio mesh rescue-all \
  --inventory /etc/bindu/production-nodes.txt \
  --yes \
  --auth console:SuperSecret1
```

Supported inventory file formats:

1. **Key-Value Pairs (equals or colon):**
   ```text
   # Production subnet nodes
   node-1=10.0.5.51
   dell-debian1: 10.0.5.52
   bindu-pi3-uno=10.0.5.53
   ```
2. **Whitespace-Separated Columns:**
   ```text
   node-1        10.0.5.51
   dell-debian1  10.0.5.52
   bindu-pi3-uno 10.0.5.53
   ```
3. **JSON Object:**
   ```json
   {
     "node-1": "10.0.5.51",
     "dell-debian1": "10.0.5.52",
     "bindu-pi3-uno": "10.0.5.53"
   }
   ```

Output summary:
```text
Parent presence rescue: batch (3 nodes)
  ✅ node-1 (10.0.5.51:1883)
  ✅ dell-debian1 (10.0.5.52:1883)
  ✅ bindu-pi3-uno (10.0.5.53:1883)
Parent presence rescue: batch completed (3/3 succeeded)
```

---

### Broadcast Rescue Mode (`--all`)

```bash
bindu-studio mesh rescue-parent --all --yes --auth console:SuperSecret1
```

- Forces Root to immediately emit `ParentPresence` to all registered children using the addresses stored in its presence cache.
- Useful when Root restarted on a new DHCP address or interface, but child nodes did not move.

---

## Troubleshooting

### Validation Failures

Common causes:

- Missing required channels (`mesh_mqtt`, `mqtt_integration`, and on root: `rest_query`, `ws_integration`)
- Duplicate IDs or cycles
- Operational star violation (`gateway` resolves to non-root operational parent)
- Forbidden networking fields in mesh.json
- Invalid `id` or `slug` patterns

How to recover:

1) Run `bindu-studio mesh validate` to see errors.
2) Fix the mesh via `mesh edit` or by editing the file manually.
3) Re-run `mesh validate`.
4) Run `mesh apply` to commit.

For hierarchy changes, prefer:

- `mesh move` when the node should stay in the mesh under a different parent
- `mesh delete` when the node should disappear but its children should stay
- `mesh remove --subtree` when the whole branch should disappear

---

## Appendix: Command Reference

```
# Topology & Configuration Management
bindu-studio mesh tree [--all] [--mesh PATH] [--base-dir DIR]
bindu-studio mesh gateways [--all] [--mesh PATH] [--base-dir DIR]
bindu-studio mesh channels [--mesh PATH] [--base-dir DIR]
bindu-studio mesh platforms [--mesh PATH] [--base-dir DIR]
bindu-studio mesh validate [--mesh PATH] [--base-dir DIR]
bindu-studio mesh add [--mesh PATH] [--base-dir DIR]
bindu-studio mesh bulk-add --pattern PATTERN [--platform OS/ARCH] [--parent ID] [--connectivity KIND]
bindu-studio mesh edit [id] [--all] [--mesh PATH] [--base-dir DIR]
bindu-studio mesh move [from,to] [--yes] [--mesh PATH] [--base-dir DIR]
bindu-studio mesh delete [id] [--yes] [--mesh PATH] [--base-dir DIR]
bindu-studio mesh remove [id] [--subtree] [--yes] [--mesh PATH] [--base-dir DIR]
bindu-studio mesh apply [--mesh PATH] [--base-dir DIR]
bindu-studio mesh export --out DIR [--mesh PATH] [--base-dir DIR]
bindu-studio mesh import --from DIR [--mesh PATH] [--base-dir DIR]
bindu-studio mesh merge --with PATH [--mesh PATH] [--base-dir DIR]
bindu-studio mesh node-deploy --node ID [--out DIR]

# Live Mesh Diagnostics & Node Operations
bindu-studio mesh health [--json] [--mesh PATH] [--base-dir DIR]
bindu-studio mesh ping NODE_ID [--count N] [--timeout DURATION] [--auth USER:PASS] [--json]
bindu-studio mesh node-presence NODE_ID [--json] [--config PATH] [--base-dir DIR]  # Alias: presence
bindu-studio mesh node-status NODE_ID [--json] [--config PATH] [--base-dir DIR]
bindu-studio mesh node-version NODE_ID [--json] [--config PATH] [--base-dir DIR]
bindu-studio mesh node-refresh NODE_ID [--reason TEXT] [--auth USER:PASS] [--json] # Aliases: refresh-node, refresh-state
bindu-studio mesh node-restart NODE_ID [--reason TEXT] [--force] [--yes] [--auth USER:PASS] [--json]

# Emergency Recovery
bindu-studio mesh rescue-parent --node-id ID --node-ip IP [--auth USER:PASS] [--yes]
bindu-studio mesh rescue-all --nodes "ID1=IP1,ID2=IP2" [--auth USER:PASS] [--yes]
bindu-studio mesh rescue-all --inventory PATH [--auth USER:PASS] [--yes]
bindu-studio mesh rescue-parent --all [--yes] [--auth USER:PASS]
```
