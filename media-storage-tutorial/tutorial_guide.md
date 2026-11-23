# Out-of-Band Media Storage & Multimodal AI Reactor Tutorial

This tutorial guides you through configuring a local **MinIO** storage server, starting a local **Ollama** server with vision models, deploying message types for camera captures and visual analytics, and setting up a **Semantic AI Reactor** that uses `moondream` to analyze backyard patio photos and output flower counts by color.

---

## 1. Setup & Launch MinIO
Run MinIO locally using the required user credentials:
```bash
export MINIO_ROOT_USER="bindu-core-agent"
export MINIO_ROOT_PASSWORD="bindu-media-123"

# Create test storage directory
mkdir -p ~/bindu-media-test

# Start MinIO server on ports 9000 (API) and 9001 (Console)
minio server ~/bindu-media-test --address ":9000" --console-address ":9001"
```
Navigate to the MinIO Console at `http://192.168.12.160:9001`, log in, and create a bucket named **`raw`**.

---

## 2. Setup Ollama on the Node (`192.168.12.160`)
To run local text reasoning and visual analysis, pull the required models:
```bash
# Pull the qwen helper model (text)
ollama pull qwen2.5:0.5b

# Pull the moondream vision model (multimodal)
ollama pull moondream
```
Ensure Ollama is running and accessible on port `11434`.

---

## 3. Register Secrets & S3 Provider in Bindu

Under the Bindu Multi-Agent System (MAS) architecture, secrets like MinIO storage keys or cloud LLM tokens must **never** be transmitted or stored in plaintext within configuration manifests. Bindu enforces the **Zero-Plaintext Wire Guarantee**: secrets are encrypted locally in AES-GCM vault (`accessControlConfig.brs`) and referenced by name (`secret:KEY_NAME`).

First, save the required secrets encrypted in Bindu's access control vault:
```bash
# MinIO S3 media storage secret
go run ./cmd/bindu-studio access-control secrets set minio_media_secret --value "bindu-media-123" --yes

# Ollama vision inference secret (for edge VLM workers)
go run ./cmd/bindu-studio access-control secrets set OLLAMA_API_KEY --value "ollama-local" --yes

# TypeSafe AI System One decision token (for cognitive auditor)
go run ./cmd/bindu-studio access-control secrets set TYPESAFE_API_KEY --value "YOUR_ACTUAL_TYPESAFE_API_KEY" --yes

# Verify vault status
go run ./cmd/bindu-studio access-control secrets validate
```

Next, open the **Bindu Console UI**, click on **📁 Media Storage** in the sidebar, and register the provider:
* **Provider ID:** `minio_local`
* **Endpoint URL:** `http://localhost:9000` (or `http://192.168.12.160:9000`)
* **Access Key:** `bindu-core-agent`
* **Secret Key:** `secret:minio_media_secret`

Click **Apply & Broadcast Configuration**. This validates the `secret:` format, configures the provider in Bindu's memory, and replicates it across the mesh.

---

## 4. Deploy Custom Message Types
Deploy `PatioMonitorMedia` and `CountFlowersByColor` using `bindu-studio`:
```bash
bindu-studio message-types import \
  --in-path ./public/console/docs/media-storage-tutorial/poc_message_types.json \
  --auth console:<password> --all
```

---

## 5. Deploy the Semantic AI Reactor (`AISemanticReactorSet`)

The **Semantic AI Reactor** connects raw telemetry messages (such as `PatioMonitorMedia`) with local vision LLMs (such as `moondream:latest` running on Ollama).

When an incoming `PatioMonitorMedia` message arrives at the gateway node:
1. The AI Reactor intercepts the message.
2. It resolves the `patio_view` media field (`minio_local://raw/PatioMonitorMedia/plant-1/plant-1.jpeg`) against the active `MediaStorageSet` configuration (`http://192.168.12.160:9000`, credentials `bindu-core-agent` / `secret:minio_media_secret` resolved dynamically from `accessControlConfig.brs`).
   > [!NOTE]
   > **Semantic URI Pattern:** Bindu standardizes object keys as `{provider}://{bucket}/{message_type}/{producer}/{filename}` (e.g. `minio_local://raw/PatioMonitorMedia/plant-1/plant-1.jpeg`). This allows AI agents and retention pipelines to instantly deduce the schema origin, device producer, and retention rules directly from the URI.
3. It passes the image frame capture to the local `moondream:latest` vision model via Ollama (`http://192.168.12.160:11434/v1/chat/completions`).
4. `moondream:latest` analyzes the patio photo, counts the flowers by color (`yellow_flowers`, `orange_light_flowers`, `orange_dark_flowers`), calculates `total_flowers`, and outputs a descriptive `analysis_summary`.
5. The reactor packages the structured response into a new `CountFlowersByColor` message and publishes it to the mesh.

### Option A: Copy & Paste via Console UI (Interactive)
You can deploy the reactor directly in the **Bindu Console UI** under **🧠 AI Reactor** or via **Publish Message** using the following ready-to-use `AISemanticReactorSet` JSON payload:

```json
{
  "$type": "AISemanticReactorSet",
  "$group": "mesh",
  "$producer": "root",
  "$user": "admin",
  "$when": "2026-07-17T20:00:00Z",
  "fields": {
    "gateway_node_id": "bindu-minis",
    "reactors": [
      {
        "reactor_id": "patio-flower-counter",
        "api_endpoint": "http://192.168.12.160:11434/v1/chat/completions",
        "api_key_secret": "secret:OLLAMA_API_KEY",
        "model_id": "moondream:latest",
        "max_concurrency": 1,
        "inference_timeout": 300,
        "trigger_message_types": [
          "PatioMonitorMedia"
        ],
        "history_count": 0,
        "context_query": "",
        "response_message_type": "CountFlowersByColor",
        "confidence_threshold": 0.5,
        "mapping_template": {
          "yellow_flowers": "{{.fields.yellow_flowers}}",
          "orange_light_flowers": "{{.fields.orange_light_flowers}}",
          "orange_dark_flowers": "{{.fields.orange_dark_flowers}}",
          "total_flowers": "{{add .fields.yellow_flowers .fields.orange_light_flowers .fields.orange_dark_flowers}}",
          "analysis_summary": "{{.fields.analysis_summary}}"
        },
        "system_instruction": "You are a local computer vision flower analyzer. You analyze patio flower captures and count the flowers by color. You MUST return a single raw JSON object with EXACTLY these 5 keys: yellow_flowers, orange_light_flowers, orange_dark_flowers, total_flowers, analysis_summary. Do not output any markdown formatting, preambles, explanations, extra keys, or backticks. Output only raw JSON. Rules: 1. Count yellow_flowers, orange_light_flowers, orange_dark_flowers. 2. total_flowers MUST be an integer representing the sum of all counts. 3. analysis_summary MUST be a string describing the visual status of the backyard plants.",
        "semantic_prompt_template": "Analyze the flower capture for patio '{{index .trigger_event \"$producer\"}}'.\n\nPatio View S3 URI: {{index .trigger_event \"fields\" \"patio_view\"}}\n\nOutput a JSON object with strictly these keys:\n{\n  \"yellow_flowers\": <int>,\n  \"orange_light_flowers\": <int>,\n  \"orange_dark_flowers\": <int>,\n  \"total_flowers\": <int>,\n  \"analysis_summary\": \"<string>\"\n}"
      }
    ]
  }
}
```

### 5.2 Deploy the Cognitive Audit AI Reactor (Second Stage System One Audit)

To audit execution metrics and latency efficiency on the mesh, you can deploy a second-stage **Cognitive Audit AI Reactor** running on the `root` coordinator. This reactor is triggered whenever a `CountFlowersByColor` message arrives, showcasing **Dual-Engine Cognitive Orchestration** (System 2 Vision LLM on the edge + System 1 Decision-Native Evaluator on `root`).

It retrieves the last `5` `AICognitiveProfile` metrics logs from history (`patio-flower-counter`), passes them to **TypeSafe AI (`System One` / `jev-latest`)** to classify `audit_status` (`choice`) and `avg_latency_ms` (`score`), and synthesizes `efficiency_summary` deterministically via Google CEL into an `AIExecutionCognivityAuditory` message.

#### Option A: Copy & Paste via Console UI (Interactive Wizard)
Use the following `AISemanticReactorSet` payload in the console:

> [!WARNING]
> **Avoid Backslash Copy-Paste Errors:**
> When copying the `context_query` SQL string from the JSON block below to paste into the **Context Query** field of the Console UI text box, make sure to remove any backslashes (`\`). Write normal double quotes: `"$type"` and `"$when"`. The backslashes (`\"`) are only used for escaping quotes inside raw JSON files.

```json
{
  "$type": "AISemanticReactorSet",
  "$group": "mesh",
  "$producer": "root",
  "$user": "admin",
  "$when": "2026-10-23T12:00:00Z",
  "fields": {
    "gateway_node_id": "root",
    "reactors": [
      {
        "reactor_id": "cognitive-execution-auditor",
        "provider": "typesafe",
        "api_endpoint": "https://api.typesafe.ai/v1/systemone",
        "api_key_secret": "secret:TYPESAFE_API_KEY",
        "model_id": "jev-latest",
        "max_concurrency": 2,
        "inference_timeout": 30,
        "trigger_message_types": [
          "CountFlowersByColor"
        ],
        "trigger_condition": "payload.total_flowers >= 0",
        "history_count": 5,
        "context_query": "SELECT raw_json FROM v_messages WHERE \"$type\" = 'AICognitiveProfile' AND json_extract(raw_json, '$.fields.reactor_id') = 'patio-flower-counter' ORDER BY \"$when\" DESC LIMIT 5",
        "response_message_type": "AIExecutionCognivityAuditory",
        "confidence_threshold": 0.75,
        "cognitive_profile": true,
        "questions": {
          "audit_status": {
            "type": "choice",
            "instructions": "Evaluate the execution efficiency and latency health of the vision AI reactor based on its AICognitiveProfile telemetry history.",
            "criteria": {
              "OPTIMAL": "Execution succeeded with low hydration (<50ms) and fast vision inference (<1500ms) or nominal empty baseline",
              "ACCEPTABLE": "Execution succeeded with moderate inference latency (1500ms - 3000ms) and zero errors",
              "DEGRADED": "Elevated inference latency (>3000ms) or cold-start bottleneck observed across historical runs",
              "ANOMALOUS": "Erratic latency spikes, schema coercion warnings, or confidence drops detected in profile history",
              "TIMEOUT": "Execution timed out or failed to return a valid response within the deadline"
            }
          },
          "avg_latency_ms": {
            "type": "score",
            "instructions": "Estimate the average total execution latency in milliseconds (0 to 5000 ms) from the AICognitiveProfile history (or 0 if no history logs are present).",
            "criteria": [
              "0 ms: No historical telemetry records yet (nominal baseline)",
              "500 ms: Ultra-fast edge inference latency",
              "1500 ms: Standard multimodal VLM inference latency",
              "3000 ms: Elevated inference latency under edge compute load",
              "5000 ms: Severe latency saturation or cold-start stall"
            ]
          }
        },
        "mapping_template": {
          "model_id": "'moondream:latest'",
          "audit_status": "string(resp.audit_status)",
          "avg_latency_ms": "int(resp.avg_latency_ms)",
          "efficiency_summary": "string(resp.audit_status) == 'OPTIMAL' || string(resp.audit_status) == 'ACCEPTABLE' ? 'System One cognitive audit verified nominal VLM execution (' + string(resp.audit_status) + ', avg ' + string(resp.avg_latency_ms) + ' ms) with RLCD confidence ' + string(ai.round(double(resp.confidence) * 100.0, 1)) + '%.' : 'ALERT: System One cognitive audit flagged ' + string(resp.audit_status) + ' execution state (' + string(resp.avg_latency_ms) + ' ms); inspect edge VLM latency and hydration load.'"
        },
        "system_instruction": "You are a System One cognitive performance auditor evaluating AICognitiveProfile execution telemetry for edge multimodal AI reactors.",
        "semantic_prompt_template": "Audit the execution metrics for the vision model that generated this CountFlowersByColor event (Total Flowers Counted: {{index .trigger_event \"fields\" \"total_flowers\"}}).\n\nHistorical telemetry profiles (AICognitiveProfile):\n{{range .history}}\n- Status: {{index .fields \"execution_status\"}}, Latency: {{index .fields \"hydration_latency_ms\"}} (hydration) + {{index .fields \"inference_latency_ms\"}} (inference) ms, Model: {{index .fields \"target_model\"}}\n{{else}}\n(No historical profiles found; baseline performance is nominal)\n{{end}}"
      }
    ]
  }
}
```

### Option C: Automated CLI Merge & Deploy
Alternatively, merge and deploy the pre-packaged configuration file across gateways via MQTT using `bindu-studio`:
```bash
go run ./cmd/bindu-studio semantic-ai-reactor merge \
  --in-path ./public/console/docs/media-storage-tutorial/poc_semantic_reactor_config.json \
  --all \
  --auth console:StrongPass123
```

---

## 6. Run the Patio Media Simulation
You can run the unified simulation using either the Bash script or the Go script. Both scripts automatically connect directly to MinIO (using the active `MediaStorageSet` credentials), ensure the `raw` bucket exists, upload the local plant images (`plant-1.jpeg`, `plant-2.jpeg`, `plant-3.jpeg`), and publish the matching `PatioMonitorMedia` MQTT telemetry messages.

### Option A: Unified Bash Simulation Script
```bash
./public/console/docs/media-storage-tutorial/simulate_patio_media.sh \
   --host 192.168.12.160 \
   --port 1885 \
   --s3-endpoint http://192.168.12.160:9000 \
   --auth console:StrongPass123
```

### Option B: Direct Go Simulation Script
```bash
go run ./public/console/docs/media-storage-tutorial/publish_media.go \
   -host 192.168.12.160 \
   -port 1885 \
   -user console \
   -pass StrongPass123 \
   -s3-endpoint http://192.168.12.160:9000 \
   -access-key bindu-core-agent \
   -secret-key bindu-media-123 \
   -bucket raw
```

---

## 7. Query and View Media inside Console UI
1. Navigate to the **Queries** tab in the Console UI.
2. Run a query to fetch the raw patio telemetry messages:
   ```sql
   SELECT * FROM v_messages WHERE "$type" = 'PatioMonitorMedia' ORDER BY datetime("$when") DESC;
   ```
   Click the **🎬 View Media** button that appears on any card to verify the MinIO upload and preview the backyard photo.
3. Run a query to fetch the resulting AI flower count analyses:
   ```sql
   SELECT * FROM v_messages WHERE "$type" = 'CountFlowersByColor' ORDER BY datetime("$when") DESC;
   ```
   You will see the structured JSON output generated by the local `moondream` vision model detailing `yellow_flowers`, `orange_light_flowers`, `orange_dark_flowers`, and `total_flowers` counted!

---

## 8. Console UI Dashboards for Visual AI Results 📊🌺

To monitor the real-time visual analytics and media storage pipeline in the **Bindu Console UI**, configure the following 2 dedicated dashboards using the SQL Queries engine.

---

### 📊 Dashboard 1: Vision AI Flower Analytics Cockpit (`CountFlowersByColor`)

This dashboard displays real-time flower counts, color distribution breakdowns, and natural language visual summaries produced asynchronously by the local `moondream:latest` Vision AI model.

#### Query 1A: Flower Counts & Color Distribution Breakdown
```sql
SELECT 
  "$producer" AS patio_camera,
  CAST(json_extract(raw_json, '$.fields.yellow_flowers') AS INTEGER) AS yellow_flowers,
  CAST(json_extract(raw_json, '$.fields.orange_light_flowers') AS INTEGER) AS orange_light_flowers,
  CAST(json_extract(raw_json, '$.fields.orange_dark_flowers') AS INTEGER) AS orange_dark_flowers,
  CAST(json_extract(raw_json, '$.fields.total_flowers') AS INTEGER) AS total_flowers,
  datetime("$when") AS timestamp
FROM v_messages
WHERE "$type" = 'CountFlowersByColor'
ORDER BY datetime("$when") DESC
LIMIT 15;
```
* **Dashboard Name:** `Vision AI Analytics`
* **Chart Title:** `Patio Flower Counts by Color`
* **Chart Type:** `Bar Chart` / `Stacked Bar`
* **X-Axis:** `patio_camera` (Patio ID)
* **Y-Axis / Series:** `yellow_flowers`, `orange_light_flowers`, `orange_dark_flowers`, `total_flowers`

#### Query 1B: Latest Vision Inspection & AI Summary Feed
```sql
SELECT 
  datetime("$when") AS timestamp,
  "$producer" AS patio_camera,
  json_extract(raw_json, '$.fields.total_flowers') AS total_flowers,
  json_extract(raw_json, '$.fields.yellow_flowers') AS yellow,
  json_extract(raw_json, '$.fields.orange_light_flowers') AS orange_light,
  json_extract(raw_json, '$.fields.orange_dark_flowers') AS orange_dark,
  json_extract(raw_json, '$.fields.analysis_summary') AS vision_ai_summary
FROM v_messages
WHERE "$type" = 'CountFlowersByColor'
ORDER BY datetime("$when") DESC
LIMIT 20;
```
* **Dashboard Name:** `Vision AI Analytics`
* **Chart Title:** `Recent Vision Inspection Summaries`
* **Chart Type:** `Data Table`
* **Columns:** `timestamp`, `patio_camera`, `total_flowers`, `yellow`, `orange_light`, `orange_dark`, `vision_ai_summary`

---

### 📷 Dashboard 2: Out-of-Band Media Ingress & Pipeline Telemetry (`PatioMonitorMedia`)

This dashboard tracks MinIO S3 media captures, camera locagrams, and AI reactor execution throughput across the node mesh.

#### Query 2A: S3 Media Storage Captures Feed
```sql
SELECT 
  datetime("$when") AS capture_time,
  "$producer" AS patio_camera,
  json_extract("$where", '$.locagram') AS locagram,
  json_extract(raw_json, '$.fields.patio_view') AS s3_media_uri
FROM v_messages
WHERE "$type" = 'PatioMonitorMedia'
ORDER BY datetime("$when") DESC
LIMIT 15;
```
* **Dashboard Name:** `Media Storage & Pipeline`
* **Chart Title:** `Recent S3 Patio Image Captures`
* **Chart Type:** `Data Table`
* **Columns:** `capture_time`, `patio_camera`, `locagram`, `s3_media_uri`

#### Query 2B: Ingress Captures vs AI Inference Output Throughput
```sql
SELECT 
  strftime('%Y-%m-%d %H:%M', "$when") AS minute,
  "$type" AS event_type,
  COUNT(*) AS event_count
FROM v_messages
WHERE "$type" IN ('PatioMonitorMedia', 'CountFlowersByColor')
GROUP BY minute, "$type"
ORDER BY minute DESC
LIMIT 20;
```
* **Dashboard Name:** `Media Storage & Pipeline`
* **Chart Title:** `Media Captures vs AI Inference Throughput`
* **Chart Type:** `Bar Chart` / `Time Series`
* **X-Axis:** `minute`
* **Y-Axis:** `event_count`
* **Series:** `event_type`

---

### 🧠 Dashboard 3: Cognitive Auditing & Performance Audits (`AIExecutionCognivityAuditory`)

This dashboard visualizes cognitive auditing metrics, showing the latency patterns and overall performance audit statuses (OPTIMAL, ACCEPTABLE, DEGRADED, ANOMALOUS, TIMEOUT) grouped by AI model.

#### Query 3A: Audit Status Distribution by Model
```sql
SELECT 
  json_extract(raw_json, '$.fields.model_id') AS model_id,
  json_extract(raw_json, '$.fields.audit_status') AS audit_status,
  COUNT(*) AS audit_count
FROM v_messages
WHERE "$type" = 'AIExecutionCognivityAuditory'
GROUP BY model_id, audit_status;
```
* **Dashboard Name:** `Cognitive Efficiency Audits`
* **Chart Title:** `Audit Status Distribution by Model`
* **Chart Type:** `Bar Chart` / `Grouped Bar`
* **X-Axis:** `model_id`
* **Y-Axis:** `audit_count`
* **Series:** `audit_status`

#### Query 3B: Cognitive Audit Log Feed
```sql
SELECT 
  datetime("$when") AS audit_time,
  json_extract(raw_json, '$.fields.model_id') AS model_id,
  CAST(json_extract(raw_json, '$.fields.avg_latency_ms') AS INTEGER) AS avg_latency,
  json_extract(raw_json, '$.fields.audit_status') AS status,
  json_extract(raw_json, '$.fields.efficiency_summary') AS audit_critique
FROM v_messages
WHERE "$type" = 'AIExecutionCognivityAuditory'
ORDER BY datetime("$when") DESC
LIMIT 20;
```
* **Dashboard Name:** `Cognitive Efficiency Audits`
* **Chart Title:** `Cognitive Audit Log Feed`
* **Chart Type:** `Data Table`
* **Columns:** `audit_time`, `model_id`, `avg_latency`, `status`, `audit_critique`

