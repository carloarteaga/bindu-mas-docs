# Bindu — Mesh User Manual (Updated)

Role of this document:

- Conceptual/public mesh semantics guide
- Best companion to the Studio mesh manual, not the primary CLI workflow guide

For day-to-day mesh operations, start with `Bindu Studio Mesh User Manual.md`.

Operational editing commands now include:

- `mesh add`
- `mesh edit`
- `mesh move`
- `mesh delete`
- `mesh remove`
- `mesh apply`

## 1. What the Mesh Is

The **Mesh** is the authoritative model for both:

- **operational connectivity** (runtime star: `root` <-> gateways),
- **semantic hierarchy** (reference tree for organization/location).

It defines:
- where runtime nodes exist,
- where plugins are deployed,
- how routing intent is derived from topology + identity.

The Mesh is:

- ❌ not a visual diagram
- ❌ not a simulation
- ❌ not edited through the Console UI

It is the **single source of truth** for connectivity, routing, and execution.

---

## 2. Mesh Responsibility Split (Very Important)

Bindu deliberately separates **definition** from **observation**.

### Studio vs Console

| Tool | Responsibility |
|----|----|
| **Bindu Studio (CLI)** | Define, validate, and apply the mesh |
| **Bindu Console (UI)** | Observe the running mesh (read-only) |

> **Studio defines. Console observes.**

This separation prevents:
- partial edits,
- UI inference bugs,
- configuration drift,
- unsafe runtime changes.

---

## 3. Mesh Lifecycle

Design / Edit → Bindu Studio
Validate → bindu-studio mesh validate
Apply (commit) → bindu-studio mesh apply
Run → bindu-root
Observe → Bindu Console


The mesh is stored as a **single file**:

config/mesh.json


There are **no partial updates**.  
The entire document is validated and applied atomically.

---

## 4. Mesh Topology Model

### Semantic Tree

The mesh is a strict parent/child tree:
- One root node
- No cycles
- Deterministic hierarchy for grouping and location semantics

### Operational Star

Connectivity is constrained by an operational star rule:
- There is exactly one operational parent for all non-root gateways: `root`.
- Reference nodes are skipped when resolving operational parent.
- A gateway resolving to a non-root operational parent is invalid.

---

## 5. Root Node (Gateway)

Every mesh has **exactly one root node**.

### Root characteristics

- `connectivity_kind: gateway`
- Always exists
- Cannot be deleted
- Anchors routing and identity
- Defines required mesh channels

If the root is invalid or missing required channels, **Bindu will not start**.

---

## 6. Node Types (`connectivity_kind`)

Supported values are:
- `gateway`
- `reference`

### 6.1 Gateway

- Used for root and runtime/deployable nodes
- Can have children
- Can run plugins
- Must include `bindu` configuration
- Deployable by `bindu-studio mesh node-deploy` (non-root gateways only)

---

### 6.2 Reference

- Organizational / logical node only
- No real connectivity
- Does not run plugins
- Must include `bindu.node_id` and required channels in mesh config
- Not deployable (use for grouping only)

Used to:
- group nodes,
- model locations or ownership,
- structure large meshes.

### 6.3 Operational Parent Rule (Enforced)

For each non-root gateway:
- Traverse ancestors upward.
- Ignore any `reference` nodes.
- First remaining gateway must be `root`.

If not, Studio rejects the mesh with a `mesh star violation` error.

---

## 7. Functional Networking Channels

Bindu networking is configured **by function**, not by low-level transport options.

### Channels

| Channel | Purpose | Required |
|------|--------|---------|
| **mesh_mqtt** | Internal backbone (root ↔ nodes) | Yes (root and nodes) |
| **mqtt_integration** | External publish/subscribe | Yes (all nodes in mesh config) |
| **ws_integration** | Realtime streaming | Root only |
| **rest_query** | Read-only queries | Root only |

### Rules

- **Declared = exists**
- **Not declared = does not exist**
- No “blank means off”
- No partial channel objects
- TLS / mTLS are enforced automatically
- `bindu.node_id` + channel set are part of the mesh contract for all nodes; references still remain non-deployable

If a required channel is missing, **Bindu fails to start**.

---

## 8. What You Configure (and What You Don’t)

### You configure only:

- Bind address
- Port

### You do NOT configure:

- TLS on/off
- Certificates
- Authentication modes
- Console exposure

These are **system invariants**.

Identity and addressing are separate concerns:
- `bindu.node_id` is stable identity.
- `tls_server_name` is selected at runtime for TLS verification.
- `dial_address` is selected at runtime as host:port.

---

## 9. Plugins and Deployment

The `plugins` list on a node declares **intent**:

- which plugins should run on that node
- where execution happens

Rules:

- Reference nodes cannot run plugins
- Plugins are validated against the registry
- The mesh does not execute plugins — it declares placement

---

## 10. Validation Philosophy

Bindu uses **strict validation**:

- Full-document validation
- Schema + contract rules
- Atomic failure on invalid state

This guarantees:

- no partial configuration,
- no undefined routing,
- predictable startup.

---

## 11. Mesh in the Console

The **Mesh Config** section in the Console:
- Shows the current mesh hierarchy.
- Displays resolved channels.
- Exposes runtime status (ping, presence, and health).

It is **read-only**. If something looks wrong, do not edit in the Console; use **Bindu Studio** to fix and apply changes.

### 11.1. Interactive Diagnostics & Node Actions
Within the Mesh Configuration tree, each active gateway node provides dynamic action buttons:
* **📊 Status**: Opens the **Mesh Status Dashboard** modal. This modal provides deep, real-time diagnostic visibility organized into specialized tabs:
  * **🖥️ General Status**: Live circular progress gauges for hardware metrics (`CPU USAGE`, `MEMORY USAGE`, `DISK USAGE`), Linux Load Averages (1m, 5m, 15m), and Node Properties (Node ID, Role, Uptime, Release Version, Last State Timestamp).
  * **🔌 BPR & System Plugins**: Lists active system daemons (`node_presence`, `node_state`) and supervised BPR binary plugins running on the node with their execution status and OS Process IDs (PIDs).
  * **🤖 AI Reactors**: Displays active cognitive AI reactors (`AISemanticReactorSet`) with assigned model names, status badges (`active`, `errored`), PIDs, and active worker pool counts.
  * **🌐 REST Reactors**: Dedicated operational tab for declarative REST Call Reactors (`RESTReactorSet`) deployed to the node:
    * **Reactor ID**: Unique identifier of the REST integration worker.
    * **Input / Trigger**: Reactive ingress message type or proactive cron schedule (`⏰ Cron Trigger`).
    * **Output Type**: Target structured `MessageType` schema validated and published back to the mesh.
    * **Status**: Live worker health indicator (`active` in green, `errored` in red, `stopped` in gray).
    * **PID**: Host operating system Process ID for zero-SSH supervisory process tracking.
  * **📂 Spool & Messages**: Breakdown table showing active message counts grouped by `MessageType` buffered in local storage, plus total buffered message volume.
  * **📄 Raw JSON**: Direct inspector for the underlying `MessageState` telemetry payload emitted by the node for low-level triage.
* **🧾 Version**: Displays the live runtime version, build SHA, and installer metadata.
* **Presence**: Inspects the presence state lane and communication endpoints.
* **📡 Ping**: Pings the node directly via MQTT and displays real-time connection status.
* **📋 Logs**: Opens the **On-Demand Log Viewer** modal:
  * **Zero-SSH Diagnostics**: Inspect, search, and copy logs from remote edge gateways directly from the UI without SSH.
  * **Flexible Filtering**:
    * **Log Levels**: Filter messages dynamically by severity levels: `ALL`, `DEBUG`, `INFO & ABOVE` (default), `WARN & ABOVE`, or `ERROR ONLY`.
    * **Search Queries**: Filter log lines containing specific words or patterns.
  * **Filtering Examples**:
    * *Standard File Logs (Fallback)*: Uses case-insensitive substring search (e.g. searching for `bpr.plugin` will match any lines containing `bpr.plugin`, `BPR.PLUGIN`, or `bpr.plugin.error`).
    * *Systemd/Journal Logs (Production)*: Uses PCRE (Perl Compatible Regular Expressions) search when backed by systemd (e.g. searching for `bpr.*failed` captures all plugin failure sequences, or `^(ERROR|WARN)` matches specific starting tags).
  * **Linux Documentation & Scope**:
    To understand the full capability and matching rules of the active filters, refer to the following standard Linux manual pages:
    * **`journalctl(1)` (`man journalctl`)**: Explains the `--grep` (regular expression pattern matching) and `-p` / `--priority` (syslog severity level filtering) flags. (See [journalctl online manual](https://www.freedesktop.org/software/systemd/man/journalctl.html)).
    * **`syslog(3)` (`man syslog`)**: Describes syslog severity priority levels (0 to 7) used by systemd. Bindu maps `DEBUG` to priority 7, `INFO` to 6, `WARN` to 4, and `ERROR` to 3.
    * **`proc(5)` (`man proc`)**: Describes the `/proc` filesystem. Bindu reads `/proc/self/fd/1` to dynamically resolve stdout target descriptors.
  * **Auto-Refresh**: Toggle automatic log polling every 5 seconds.
  * **Copy to Clipboard**: Quick copy of the active terminal session logs.
  * **Zero RAM Overhead**: Integrates directly with Linux `/proc/self/fd/1` file redirects and systemd journalctl, using 0 bytes of background memory and preserving full historical log evidence.

### 11.2. Inspect Mesh Topology Visualizer
To view a live, interactive topological map of the mesh, click the **🕸️ Inspect Mesh Topology** button inside the Mesh Config page. This visualizer provides a complete event-driven view of nodes, agents, and data flows.

#### A. Interactive Canvas & Navigation
* **Zoom & Pan:** Grab and drag the canvas to pan across the mesh. Use your mouse scroll wheel or the zoom buttons (`+`, `-`, `↺ Reset`) in the floating toolbar to adjust the camera.
* **Interactive Minimap:** A live minimap is pinned to the bottom-right corner. Dragging the highlighted viewport inside the minimap moves the camera across the main canvas.
* **Canvas Focus Selection:** Clicking directly on any node's subgraph box in the diagram focuses the visualizer on that specific gateway (and the root coordinator).

#### B. Search & Toolbar Filters
* **🔍 Focus:** A searchable dropdown combobox to zoom into a specific gateway node or display the full mesh (`💠 All Nodes`).
* **🛠️ Filter:** A combobox to filter the displayed agent types in the diagram:
  * **🔀 CEL Reactors:** Show only processors using Common Expression Language.
  * **🧠 Semantic AI Reactors:** Show only AI-assisted LLM processors.
  * **🌐 REST Reactors:** Show only declarative HTTP/REST integration bridges.
  * **🔌 BPR Plugins:** Show OS-level plugins.
  * **🔘 All Agents:** Display everything.
* **✉️ Message Type:** A searchable multiselect combobox. Typing a term filters the checklist. You can select one or more message types (or `All Message Types`):
  * The diagram dynamically hides any nodes (including the coordinator root) that do not contain at least one reactor or plugin consuming or producing the selected message types.
  * Connection links and process flows are filtered to show only lines connected to matching nodes.
* **⬅️ Back to Full Mesh:** A unified reset button that appears automatically when any filter or focus is active. Clicking it clears all selections, returning Focus, Filter, and Message Types back to `All`.

#### C. Color-Coded Legend & Shapes
* **Blue/Navy Square:** Gateway nodes (including `root`).
* **Gray Rounded:** logical Reference nodes.
* **Purple Dashed Box:** BPR Plugins (`🔌`), CEL Reactors (`🔀`), REST Reactors (`🌐`), and Semantic AI Reactors (`🧠`).
* **Green Capsule:** Local Models (inference running locally).
* **Yellow Capsule:** Cloud Models (inference running in the cloud).
* **White Box:** Message Flow containers:
  * `📥 Triggers` / `📥 Consumes`: Incoming messages or **⏰ Cron** scheduler strings (yellow badges representing periodic execution triggers, e.g., `@every 1m`).
  * `📤 Emits` / `📤 Produces`: Outgoing messages published by the agent.

---

## 12. Summary

The Mesh:

- defines *what exists*,
- defines *where things run*,
- defines *how messages move*.

Bindu Studio is where the mesh is **designed and applied**.  
Bindu Console is where the mesh is **observed in action**.

This clarity is intentional — and foundational to Bindu.
