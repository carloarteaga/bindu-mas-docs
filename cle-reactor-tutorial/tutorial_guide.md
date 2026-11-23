# Proactive EPN: Scheduled CEL Reactor & Performance Auditor Tutorial ⏰⚡

This tutorial guides you through configuring, validating, and deploying a **Proactive Event Processing Agent (EPA)** using a **CEL Reactor (`CELReactorSet`)**. 

Unlike a reactive agent that waits for external telemetry messages to arrive, this CEL Reactor runs autonomously on a cron schedule (`@every 1m`), queries historical performance logs from the local SQLite database to compute average model latency, and publishes a structured `AIExecutionCognivityAuditory` message to the mesh.

---

## 1. Didactic Architecture: Reactive vs. Proactive EPAs

In a classic Event Processing Network (EPN), Event Processing Agents (EPAs) are usually reactive, triggered by incoming events. However, Bindu supports **Triggering Dualism**, enabling agents to also act **proactively** using internal cron timers.

```
                  ⏰ INTERNAL CRON TIMER (Every 1 Minute)
                             │
                             ▼
                  [ 1. WAKE UP CEL REACTOR ]
                             │
                             ▼
            [ 2. CONTEXT HYDRATION (SQLite Read) ]
              SELECT last 5 AICognitiveProfile records
                             │
                             ▼
             [ 3. IN-PROCESS CEL TRANSFORMATIONS ]
               Evaluate avg latency & status level
                             │
                             ▼
             [ 4. PUBLISH AUDIT EVENT TO MESH ]
               AIExecutionCognivityAuditory published
```

Because there is no incoming message during a scheduled execution, **Context Hydration (SQLite Query) is strictly mandatory**. The reactor reads the state history directly from `v_messages` to compile the variables used during payload transformations.

---

## 2. Deploy the Prerequisites (Message Types)

If you haven't deployed the message types from the Media Storage tutorial, deploy them now. This imports `AIExecutionCognivityAuditory` and its dependencies:

```bash
bindu-studio message-types import \
  --in-path ./public/console/docs/media-storage-tutorial/poc_message_types.json \
  --auth console:<password> --all
```

---

## 3. Deep-Dive into the CEL Reactor Configuration

Below is the complete `CELReactorSet` JSON payload designed to audit performance. It contains a single reactor set configuration with five independent CEL reactor instances auditing the different AI models deployed across the tutorials and tests:

```json
{
  "$type": "CELReactorSet",
  "$node-dest": "root",
  "$group": "mesh",
  "$producer": "root",
  "$user": "admin",
  "$when": "2026-07-31T20:00:00Z",
  "fields": {
    "gateway_node_id": "root",
    "reactors": [
      {
        "reactor_id": "cognitive-execution-auditor-cel-patio",
        "input_message_type": "",
        "cron": "@every 1m",
        "output_message_type": "AIExecutionCognivityAuditory",
        "context_query": "SELECT (SELECT COALESCE(json_extract(raw_json, '$.fields.target_model'), 'moondream:latest') FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'patio-flower-counter' ORDER BY \"$when\" DESC LIMIT 1) AS model_id, (SELECT COALESCE(AVG(CAST(json_extract(raw_json, '$.fields.hydration_latency_ms') AS REAL) + CAST(json_extract(raw_json, '$.fields.inference_latency_ms') AS REAL)), 0) FROM (SELECT raw_json FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'patio-flower-counter' ORDER BY \"$when\" DESC LIMIT 5)) AS avg_latency",
        "trigger_condition": "true",
        "transformation_expression": {
          "model_id": "has(db.rows) && size(db.rows) > 0 ? string(db.rows[0].fields.model_id) + ' (patio)' : 'moondream:latest (patio)'",
          "avg_latency_ms": "has(db.rows) && size(db.rows) > 0 ? int(double(db.rows[0].fields.avg_latency)) : 0",
          "audit_status": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'OPTIMAL' : (double(db.rows[0].fields.avg_latency) < 3000.0 ? 'ACCEPTABLE' : (double(db.rows[0].fields.avg_latency) < 5000.0 ? 'DEGRADED' : 'ANOMALOUS'))) : 'OPTIMAL'",
          "efficiency_summary": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'Execution efficiency is optimal. Edge SLM is performing within target bounds.' : 'Execution latency is elevated. Consider model right-sizing or local compute optimization.') : 'No telemetry data found. System nominal.'"
        }
      },
      {
        "reactor_id": "cognitive-execution-auditor-cel-patio-llava",
        "input_message_type": "",
        "cron": "@every 1m",
        "output_message_type": "AIExecutionCognivityAuditory",
        "context_query": "SELECT (SELECT COALESCE(json_extract(raw_json, '$.fields.target_model'), 'llava-phi3:latest') FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'patio-flower-counter-llava:7b' ORDER BY \"$when\" DESC LIMIT 1) AS model_id, (SELECT COALESCE(AVG(CAST(json_extract(raw_json, '$.fields.hydration_latency_ms') AS REAL) + CAST(json_extract(raw_json, '$.fields.inference_latency_ms') AS REAL)), 0) FROM (SELECT raw_json FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'patio-flower-counter-llava:7b' ORDER BY \"$when\" DESC LIMIT 5)) AS avg_latency",
        "trigger_condition": "true",
        "transformation_expression": {
          "model_id": "has(db.rows) && size(db.rows) > 0 ? string(db.rows[0].fields.model_id) + ' (patio-llava)' : 'llava-phi3:latest (patio-llava)'",
          "avg_latency_ms": "has(db.rows) && size(db.rows) > 0 ? int(double(db.rows[0].fields.avg_latency)) : 0",
          "audit_status": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'OPTIMAL' : (double(db.rows[0].fields.avg_latency) < 3000.0 ? 'ACCEPTABLE' : (double(db.rows[0].fields.avg_latency) < 5000.0 ? 'DEGRADED' : 'ANOMALOUS'))) : 'OPTIMAL'",
          "efficiency_summary": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'Execution efficiency is optimal. Edge SLM is performing within target bounds.' : 'Execution latency is elevated. Consider model right-sizing or local compute optimization.') : 'No telemetry data found. System nominal.'"
        }
      },
      {
        "reactor_id": "cognitive-execution-auditor-cel-freezer",
        "input_message_type": "",
        "cron": "@every 1m",
        "output_message_type": "AIExecutionCognivityAuditory",
        "context_query": "SELECT (SELECT COALESCE(json_extract(raw_json, '$.fields.target_model'), 'qwen2.5-coder:0.5b') FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'freezer-cold-chain-guard' ORDER BY \"$when\" DESC LIMIT 1) AS model_id, (SELECT COALESCE(AVG(CAST(json_extract(raw_json, '$.fields.hydration_latency_ms') AS REAL) + CAST(json_extract(raw_json, '$.fields.inference_latency_ms') AS REAL)), 0) FROM (SELECT raw_json FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'freezer-cold-chain-guard' ORDER BY \"$when\" DESC LIMIT 5)) AS avg_latency",
        "trigger_condition": "true",
        "transformation_expression": {
          "model_id": "has(db.rows) && size(db.rows) > 0 ? string(db.rows[0].fields.model_id) + ' (freezer)' : 'qwen2.5-coder:0.5b (freezer)'",
          "avg_latency_ms": "has(db.rows) && size(db.rows) > 0 ? int(double(db.rows[0].fields.avg_latency)) : 0",
          "audit_status": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'OPTIMAL' : (double(db.rows[0].fields.avg_latency) < 3000.0 ? 'ACCEPTABLE' : (double(db.rows[0].fields.avg_latency) < 5000.0 ? 'DEGRADED' : 'ANOMALOUS'))) : 'OPTIMAL'",
          "efficiency_summary": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'Execution efficiency is optimal. Edge SLM is performing within target bounds.' : 'Execution latency is elevated. Consider model right-sizing or local compute optimization.') : 'No telemetry data found. System nominal.'"
        }
      },
      {
        "reactor_id": "cognitive-execution-auditor-cel-compressor",
        "input_message_type": "",
        "cron": "@every 1m",
        "output_message_type": "AIExecutionCognivityAuditory",
        "context_query": "SELECT (SELECT COALESCE(json_extract(raw_json, '$.fields.target_model'), 'qwen2.5-coder:0.5b') FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'compressor-maintenance-guard' ORDER BY \"$when\" DESC LIMIT 1) AS model_id, (SELECT COALESCE(AVG(CAST(json_extract(raw_json, '$.fields.hydration_latency_ms') AS REAL) + CAST(json_extract(raw_json, '$.fields.inference_latency_ms') AS REAL)), 0) FROM (SELECT raw_json FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'compressor-maintenance-guard' ORDER BY \"$when\" DESC LIMIT 5)) AS avg_latency",
        "trigger_condition": "true",
        "transformation_expression": {
          "model_id": "has(db.rows) && size(db.rows) > 0 ? string(db.rows[0].fields.model_id) + ' (compressor)' : 'qwen2.5-coder:0.5b (compressor)'",
          "avg_latency_ms": "has(db.rows) && size(db.rows) > 0 ? int(double(db.rows[0].fields.avg_latency)) : 0",
          "audit_status": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'OPTIMAL' : (double(db.rows[0].fields.avg_latency) < 3000.0 ? 'ACCEPTABLE' : (double(db.rows[0].fields.avg_latency) < 5000.0 ? 'DEGRADED' : 'ANOMALOUS'))) : 'OPTIMAL'",
          "efficiency_summary": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'Execution efficiency is optimal. Edge SLM is performing within target bounds.' : 'Execution latency is elevated. Consider model right-sizing or local compute optimization.') : 'No telemetry data found. System nominal.'"
        }
      },
      {
        "reactor_id": "cognitive-execution-auditor-cel-auditor",
        "input_message_type": "",
        "cron": "@every 1m",
        "output_message_type": "AIExecutionCognivityAuditory",
        "context_query": "SELECT (SELECT COALESCE(json_extract(raw_json, '$.fields.target_model'), 'qwen2.5:0.5b') FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'cognitive-execution-auditor' ORDER BY \"$when\" DESC LIMIT 1) AS model_id, (SELECT COALESCE(AVG(CAST(json_extract(raw_json, '$.fields.hydration_latency_ms') AS REAL) + CAST(json_extract(raw_json, '$.fields.inference_latency_ms') AS REAL)), 0) FROM (SELECT raw_json FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'cognitive-execution-auditor' ORDER BY \"$when\" DESC LIMIT 5)) AS avg_latency",
        "trigger_condition": "true",
        "transformation_expression": {
          "model_id": "has(db.rows) && size(db.rows) > 0 ? string(db.rows[0].fields.model_id) + ' (auditor)' : 'qwen2.5:0.5b (auditor)'",
          "avg_latency_ms": "has(db.rows) && size(db.rows) > 0 ? int(double(db.rows[0].fields.avg_latency)) : 0",
          "audit_status": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'OPTIMAL' : (double(db.rows[0].fields.avg_latency) < 3000.0 ? 'ACCEPTABLE' : (double(db.rows[0].fields.avg_latency) < 5000.0 ? 'DEGRADED' : 'ANOMALOUS'))) : 'OPTIMAL'",
          "efficiency_summary": "has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'Execution efficiency is optimal. Edge SLM is performing within target bounds.' : 'Execution latency is elevated. Consider model right-sizing or local compute optimization.') : 'No telemetry data found. System nominal.'"
        }
      },
      {
        "reactor_id": "freezer-cold-chain-guard-cel",
        "input_message_type": "FreezerTelemetry",
        "cron": "",
        "output_message_type": "StorageRiskAssessment",
        "context_query": "SELECT raw_json FROM v_messages WHERE \"$type\" = 'FreezerTelemetry' AND json_extract(raw_json, '$.fields.target_producer') = ? ORDER BY \"$when\" DESC LIMIT 1",
        "trigger_condition": "true",
        "transformation_expression": {
          "risk_level": "double(payload.temperature) > -15.0 || bool(payload.door_open) ? 'CRITICAL' : 'NOMINAL'",
          "estimated_minutes_to_spoilage": "double(payload.temperature) > -15.0 || bool(payload.door_open) ? 15 : 1440",
          "root_cause_prediction": "bool(payload.door_open) ? 'Freezer door left open' : (double(payload.temperature) > -15.0 ? 'Cooling system efficiency degradation' : 'No anomalies detected')",
          "recommended_action_code": "double(payload.temperature) > -15.0 || bool(payload.door_open) ? 'CHECK_COOLING_AND_CLOSE_DOOR' : 'NO_ACTION_REQUIRED'",
          "target_producer": "string(head[\"$producer\"])"
        }
      }
    ]
  }
}
```

---

## 4. Deconstructing the Fields

### A. The SQLite Context Query
To avoid writing complex loop calculations in CEL, we leverage SQLite's built-in aggregation engine. For the proactive auditors, the query selects two values:
1. **`model_id`**: Subquery extracts the `target_model` value from the latest `AICognitiveProfile` event for `patio-flower-counter`.
2. **`avg_latency`**: Subquery retrieves the last 5 `AICognitiveProfile` JSON payloads, extracts both `hydration_latency_ms` and `inference_latency_ms`, casts them as real numbers, sums them up, and computes their mathematical average. If no records are found, `COALESCE` defaults to `0`.

### B. CEL Transformation Expressions
These sandboxed expressions compile into binary instructions in memory and map values to the output properties:
* **`model_id`**: Safety checks that rows exist, converts the column value to string, or falls back to a default value.
  ```javascript
  has(db.rows) && size(db.rows) > 0 ? string(db.rows[0].fields.model_id) : 'moondream:latest'
  ```
* **`avg_latency_ms`**: Converts the double average latency float returned by SQLite into a strict 64-bit integer required by the schema.
  ```javascript
  has(db.rows) && size(db.rows) > 0 ? int(double(db.rows[0].fields.avg_latency)) : 0
  ```
* **`audit_status`**: Implements a nested ternary conditional operator to decide status limits:
  ```javascript
  has(db.rows) && size(db.rows) > 0 ? (double(db.rows[0].fields.avg_latency) < 1500.0 ? 'OPTIMAL' : (double(db.rows[0].fields.avg_latency) < 3000.0 ? 'ACCEPTABLE' : (double(db.rows[0].fields.avg_latency) < 5000.0 ? 'DEGRADED' : 'ANOMALOUS'))) : 'OPTIMAL'
  ```
* **`efficiency_summary`**: Writes a diagnostic critique text depending on the computed latency range.

### C. Reactive LLM-to-CEL Migration: `freezer-cold-chain-guard-cel`
This reactor demonstrates how to migrate a computationally expensive, reactive Semantic AI Reactor (`freezer-cold-chain-guard`) to a sub-millisecond, local-evaluation CEL Reactor.
* **Trigger Mechanism**: It is reactive. The `input_message_type` is set to `FreezerTelemetry` (and `cron` is empty), so it executes immediately when a telemetry event is published.
* **`risk_level`**: Evaluates whether temperature exceeds `-15.0` or the door is open.
  ```javascript
  double(payload.temperature) > -15.0 || bool(payload.door_open) ? 'CRITICAL' : 'NOMINAL'
  ```
* **`estimated_minutes_to_spoilage`**: Evaluates a deterministic response time (15 minutes if critical, 1440 minutes if nominal).
  ```javascript
  double(payload.temperature) > -15.0 || bool(payload.door_open) ? 15 : 1440
  ```
* **`root_cause_prediction`**: Dynamically writes a textual cause based on the triggered sensors.
  ```javascript
  bool(payload.door_open) ? 'Freezer door left open' : (double(payload.temperature) > -15.0 ? 'Cooling system efficiency degradation' : 'No anomalies detected')
  ```
* **`recommended_action_code`**: Suggests standard operation actions.
  ```javascript
  double(payload.temperature) > -15.0 || bool(payload.door_open) ? 'CHECK_COOLING_AND_CLOSE_DOOR' : 'NO_ACTION_REQUIRED'
  ```

> [!CAUTION]
> **🚫 Anti-Recursion Safety Invariant (`E_REACTOR_RECURSIVE_LOOP`):**
> Notice that `freezer-cold-chain-guard-cel` defines `input_message_type: "FreezerTelemetry"` and `output_message_type: "StorageRiskAssessment"`. 
> In Bindu, **the output message type CANNOT be identical to the input message type**. If a reactor emitted `FreezerTelemetry`, publishing it would re-trigger the reactor immediately, entering an infinite loop. Always emit a distinct downstream event.

---

## 5. Deployment Guide

### Option A: Deploy via Console UI (Interactive)
1. Open the **Bindu Console UI** and click on **🔀 CEL Reactor** in the sidebar.
2. Select target node **`root`** from the node dropdown list.
3. Click **➕ Add CEL Reactor**.
4. Fill in the form fields for one of the auditors (e.g., `cognitive-execution-auditor-cel-patio`):
   * **Reactor ID:** `cognitive-execution-auditor-cel-patio`
   * **Cron Schedule:** `@every 1m`
   * **Output Type:** `AIExecutionCognivityAuditory`
   * **Context Query:** Paste the SQL query for the patio auditor from the JSON above.
   * **Trigger Condition:** `true`
5. In the **Transformations** section, configure the four keys with the respective CEL expressions:
   * `model_id` ➔ expression string
   * `avg_latency_ms` ➔ expression string
   * `audit_status` ➔ expression string
   * `efficiency_summary` ➔ expression string
6. Click **Save Instance**, and repeat for the other auditors if desired. Finally, click **🚀 Deploy Node Configuration**.

### Option B: Automated CLI Import
Deploy all five pre-packaged configurations instantly using `bindu-studio`:
```bash
bindu-studio cel-reactor import \
  --file ./public/console/docs/cle-reactor-tutorial/poc_cel_reactor_config.json \
  --auth console:<password> --yes
```

---

## 6. Verification and Output Checking

Once deployed, the `root` engine starts the background cron runners. Every 60 seconds, they compile and run the SQL queries, evaluate the CEL mappings, and publish new `AIExecutionCognivityAuditory` events.

To verify the audit events are generating:
1. Open the **Queries** dashboard tab in the Console.
2. Run the SQL query:
   ```sql
   SELECT 
     datetime("$when") AS timestamp,
     json_extract(raw_json, '$.fields.model_id') AS model,
     json_extract(raw_json, '$.fields.audit_status') AS status,
     CAST(json_extract(raw_json, '$.fields.avg_latency_ms') AS INTEGER) AS avg_latency_ms,
     json_extract(raw_json, '$.fields.efficiency_summary') AS summary
   FROM v_messages 
   WHERE "$type" = 'AIExecutionCognivityAuditory' 
   ORDER BY datetime("$when") DESC;
   ```
3. You will see new rows generated every minute showing the dynamic latency audit values!

---

## 7. Console UI Dashboard for Cognitive Observability & Latency Comparison 📊🧠

To monitor the real-time cognitive latencies, compare model performance, and spot edge hardware degradation, configure a dedicated dashboard in the **Bindu Console UI** using the SQL Queries engine.

### 📊 Dashboard: Cognitive Latency Auditor Cockpit

This cockpit provides a side-by-side comparison of local/cloud model latencies, illustrating the "Semantic Friction" and execution overhead on the mesh.

#### Query A: Model Latency Comparison Chart (Real-time comparison)
This query groups the latest audit reports for each model and converts their latency to seconds for visual scaling.
```sql
SELECT 
  model_id,
  avg_latency_sec,
  status,
  timestamp
FROM (
  SELECT 
    json_extract(raw_json, '$.fields.model_id') AS model_id,
    CAST(json_extract(raw_json, '$.fields.avg_latency_ms') AS REAL) / 1000.0 AS avg_latency_sec,
    json_extract(raw_json, '$.fields.audit_status') AS status,
    datetime("$when") AS timestamp
  FROM v_messages
  WHERE "$type" = 'AIExecutionCognivityAuditory'
  ORDER BY datetime("$when") DESC
)
GROUP BY model_id;
```
* **Dashboard Name:** `Cognitive Latency Auditor`
* **Chart Title:** `AI Model Latency Comparison (Seconds)`
* **Chart Type:** `Bar Chart`
* **X-Axis:** `model_id`
* **Y-Axis / Series:** `avg_latency_sec`

#### Query B: Detailed System Audit Grid (Tabular Feed)
Displays the latest complete audit status and critiques for each model path.
```sql
SELECT 
  datetime("$when") AS timestamp,
  json_extract(raw_json, '$.fields.model_id') AS model,
  json_extract(raw_json, '$.fields.audit_status') AS status,
  CAST(json_extract(raw_json, '$.fields.avg_latency_ms') AS INTEGER) AS avg_latency_ms,
  json_extract(raw_json, '$.fields.efficiency_summary') AS summary
FROM (
  SELECT *,
    json_extract(raw_json, '$.fields.model_id') AS model_id
  FROM v_messages
  WHERE "$type" = 'AIExecutionCognivityAuditory'
  ORDER BY datetime("$when") DESC
)
GROUP BY model_id;
```
* **Dashboard Name:** `Cognitive Latency Auditor`
* **Widget Title:** `Active Model Audit Logs`
* **Widget Type:** `Data Table`
* **Columns:** `timestamp`, `model`, `status`, `avg_latency_ms`, `summary`
