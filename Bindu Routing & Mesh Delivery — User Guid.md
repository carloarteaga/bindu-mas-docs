# Bindu Routing & Mesh Delivery — User Guide

Role of this document:

- Canonical public routing guide
- Best first stop for operators and developers debugging routing behavior

For a shorter internal routing summary, see `docs/ROUTING_MODEL.md`.

This document defines the routing semantics and delivery model used by Bindu.
It is written for operators and developers debugging the system.

Bindu separates:
- **Intent** (declared by message types / producers)
from
- **Delivery** (performed by the routing engine and mesh transport)

---

## 1) Core Concepts

### 1.1 node_id
A `node_id` is the stable semantic identifier of a node in the mesh (from mesh config).  
Examples: `root`, `dell-debian1`.

### 1.2 Ingress vs Canonical Topics (CRITICAL)
Bindu uses two kinds of topics:

#### Ingress topic (the only entrypoint)
- **`bindu/in`** is the **only** topic treated as “new message entering the system”.
- Messages arriving on `bindu/in` are eligible for:
  - ACL validation (unless mesh/system bypass applies)
  - routing evaluation
  - persistence evaluation
  - plugin execution

#### Canonical topics (terminal outputs)
Canonical topics have the form:

`MessageX/<group>/<producer>`

They exist for:
- observation / subscriptions
- UI monitoring
- replay / state distribution

**Canonical publishes must never re-enter routing/forwarding and must never re-trigger plugin execution.**
They are terminal outputs by design.

> Operational implication:
> If you ever see canonical topics being treated as ingress, you will get duplicates and/or routing loops.

---

## 2) Message Headers That Drive Routing & Governance

These metadata fields are validated and stamped by the Enforcer during ingestion:

### 2.1 `$node-dest`
`$node-dest` is the **final semantic destination node**.
It is NOT:
- a hop-by-hop path
- a transport endpoint
- something users compute manually

**Rule:** Producers declare *destination*. Routing engine computes *paths*.

### 2.2 `$node-origin`
`$node-origin` identifies where the message first entered the mesh. 
- **Enforcer Stamped**: The Enforcer stamps this header with the node ID of the host receiving the event. 
- **Bypass prevention**: If a plugin attempts to pre-populate or forge this header, the host overrides it with the true local node ID.

### 2.3 `$node-exec`
`$node-exec` represents the node where execution/processing occurred.
- **Enforcer Stamped**: Stamped automatically on ingestion with the ID of the local node. 
- **Bypass prevention**: If a plugin sets this, the host runtime ignores it, logs a warning (`msg.node_exec.ignored`), and stamps the true local node ID.

### 2.4 `$producer`
`$producer` identifies the component or plugin that created the message.
- **BPR Plugins**: set to the plugin name (e.g. `hello-world-plugin`).
- **AI Semantic Reactors**: set to the specific `ReactorID` (e.g. `backyard-flower-counter`) to allow granular origin tracking.
- **System Messages**: stamped automatically with the `LocalNodeID` by the Enforcer if empty.

### 2.5 `$src`
`$src` represents the node initiating a network control or connectivity diagnostic request (e.g., in `MessagePing`) or replying with a response (e.g., in `MessagePong`). It allows hop-by-hop path tracing and response routing.

---

## 3) Declarative Routing Intent (Message Type)

Routing intent answers:

> Should this message be delivered locally, forwarded to parent, forwarded to children?

Current model uses directional intent such as:
- `local`
- `parent`
- `child`
- (and combinations depending on your schema/model)

**Routing intent is policy.**
The routing engine does not invent intent; it only executes it.

---

## 4) Mesh Topology Model

Bindu nodes form a **hierarchical mesh**:
- each node has at most one parent
- each node has zero or more children

Example:

root
 └─ dell-debian1

Topology and endpoints are resolved structurally from mesh/config in the active runtime path.

Presence is not an endpoint authority. It only re-opens a publish attempt after a failed cycle has stopped waiting for a fresh trigger.

---

## 5) Delivery Model (Routing Engine)

### 5.1 Responsibility boundary
The Routing Engine is the only component responsible for:
- selecting physical endpoints (IP:port, TLS endpoint, etc.)
- sending messages to other nodes
- retries/backoff and failure isolation
- per-endpoint ordering

It does NOT:
- execute business logic
- decide routing policy
- interpret canonical topics as ingress

### 5.2 Endpoint resolution (physical)
When a message is approved to route, the engine resolves the destination to a concrete endpoint using the current mesh view (presence/session).

Resolution happens at **send time**, so the system adapts to:
- reconnects
- topology changes
- node restarts

Practical order is: active session first, fresh presence next, then stable bootstrap/config hints when presence is unavailable or expired.

---

## 6) Routing Queue & Ordering

Messages approved for routing enter a routing queue/spool.
Bindu enforces:
- per-endpoint ordering (one writer per endpoint)
- isolation (a failing endpoint does not block others)

Retries should be exponential backoff and state-transition logged (not per-attempt noise).

---

## 7) Root semantics

Root has no parent.

Therefore:
- If routing includes `parent` and the current node is `root`,
  “parent” direction terminates locally (it is not an error).

This is how system state naturally bubbles up leaf → root.

---

## 8) Plugin Interaction Rules (VERY IMPORTANT)

Plugins must not duplicate routing responsibilities.

### 8.1 Plugins may produce new messages
Example: MeshPingPlugin may emit a `MessagePong` when a `MessagePing` is destined to the local node.

### 8.2 Plugins must NOT forward messages that routing will forward
If a `MessagePing` arrives with `$node-dest != local_node_id`, routing is already responsible for forwarding it.

A plugin forwarding the same ping produces duplicates:
- same `$id` routed twice
- two deliveries
- two responses (e.g., double pong)

**Rule:** Plugin should only “reply” when destination is local, not forward.

---

## 9) Debugging Checklist

### 9.1 Confirm ingress vs canonical boundary
- New messages should enter via `bindu/in`
- Canonical publishes should appear only as outputs and never be reprocessed

### 9.2 Confirm routing decision logs
Look for:
- routing.effective (direction, forward decisions)
- queue enqueue reason/cause
- endpoint selected (presence/session/config source)

### 9.3 Confirm “one message id = one forward”
For a given `$id`:
- you should see at most one enqueue per target endpoint per node

If you see duplicates with the same `$id`, check:
- plugin forwarding
- canonical topic being treated as ingress
- re-entry from local publishes

---

## 10) Summary

- Producers declare semantic intent (`$node-dest`, routing intent)
- `bindu/in` is the only ingress topic
- Canonical topics are terminal outputs
- Routing engine performs delivery, ordering, retries, endpoint selection
- Plugins can generate new messages but must not forward messages already handled by routi
