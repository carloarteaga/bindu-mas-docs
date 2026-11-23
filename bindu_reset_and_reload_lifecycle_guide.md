# Bindu Reset, Reload & Configuration Lifecycle Guide

## Overview

Bindu operates across distributed edge nodes and central root gateways. To maximize uptime in mission-critical Multi-Agent Systems (MAS), Bindu strictly distinguishes between:
1. **Boot-Time Static Configurations** (require root process restart)
2. **In-Memory Semantic Contracts & Schemas** (zero-downtime in-process hot reload)
3. **Dynamic Declarative Reactors** (zero-downtime event-driven state swap)
4. **Distributed Edge Node Fleet Rollouts** (orchestrated delivery with autonomous node service restart)

This document serves as the authoritative operational guide for when a restart is necessary, how hot reloading works, and how to use `bindu-studio config-state` and `bindu-studio reset-bindu`.

---

## 1. Lifecycle & Reset Taxonomy Matrix

| Component | Storage / Transport | Hot-Reload Mechanism | Process Restart Needed? | How Changes Propagate to Nodes |
| :--- | :--- | :--- | :--- | :--- |
| **System Config** (`binduConfig.json`) | `runtime/config/binduConfig.json` | None (static boot parameters: ports, TLS, listeners) | **YES (Root Restart)** | N/A (Root-only config) |
| **Mesh Topology** (`mesh.json`) | `runtime/config/mesh.json` | None (dialing topology, certificates, parent-child routes) | **YES (Root Restart)** | Packaged via `runtime desired commit -> update wizard` |
| **Access Control** (`accessControlConfig.brs`) | `runtime/config/accessControlConfig.brs` | None (encrypted credentials, RBAC token stores) | **YES (Root Restart)** | Packaged via `runtime desired commit -> update wizard` |
| **Message Types** (`messageTypesConfiguration.json`) | `runtime/config/messageTypesConfiguration.json` | **Dynamic In-Memory Registry** (`messagetypes.RefreshGlobalRegistry`) via `POST /api/v1/message-types/reload` | **NO (Zero Downtime)** | Packaged via `runtime desired commit -> update wizard` |
| **CEL Reactors** (`CELReactorSet`) | SQLite `v_messages` & MQTT `bindu/in` | **In-Memory Lock-Swap** in execution engine | **NO (Zero Downtime)** | Delivered live via MQTT to target nodes |
| **REST Reactors** (`RESTReactorSet`) | SQLite `v_messages` & MQTT `bindu/in` | **In-Memory Lock-Swap** in execution engine | **NO (Zero Downtime)** | Delivered live via MQTT to target nodes |
| **Semantic AI Reactors** (`AISemanticReactorSet`) | SQLite `v_messages` & MQTT `bindu/in` | **In-Memory Lock-Swap** in execution engine | **NO (Zero Downtime)** | Delivered live via MQTT to target nodes |
| **Node Runtime Updates** (`NodeRuntimeUpdate`) | Bundle archive via HTTP/MQTT | Extracted to `runtime/config` and staged | **YES (Node Auto-Restart via systemd)** | Delivered autonomously during rollout wizard |

---

## 2. In-Depth Component Behavior

### 2.1 Static Boot-Time Configurations (Restart Required)
Static configurations define the low-level process wiring, network listeners, TLS certificates, and cryptographic roots of trust. These are read once during process initialization:
- **`binduConfig.json`**: HTTP/WS listener addresses, ports, timeouts, storage engine paths.
- **`mesh.json`**: Physical topology, parent endpoints, peer IP addresses, mTLS credentials.
- **`accessControlConfig.brs`**: Encrypted user credentials, roles, and administrative API tokens.

When any of these files change:
1. The filesystem watcher bumps `runtime/config/VERSION` on disk.
2. The running `bindu-root` process continues running with its initial boot configuration.
3. `bindu-studio config-state` detects that disk `VERSION` exceeds the loaded boot version and displays `Action: restart root`.
4. Running `bindu-studio reset-bindu` performs a graceful service restart.

### 2.2 Message Types (Hot-Reloadable, Zero Downtime)
Message Types represent semantic contracts and JSON schemas governing event payloads.
- **Console UI Editing**: Saving or deleting a Message Type in the Bindu Web Console updates `messageTypesConfiguration.json` and immediately refreshes the active memory registry (`ReloadGlobalRegistry()`).
- **Studio CLI Operations**: Running `bindu-studio message-types import`, `update-type`, or `reload` updates disk files and automatically calls the `POST /api/v1/message-types/reload` endpoint on the running root gateway.
- **Result**: No root downtime. New schemas are active immediately for validation and routing.

### 2.3 Declarative Reactors (Zero Downtime)
CEL, REST, and Semantic AI Reactors are event-driven computation policies deployed across the mesh.
- Reactor sets travel as typed messages (`CELReactorSet`, `RESTReactorSet`, `AISemanticReactorSet`) over MQTT.
- When an updated reactor set arrives, the reactor runtime locks its execution context, replaces the rule set in memory, and releases the lock.
- **Neither root nor edge node processes ever need to be restarted when creating, updating, or deleting reactors.**

### 2.4 Distributed Edge Node Updates (Autonomous Node Restart)
When updated configurations, mesh files, or plugins must be distributed to edge gateways:
1. The operator commits the desired state (`bindu-studio runtime desired commit`).
2. The operator runs the rollout wizard (`bindu-studio runtime update wizard --yes`).
3. Root packages a signed runtime update bundle and delivers `NodeRuntimeUpdate` messages to the target nodes over MQTT.
4. Each node agent downloads the bundle, unpacks files to `runtime/config`, and triggers an autonomous systemd service restart (`systemctl restart bindu-node`).
5. After reboot, the node reports its new `SystemVersion` and `MessageState` back to root.

---

## 3. The `VERSION` File & `config-state` Diagnostics

### How Version Tracking Works
- The `runtime/config/VERSION` file is an integer counter tracking configuration modifications.
- At startup, `bindu-root` and `bindu-node` read `runtime/config/VERSION` and record their **Active Boot Version**.
- The `bindu-sysmon-plugin` reports this active boot version inside periodic `SystemVersion` messages (`fields.config_version`).
- Whenever a file in `runtime/config/` is written or modified, an inotify watcher increments the disk `VERSION`.

### Interpreting `bindu-studio config-state`
Running `bindu-studio config-state` queries the live runtime and contrasts disk state against active boot state:

```
⚙️ CONFIG STATE

bindu-root: running

Component       Root                              Nodes                             Action
---------------------------------------------------------------------------------------------
system-config   in sync (v14)                     all nodes match current config    none
mesh            baseline accepted and unchanged   all nodes on current baseline     none
plugins         assigned matches deployed         all nodes aligned                 none
access-control  in sync                           all nodes match current config    none
```

#### State: Root Loaded Older Config Version
When a static file (e.g., `binduConfig.json` or `mesh.json`) is modified on disk:
```
Component       Root                                     Nodes                   Action
---------------------------------------------------------------------------------------------
system-config   loaded older version (v14 vs disk v15)   1/1 nodes need config   restart root
mesh            baseline accepted and unchanged          all nodes on current    none
plugins         assigned matches deployed                all nodes aligned       none
access-control  root loaded older config version         all nodes match         restart root

Action:
  ℹ️ Run: bindu-studio reset-bindu
```

#### State: Root In Sync, Nodes Need Rollout
When root has been restarted or message types were hot-reloaded, but nodes have not yet received the new version:
```
Component       Root                              Nodes                                Action
---------------------------------------------------------------------------------------------
system-config   in sync (v15)                     1/1 nodes need config rollout        runtime deploy config
mesh            baseline accepted and unchanged   all nodes on current baseline        none
plugins         assigned matches deployed         all nodes aligned                    none
access-control  in sync                           1/1 nodes still on older config      runtime deploy config
```

---

## 4. Operational Workflows & Commands

### Workflow A: Modifying Message Types (Zero Downtime)
```bash
# 1. Import new message types from JSON
bindu-studio message-types import --in-path ./my-message-types.json --all

# 2. Or manually trigger a reload on running root
bindu-studio message-types reload --auth console:StrongPass123

# 3. Check config state (Root remains in sync)
bindu-studio config-state

# 4. Push updated schemas to edge nodes
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit
bindu-studio runtime update wizard --yes --auth console:StrongPass123
```

### Workflow B: Modifying System Configuration or Mesh (Restart Required)
```bash
# 1. Modify system config or mesh
bindu-studio system-config set
# or edit mesh.json / bindu-studio mesh apply

# 2. Verify drift detection
bindu-studio config-state

# 3. Restart bindu-root gracefully
bindu-studio reset-bindu --auth console:StrongPass123 --yes

# 4. Roll out changes to edge nodes if mesh/config changed
bindu-studio runtime baseline accept --yes
bindu-studio runtime desired commit
bindu-studio runtime update wizard --yes --auth console:StrongPass123
```

### Workflow C: Deploying or Updating Reactors (Zero Downtime)
```bash
# 1. Validate rules locally
bindu-studio cel-reactor validate --file ./cel_rules.json
bindu-studio semantic-ai-reactor validate --file ./ai_rules.json
bindu-studio rest-reactor validate --file ./rest_rules.json

# 2. Deploy directly to running mesh (no restart needed)
bindu-studio cel-reactor import --file ./cel_rules.json --auth console:StrongPass123
bindu-studio semantic-ai-reactor import --file ./ai_rules.json --auth console:StrongPass123
bindu-studio rest-reactor import --file ./rest_rules.json --auth console:StrongPass123
```

---

## Summary Checklist for Operators

- **Message Types**: Hot-reloaded in memory. No restart needed.
- **CEL / REST / Semantic AI Reactors**: Ingested via MQTT. No restart needed.
- **`binduConfig.json` / `mesh.json` / `accessControlConfig.brs`**: Require `bindu-studio reset-bindu`.
- **Edge Node Updates**: Automated through `runtime update wizard`. Nodes restart themselves via systemd.
- **Automated Nightly Maintenance Reset (`maintenance_reset: true`)**: During scheduled maintenance (`maintenance_cron`), service-managed root and node processes perform an automatic clean restart via systemd to purge CGO/glibc memory pools and OS buffers.
- **Diagnosis**: Use `bindu-studio config-state` to know exactly what action is required.
