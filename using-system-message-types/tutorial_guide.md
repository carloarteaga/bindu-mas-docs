# 🛡️ Auditing Mesh Routing & Retention with System Telemetry (`MessageState`)

## 📋 Executive Summary & Problem Context

In enterprise production deployments, Bindu coordinates dozens of autonomous Multi-Agent Systems (e.g. 15 simultaneous business solutions spanning Cold-Chain logistics, RFID dock portals, computer vision inference, BLE asset tracking, and Modbus industrial telemetry) operating across distributed edge gateway nodes (`root`, `node-1`, `bindu-minis`, `bindu-pi3-uno`).

Every node in the cluster periodically emits a built-in infrastructure heartbeat: **`MessageState`**.
In Bindu's schema registry (`messagetypes.go`), `MessageState` is specified with:
```go
Name:           "MessageState",
RetentionMode:  "count",
RetentionValue: 1,
Routing:        RoutingSpec{"parent", "local"}
```

Because of bottom-up parent routing (`["parent", "local"]`), every `MessageState` heartbeat emitted by any edge node naturally converges at the **`root` coordinator**, carrying rich system and storage metrics:
```json
{
  "storage": {
    "message_total": 450,
    "message_counts_by_type": [
      { "type": "FreezerTelemetry", "total": 280 },
      { "type": "MessageState", "total": 1 },
      { "type": "SystemVersion", "total": 1 },
      { "type": "UnassignedSensorEvent", "total": 168 }
    ]
  },
  "bpr_plugins": [
    { "id": "coldchain-monitor-plugin", "status": "running" },
    { "id": "rfid-portal-plugin", "status": "error", "last_error": "EOF" }
  ]
}
```

### The 3 Critical Fleet Vulnerabilities
Without automated audit kinetics, large-scale multi-solution deployments risk silent operational decay:
1. **Retention Bloat on Constrained Flash Memory:** High-frequency telemetry streams (e.g. 10 Hz sensor reads) lacking sliding-window count limits (`retentionMode: "count", retentionValue: 50..100`) or mistakenly configured with calendar days (`retentionMode: "days"`), rapidly exhausting SD card write cycles and SQLite disk quotas on edge hardware like Raspberry Pis.
2. **Directionality Leaks & Ghost Telemetry:** Telemetry mistakenly assigned `"child"` routing, causing high-frequency sensor readings to flood downstream edge nodes that do not run those solutions, generating cascade rejection loops (`UNKNOWN_TYPE` $\rightarrow$ `ErrorEvent`).
3. **Stale / Crashed BPR Plugins:** Hardware or external system plugins crashing into `error` state while continuing to occupy scheduled resources.

---

## 🏛️ Architecture: In-Process CEL Kinetics on Root

Rather than writing external background daemons or running heavy AI models on resource-constrained edge devices, Bindu solves this at the coordinator level through **declarative Google CEL Kinetics**:

```mermaid
flowchart TD
    subgraph EdgeFleet ["Distributed Mesh Gateways (Edge Fleet)"]
        N1["bindu-pi3-uno<br/>(Raspberry Pi)"] -->|["parent", "local"]| MS1["MessageState<br/>(message_counts_by_type)"]
        N2["node-1<br/>(Local Edge)"] -->|["parent", "local"]| MS2["MessageState<br/>(message_counts_by_type)"]
        N3["bindu-minis<br/>(Field Node)"] -->|["parent", "local"]| MS3["MessageState<br/>(message_counts_by_type)"]
    end

    subgraph CentralRoot ["Central Coordinator (bindu-root)"]
        MS1 & MS2 & MS3 --> Ingress["Ingress Queue<br/>(bindu/in)"]
        
        Ingress --> CEL["CEL Reactor Agent<br/>(mesh-message-state-retention-auditor)"]
        
        subgraph CELKinetics ["In-Process CEL Evaluation (< 1μs)"]
            K1["1. Parse message_counts_by_type"]
            K2["2. Cross-reference retention thresholds (total > 200)"]
            K3["3. Inspect BPR plugin status (status == 'error')"]
            K4["4. Compute health_score & diagnostic summary"]
            K1 --> K2 --> K3 --> K4
        end
        
        CEL --> CELKinetics
        
        CELKinetics --> Emit["Emit Augmented Audit<br/>MessageStateAudit<br/>($producer = origin_node, $group = 'mesh')"]
        
        Emit --> Ledger[("SQLite Ledger<br/>v_messages")]
    end

    subgraph WebConsole ["Bindu Web Console (Observability Plane)"]
        Ledger --> D1["📊 mesh-storage-auditor<br/>(Fleet-Wide Storage Table)"]
        Ledger --> D2["🚨 mesh-routing-leak-inspector<br/>(Anomaly Triage Feed)"]
    end
```

### Key Architectural Invariants
1. **Root-Only Execution:** The reactor is deployed **only on root** (`gateway_node_id: "root"`). Edge nodes remain 100% untouched and unburdened.
2. **Preservation of `$producer` and `$group`:** The incoming `MessageState` carries `$producer: "<node_id>"` and `$group: "mesh"`. The CEL reactor inherits these envelope fields, so the resulting `MessageStateAudit` is partitioned strictly under `(MessageStateAudit, mesh, <node_id>)`.
3. **Bounded Retention Window:** `MessageStateAudit` declares `retentionMode: "count", retentionValue: 30`. Root maintains a sliding window of the last 30 audit events per node, guaranteeing zero storage bloat on root.

---

## 📦 Solution Artifact Inventory

Following [`public/console/docs/AGENTS.md`](../AGENTS.md), this MAS solution is packaged as a complete, certified declarative suite:

```text
public/console/docs/using-system-message-types/
├── tutorial-plan.md                      # Canonical architecture & engineering design
├── tutorial_guide.md                     # This didactic operational guide
├── poc_message_types.json                # Schema contract: MessageStateAudit@1.0.0
├── types_gen.go                          # Strongly-typed Go struct bindings
├── examples/out/MessageStateAudit.json   # Compliant schema test fixture
├── poc_cel_reactor_config.json           # Declarative CEL ReactorSet for Root
├── poc_dashboards.json                   # 2 Web Console dashboards (Fleet Auditor & Leak Inspector)
├── solution.json                         # MASSolutionSet semantic manifest
├── solution.bundle.json                  # Compiled, standalone deployable MAS bundle
└── deploy_mas_solution.sh                # 5-Phase automated deployment script with Preflight Gate
```

---

## 🔬 Deep-Dive: Declarative Artifacts

### 1. Augmented Schema: `MessageStateAudit` ([`poc_message_types.json`](./poc_message_types.json))
```json
{
  "name": "MessageStateAudit",
  "version": "1.0.0",
  "emoji": "🛡️",
  "category": "system",
  "description": "Augmented operational audit of node message storage, retention health, and routing anomalies.",
  "routing": ["local", "parent"],
  "persist": true,
  "retentionMode": "count",
  "retentionValue": 30,
  "schema": {
    "type": "object",
    "required": [
      "node_id",
      "node_role",
      "total_stored_messages",
      "health_score",
      "irregularity_count",
      "status_summary",
      "top_accumulating_types",
      "irregularities_detected",
      "audited_at"
    ],
    "properties": {
      "node_id": { "type": "string" },
      "node_role": { "type": "string" },
      "total_stored_messages": { "type": "integer" },
      "health_score": { "type": "integer", "minimum": 0, "maximum": 100 },
      "irregularity_count": { "type": "integer" },
      "status_summary": { "type": "string" },
      "top_accumulating_types": { "type": "array" },
      "irregularities_detected": { "type": "array" },
      "audited_at": { "type": "string", "format": "date-time" }
    }
  }
}
```

### 2. CEL Reactor Configuration ([`poc_cel_reactor_config.json`](./poc_cel_reactor_config.json))
The reactor triggers reactively whenever a `MessageState` arrives containing `storage.message_counts_by_type`:
- **Trigger condition:** `"storage" in payload && "message_counts_by_type" in payload.storage`
- **Dynamic Comprehensions:** Uses `dyn(payload.storage.message_counts_by_type)` to safely iterate untyped JSON arrays in CEL:
```json
"health_score": "int(ai.clamp(double(100 - (\"storage\" in payload && \"message_counts_by_type\" in payload.storage && dyn(payload.storage.message_counts_by_type).exists(m, m.total > 200) ? 30 : 0) - (\"bpr_plugins\" in payload && dyn(payload.bpr_plugins).exists(p, p.status == 'error') ? 25 : 0) - (\"system_resources\" in payload && \"disk_pct\" in payload.system_resources && double(payload.system_resources.disk_pct) > 85.0 ? 30 : 0)), 0.0, 100.0))"
```

### 3. Central Web Console Dashboards ([`poc_dashboards.json`](./poc_dashboards.json))
1. **`mesh-storage-auditor` (Fleet Overview):**
   ```sql
   SELECT "$producer" AS node_id,
          json_extract(raw_json, '$.fields.node_role') AS node_role,
          CAST(json_extract(raw_json, '$.fields.total_stored_messages') AS INTEGER) AS total_messages,
          CAST(json_extract(raw_json, '$.fields.health_score') AS INTEGER) AS health_score,
          CAST(json_extract(raw_json, '$.fields.irregularity_count') AS INTEGER) AS anomalies,
          json_extract(raw_json, '$.fields.status_summary') AS status_summary,
          "$when" AS audited_at
   FROM v_messages
   WHERE "$type" = 'MessageStateAudit'
   ORDER BY "$when" DESC LIMIT 20
   ```
2. **`mesh-routing-leak-inspector` (Critical Anomaly Triage):**
   ```sql
   SELECT "$when" AS detected_at,
          "$producer" AS reporting_node,
          CAST(json_extract(raw_json, '$.fields.health_score') AS INTEGER) AS health_score,
          CAST(json_extract(raw_json, '$.fields.irregularity_count') AS INTEGER) AS anomaly_count,
          json_extract(raw_json, '$.fields.status_summary') AS triage_notes
   FROM v_messages
   WHERE "$type" = 'MessageStateAudit'
     AND CAST(json_extract(raw_json, '$.fields.irregularity_count') AS INTEGER) > 0
   ORDER BY "$when" DESC LIMIT 50
   ```

---

## 🚀 Step-by-Step Developer Guide

### Step 0: Preflight Static Validation
Before touching a live cluster, validate all artifacts offline:
```bash
# 1. Validate CEL Reactor syntax and AST compilation
./.e2e-root/current/bin/bindu-studio cel-reactor validate \
  --base-dir ./.e2e-root/current \
  --file public/console/docs/using-system-message-types/poc_cel_reactor_config.json

# 2. Validate Dashboard schemas and SQL view queries
./.e2e-root/current/bin/bindu-studio dashboard validate \
  --base-dir ./.e2e-root/current \
  --file public/console/docs/using-system-message-types/poc_dashboards.json

# 3. Validate Solution Manifest and Strict Bundle Closure
./.e2e-root/current/bin/bindu-studio solution validate \
  --base-dir ./.e2e-root/current \
  --file public/console/docs/using-system-message-types/solution.bundle.json \
  --strict
```

### Step 1: Deploy via Universal Deployment Automation
Execute the automated 5-phase deployment script:
```bash
./public/console/docs/using-system-message-types/deploy_mas_solution.sh \
  --base-dir ./.e2e-root/current \
  --root-url http://127.0.0.1:38080 \
  --auth console:StrongPass123 \
  --yes
```

The script automatically executes:
- **Phase 0:** Strict offline static analysis.
- **Phase 1:** Merges `MessageStateAudit` into Root's schema catalog.
- **Phase 2:** Deploys `mesh-message-state-retention-auditor` on Root.
- **Phase 3 & 4:** Deploys the declarative solution bundle (`solution.bundle.json`) via `solution apply`, ensuring both operational dashboards are registered with their isolated `$producer` retention partitions and activating `MASSolutionSet`.

- **Phase 5:** Runs a preflight verification query against `v_messages`.

---

## 🔎 Verification & Zero-RAG Observability

Per Bindu engineering guidelines, cluster verification is performed strictly via the native REST SQL query endpoint (`POST /api/v1/messages/sql`), never by touching SQLite `.db` files directly on disk.

### 1. Verify Active Solution Registration
```bash
curl -s -u console:StrongPass123 \
  -X POST http://127.0.0.1:38080/api/v1/messages/sql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "SELECT \"$producer\", \"$type\", \"$when\", json_extract(raw_json, '\''$.fields.solution_id'\'') as solution_id, json_extract(raw_json, '\''$.fields.name'\'') as name FROM v_messages WHERE \"$type\" = '\''MASSolutionSet'\'' AND json_extract(raw_json, '\''$.fields.solution_id'\'') = '\''mesh-system-audit-solution'\'' ORDER BY \"$when\" DESC LIMIT 1;"
  }'
```

### 2. Verify Incoming Augmented Audit Telemetry
```bash
curl -s -u console:StrongPass123 \
  -X POST http://127.0.0.1:38080/api/v1/messages/sql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "SELECT \"$producer\" AS node_id, json_extract(raw_json, '\''$.fields.health_score'\'') AS score, json_extract(raw_json, '\''$.fields.total_stored_messages'\'') AS total_msgs, json_extract(raw_json, '\''$.fields.status_summary'\'') AS summary, \"$when\" AS ts FROM v_messages WHERE \"$type\" = '\''MessageStateAudit'\'' ORDER BY \"$when\" DESC LIMIT 5;"
  }'
```

### 3. Trigger Fresh Telemetry on Demand
To immediately trigger a node to emit fresh `MessageState` telemetry:
```bash
./.e2e-root/current/bin/bindu-studio mesh node-refresh <node_id> \
  --base-dir ./.e2e-root/current \
  --auth console:StrongPass123
```
As soon as the node responds, `mesh-message-state-retention-auditor` evaluates the message and refreshes the dashboards instantly.

---

## 🎓 Summary of Key Takeaways
- **Built-in Telemetry Augmentation:** Existing infrastructure heartbeats (`MessageState`) can be enriched into domain-specific governance events without touching node binaries or core Go engine code.
- **Microsecond In-Process CEL Kinetics:** In-process CEL evaluations run in sub-microseconds without thread contention, external RPC dependencies, or garbage collection spikes.
- **Topological Symmetry:** Because `MessageState` routes `["parent", "local"]`, centralizing audit kinetics on `root` provides 100% mesh visibility while imposing 0% processing burden on edge gateways.
