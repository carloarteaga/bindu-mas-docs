# Demo Concept: Semantic Predictive Cold Chain

This tutorial demonstrates the value of the **Unified AI-Pushing Architecture** in Bindu using a critical, real-world case study: predictive vaccine monitoring in pharmaceutical warehouses.

---

## 1. Mesh Topology & Geographical Representation

To model the client's infrastructure, we organize the mesh hierarchically. We use **reference nodes (`connectivity_kind: "reference"`)** to represent warehouses, aisles, and racks, and **network nodes (gateways)** for the physical compute devices.

A single gateway aggregates data from all aisles and racks under its physical scope.

### Mesh Hierarchy

```
Root Node (Central Control Room)
├── Madrid Central Warehouse (connectivity_kind: "reference", ID: warehouse-a)
│   └── Gateway Node Central (connectivity_kind: "gateway", ID: node-1)
│       └── Aisle 1 (connectivity_kind: "reference", ID: warehouse-a-aisle-1)
│           └── Rack 1 (connectivity_kind: "reference", ID: warehouse-a-aisle-1-rack-1)
│
├── Barcelona Regional Warehouse (connectivity_kind: "reference", ID: warehouse-b)
│   └── Gateway Node Minis (connectivity_kind: "gateway", ID: bindu-minis)
│       └── Aisle 4 (connectivity_kind: "reference", ID: warehouse-b-aisle-4)
│           └── Rack 2 (connectivity_kind: "reference", ID: warehouse-b-aisle-4-rack-2)
│
└── Valencia Regional Warehouse (connectivity_kind: "reference", ID: warehouse-c)
    └── Gateway Node Pi3 (connectivity_kind: "gateway", ID: bindu-pi3-uno)
        └── Aisle 7 (connectivity_kind: "reference", ID: warehouse-c-aisle-7)
            └── Rack 3 (connectivity_kind: "reference", ID: warehouse-c-aisle-7-rack-3)
```

Each physical gateway (`node-1`, `bindu-minis`, and `bindu-pi3-uno`) runs a **Semantic Reactor Orchestrator** and locally and independently processes telemetry from its respective warehouse.

---

## 2. Decentralized Context Hydration & System One AI Decision Engine

*   **Primary Decision Engine (System One)**: **TypeSafe AI (`jev-latest`)** via `POST https://api.typesafe.ai/v1/systemone`, evaluating parallel `noul` (boolean probability), `choice` (discrete enum), and `score` (ordered rubric) questions with mathematically calibrated confidence (`confidence_threshold: 0.85`).
*   **Air-Gapped Fallback Option (System Two)**: Local **Qwen 2.5 (0.5B / 1.5B)** via Ollama (`http://192.168.12.160:11434/v1/chat/completions`) using Bindu's grammar-guided JSON decoding.
*   **Endpoint Configuration**:
    *   All gateway nodes (`node-1`, `bindu-minis`, and `bindu-pi3-uno`) run their own local `bindu-semantic-reactor` worker, hydrating historical telemetry from local SQLite (`v_messages`) and invoking `/v1/systemone` to emit `StorageRiskAssessment` events.

---

## 3. Direct Ingress Simulation via MQTT

Instead of importing static historical data to the central database of the Root via HTTP, the telemetry simulation injects real-time messages directly into the Ingress pipeline of each node using an external MQTT client (`mosquitto_pub` or similar).

The simulation script `simulate_cold_chain_anomaly.sh` connects to the local or integration MQTT broker of each node based on the port configuration from the E2E infrastructure:
*   **Central Warehouse (node-1)**: Publishes to the integration MQTT port of `node-1`.
*   **Regional Warehouse (bindu-minis)**: Publishes to the integration MQTT port of `bindu-minis`.
*   **Valencia Warehouse (bindu-pi3-uno)**: Publishes to the integration MQTT port of `bindu-pi3-uno` (IP `192.168.12.108`).

### Ingress Structure (`FreezerTelemetry`)
The telemetry sensor publishes `FreezerTelemetry` type events to the `bindu/in` topic. The physical location is defined in the `$where` header using the structural hierarchy:

```json
{
  "$type": "FreezerTelemetry",
  "$producer": "freezer_madrid_01",
  "$group": "mesh",
  "$user": "admin",
  "$when": "2026-06-03T18:40:00Z",
  "$where": {
    "locagram": "/warehouse-a/node-1/aisle-1/rack-1"
  },
  "fields": {
    "temperature": -12.4,
    "door_open": true,
    "compressor_current_amps": 14.8
  }
}
```

---

## 4. Semantic Reasoning & Autonomous Egress

1.  **Activation**: The sensor publishes telemetry. Upon arriving at the node (e.g., `node-1`), the semantic reactor wakes up triggered by the `FreezerTelemetry` type.
2.  **Context Loop (Hydration)**: The reactor performs a local SQL query to `v_messages` to retrieve the last 5 telemetry events of the freezer (`freezer_madrid_01`) and reconstruct the temperature trend.
3.  **Semantic Inference**: The reactor generates a prompt with the physical history and the semantic contract template, sending it to the local Ollama instance at `192.168.12.160`.
4.  **Egress Generation**: The LLM predicts the risk of loss and responds with a strict JSON. The orchestrator validates the schema and autonomously publishes a `StorageRiskAssessment` message onto the local mesh bus:

```json
{
  "$type": "StorageRiskAssessment",
  "$producer": "freezer-cold-chain-guard",
  "$group": "mesh",
  "$user": "admin",
  "$when": "2026-06-03T18:40:02Z",
  "$where": {
    "locagram": "/warehouse-a/node-1/aisle-1/rack-1"
  },
  "fields": {
    "risk_level": "CRITICAL",
    "estimated_minutes_to_spoilage": 11,
    "root_cause_prediction": "Continuous temperature increase due to prolonged open door under load",
    "recommended_action_code": "ACTUATE_CLOSE_VALVE_OR_ALERT_FLOOR"
  }
}
```

This event is automatically propagated to the Root and other mesh nodes subscribed to critical alerts, demonstrating the decentralized and real-time proactivity of the Bindu bus.