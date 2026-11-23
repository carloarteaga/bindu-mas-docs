# Marine Fishery Solution Tutorial: Resilient Edge REST Reactor 🚢🌊🐟

This tutorial guides you through configuring, validating, and deploying the **REST Call Reactor (`RESTReactorSet`)** on an industrial edge fleet operating in high-seas environments with intermittent satellite connectivity (e.g., **TASA industrial fishing fleet in the South Pacific**).

You will learn how to:
1. Eliminate external Python polling scripts by deploying declarative, native HTTP fetchers inside Bindu.
2. Leverage **Triggering Dualism** (proactive scheduled `@cron` oceanographic queries and reactive `HaulRegistered` quota validations).
3. Hydrate state without incoming payloads using **Zero-RAG SQLite queries** against `v_messages`.
4. Configure an **Exponential Backoff & Jitter Retry Policy** to absorb satellite micro-outages automatically.
5. Map raw external JSON responses directly into validated Bindu `MessageType` schemas using **Google CEL**.

---

## 1. Real-World Scenario: TASA Altamar Fleet Operations 🗺️

Industrial fishing vessels (such as purse seiners and trawlers) operate hundreds of nautical miles offshore. Fleet operations depend on two critical external data streams:

1. **Oceanographic & Sea Surface Temperature (SST) Data (Proactive):** Feeding real-time water temperature, salinity, and chlorophyll levels into onboard acoustic fish-finding models.
2. **Fisheries Regulatory Quota Compliance (Reactive):** When a fishing haul is brought on deck, the onboard scale registers the catch tonnage and instantly validates remaining biomass quotas with the maritime authority API.

```
                              🛰️ SATELLITE WAN (Intermittent Link)
                                               │
               ┌───────────────────────────────┴───────────────────────────────┐
               ▼                                                               ▼
 ┌───────────────────────────┐                                   ┌───────────────────────────┐
 │   VESSEL: B/P TASA 41     │                                   │   VESSEL: B/P TASA 52     │
 │   Gateway: node-dell-2    │                                   │   Gateway: node-dell-3    │
 ├───────────────────────────┤                                   ├───────────────────────────┤
 │ ⏰ Cron (@every 10m)      │                                   │ ⏰ Cron (@every 10m)      │
 │ 📍 Hydrate Last GPS       │                                   │ 📍 Hydrate Last GPS       │
 │ 🌐 Query Marine SST API   │                                   │ 🌐 Query Marine SST API   │
 │ 🛡️ Exponential Backoff    │                                   │ 🛡️ Exponential Backoff    │
 │ 📤 Emit RawMarineData     │                                   │ 📤 Emit RawMarineData     │
 └─────────────┬─────────────┘                                   └─────────────┬─────────────┘
               │                                                               │
               └───────────────────────────────┬───────────────────────────────┘
                                               ▼
                                 ┌───────────────────────────┐
                                 │    FLEET HQ (CALLAO)      │
                                 │        Root Node          │
                                 ├───────────────────────────┤
                                 │ 📊 Central Live Mesh Bus  │
                                 │ 🗺️ Real-time Fleet Map    │
                                 └───────────────────────────┘
```

---

## 2. Step 1: Deploy Required Message Types 📦

Before configuring the reactors, register the custom schemas required for vessel positioning, raw oceanographic telemetry, haul registration, and quota validation.

Create `poc_message_types.json` and import it into the mesh:

```bash
bindu-studio message-types import \
  --in-path ./public/console/docs/rest-reactor-tutorial/poc_message_types.json \
  --auth console:SuperSecret1 --all --yes
```

### Registered Message Types Overview:

| Message Type | Category | Purpose |
| :--- | :--- | :--- |
| **`VesselGPSPosition`** | Sensor Ingress | Periodic GPS coordinates, speed, and heading emitted by onboard NMEA GPS receiver. |
| **`HaulRegistered`** | Sensor Ingress | Event emitted when deck scale confirms catch tonnage for a net haul. |
| **`RawMarineData`** | Reactor Egress | Structured oceanographic parameters (SST, chlorophyll, wave height) fetched from external marine API. |
| **`HaulQuotaValidation`**| Reactor Egress | Validated regulatory biomass quota status returned from Ministry API. |
| **`FleetOceanAssessment`**| AI / Mesh Egress | Aggregate fleet insight generated from fused oceanographic telemetry. |

---

## 3. Step 2: Deep-Dive into the `RESTReactorSet` Configuration 🧬

The configuration below defines two independent REST reactor instances deployed to gateway node **`node-dell-2`**:

```json
{
  "$type": "RESTReactorSet",
  "$node-dest": "node-dell-2",
  "$group": "mesh",
  "$producer": "root",
  "$user": "admin",
  "$when": "2026-09-09T12:00:00Z",
  "fields": {
    "gateway_node_id": "node-dell-2",
    "reactors": [
      {
        "reactor_id": "tasa41-oceanographic-weather-poller",
        "input_message_type": "",
        "cron": "@every 10m",
        "context_query": "SELECT json_extract(raw_json, '$.fields.latitude') AS lat, json_extract(raw_json, '$.fields.longitude') AS lon, json_extract(raw_json, '$.fields.vessel_id') AS vessel_id FROM v_messages WHERE \"$type\" = 'VesselGPSPosition' ORDER BY \"$when\" DESC LIMIT 1",
        "output_message_type": "RawMarineData",
        "request": {
          "method": "GET",
          "url": "'https://api.marine-copernicus.io/v1/ocean-conditions?lat=' + string(db.rows[0].fields.lat) + '&lon=' + string(db.rows[0].fields.lon) + '&vessel=' + string(db.rows[0].fields.vessel_id)",
          "headers": {
            "Accept": "'application/json'",
            "Authorization": "'Bearer ' + secret('COPERNICUS_SAT_TOKEN', 'copernicus_sat_token_tasa_edge_2026')",
            "X-Client-Node": "head['$producer']"
          },
          "timeout": "8s"
        },
        "retry_policy": {
          "max_retries": 4,
          "initial_interval": "1s",
          "backoff_multiplier": 2.0,
          "max_interval": "15s",
          "jitter": true,
          "retryable_statuses": [408, 429, 500, 502, 503, 504]
        },
        "response_map": {
          "vessel_id": "has(db.rows) && size(db.rows) > 0 ? string(db.rows[0].fields.vessel_id) : 'TASA-41'",
          "latitude": "has(db.rows) && size(db.rows) > 0 ? double(db.rows[0].fields.lat) : 0.0",
          "longitude": "has(db.rows) && size(db.rows) > 0 ? double(db.rows[0].fields.lon) : 0.0",
          "sea_surface_temp_c": "double(resp.body.measurements.sst_celsius)",
          "chlorophyll_mg_m3": "double(resp.body.measurements.chlorophyll_a)",
          "wave_height_meters": "double(resp.body.measurements.significant_wave_height)",
          "ocean_current_knots": "double(resp.body.measurements.current_speed_knots)",
          "thermocline_depth_m": "int(resp.body.measurements.thermocline_depth_meters)",
          "fetch_status": "resp.status_code == 200 ? 'SUCCESS_NOMINAL' : 'DEGRADED_FETCH'"
        }
      },
      {
        "reactor_id": "tasa41-haul-quota-validator",
        "input_message_type": "HaulRegistered",
        "cron": "",
        "trigger_condition": "double(payload.catch_tonnage) > 0.0",
        "context_query": "SELECT json_extract(raw_json, '$.fields.zone_id') AS last_zone FROM v_messages WHERE \"$type\" = 'VesselGPSPosition' ORDER BY \"$when\" DESC LIMIT 1",
        "output_message_type": "HaulQuotaValidation",
        "request": {
          "method": "POST",
          "url": "'https://api.maritime-authority.gob.pe/v2/quotas/validate'",
          "headers": {
            "Content-Type": "'application/json'",
            "Authorization": "'Bearer ' + secret('VESSEL_AUTHORITY_TOKEN')",
            "X-Vessel-Key": "secret('VESSEL_KEY')",
            "X-Client-Node": "head['$producer']"
          },
          "body": "{ 'vessel_id': string(head['$producer']), 'species': string(payload.species), 'declared_metric_tons': double(payload.catch_tonnage), 'zone': has(db.rows) && size(db.rows) > 0 ? string(db.rows[0].fields.last_zone) : 'ZONE_CENTRAL_PACIFIC' }",
          "timeout": "10s"
        },
        "retry_policy": {
          "max_retries": 3,
          "initial_interval": "500ms",
          "backoff_multiplier": 2.0,
          "max_interval": "10s",
          "jitter": true,
          "retryable_statuses": [408, 429, 500, 502, 503, 504]
        },
        "response_map": {
          "vessel_id": "string(head['$producer'])",
          "haul_id": "string(payload.haul_id)",
          "quota_authorized": "bool(resp.body.authorized)",
          "remaining_season_tons": "double(resp.body.remaining_fleet_quota_tons)",
          "compliance_code": "string(resp.body.compliance_status_code)",
          "rejection_reason": "has(resp.body.rejection_reason) ? string(resp.body.rejection_reason) : 'N/A'"
        }
      }
    ]
  }
}
```

---

## 4. Deconstructing the Reactor Engineering 🔍

### A. Context Hydration via SQLite (Zero-RAG)
In the oceanographic poller (`tasa41-oceanographic-weather-poller`), execution is triggered by `@every 10m` cron timer. Because there is no incoming message, the reactor invokes the `context_query`:
```sql
SELECT 
  json_extract(raw_json, '$.fields.latitude') AS lat, 
  json_extract(raw_json, '$.fields.longitude') AS lon, 
  json_extract(raw_json, '$.fields.vessel_id') AS vessel_id 
FROM v_messages 
WHERE "$type" = 'VesselGPSPosition' 
ORDER BY "$when" DESC LIMIT 1;
```
This query extracts the vessel's current GPS position from the last recorded NMEA telemetry message. The fields become accessible in CEL expressions as `db.rows[0].fields.lat` and `db.rows[0].fields.lon`.

### B. Dynamic Request Construction with CEL
The endpoint URL is constructed on the fly:
```javascript
'https://api.marine-copernicus.io/v1/ocean-conditions?lat=' + string(db.rows[0].fields.lat) + '&lon=' + string(db.rows[0].fields.lon) + '&vessel=' + string(db.rows[0].fields.vessel_id)
```
No hardcoded coordinates, no external string interpolation scripts.

### C. Satellite-Tolerant Exponential Backoff Retry
When operating in remote sea zones, satellite terminals (Starlink Maritime / Inmarsat FleetBroadband) experience transient attenuation and beam switches.

The retry configuration:
```json
"retry_policy": {
  "max_retries": 4,
  "initial_interval": "1s",
  "backoff_multiplier": 2.0,
  "max_interval": "15s",
  "jitter": true,
  "retryable_statuses": [408, 429, 500, 502, 503, 504]
}
```
- **Attempt 1:** Fails due to satellite drop $\rightarrow$ Waits $\approx 1.0\text{s} \pm 15\%$ jitter.
- **Attempt 2:** Fails due to HTTP 503 upstream $\rightarrow$ Waits $\approx 2.0\text{s} \pm 15\%$ jitter.
- **Attempt 3:** Upstream responds 200 OK $\rightarrow$ Continues to response mapping.
- The entire retry cycle is handled asynchronously in the `bindu-rest-reactor` child process, protecting the core event bus from blocking.

### D. Response Mapping & Type Coercion
The external API returns a third-party JSON object. The `response_map` compiles CEL expressions to extract and cast fields directly into the `RawMarineData` schema:
```javascript
"sea_surface_temp_c": "double(resp.body.measurements.sst_celsius)",
"chlorophyll_mg_m3": "double(resp.body.measurements.chlorophyll_a)",
"thermocline_depth_m": "int(resp.body.measurements.thermocline_depth_meters)"
```

---

## 5. Step 3: Offline Validation & Deployment 🚀

### Step 0: Provision Centralized Vessel Secrets in Vault (Zero-Plaintext Wire Guarantee)
Under Bindu's Multi-Agent System (MAS) architecture, secrets like maritime API tokens and vessel auth keys must **never** be hardcoded in plaintext within distributed deployment manifests or JSON configuration files.

Instead, Bindu employs the **Zero-Plaintext Wire Guarantee**:
1. Cloud/Authority credentials are encrypted locally in the node's AES-GCM vault (`accessControlConfig.brs`) using the `bindu-studio access-control secrets` CLI.
2. In `RESTReactorSet` configurations, headers dynamically resolve keys at execution time via Google CEL: `secret('VESSEL_AUTHORITY_TOKEN')` and `secret('VESSEL_KEY')`.
3. The running REST reactor worker dynamically extracts and injects the decrypted credentials in volatile memory without leaking secrets over the wire or in SQLite ledgers (`v_messages`).

```bash
# Provision vessel authority credentials into encrypted vault
go run ./cmd/bindu-studio access-control secrets set VESSEL_AUTHORITY_TOKEN \
  --value "YOUR_ACTUAL_VESSEL_AUTHORITY_TOKEN" \
  --yes

go run ./cmd/bindu-studio access-control secrets set VESSEL_KEY \
  --value "VESSEL-AUTH-NUNEGAL77-ATLANTIC" \
  --yes

# Verify vault status
go run ./cmd/bindu-studio access-control secrets validate
```

---

### A. Offline Validation via CLI
Validate the syntax, CEL compilation, and retry bounds before touching any active hardware:

```bash
bindu-studio rest-reactor validate \
  --in-path ./public/console/docs/rest-reactor-tutorial/poc_rest_reactor_config.json
```

**Expected Output:**
```
✓ Validating RESTReactorSet JSON structure...
✓ Gateway Node: node-dell-2 (2 reactors configured)
  [1] Reactor ID: tasa41-oceanographic-weather-poller
      • Cron: @every 10m (valid syntax)
      • Context Query: OK (valid SQL syntax)
      • Request: GET (URL CEL AST compiled successfully)
      • Headers: 3 CEL expressions compiled
      • Retry Policy: MaxRetries=4, Initial=1s, Multiplier=2.0 (valid bounds)
      • Response Map: 9 properties compiled & verified against RawMarineData
  [2] Reactor ID: tasa41-haul-quota-validator
      • Ingress Trigger: HaulRegistered
      • Trigger Guard: OK (CEL boolean expression compiled)
      • Request: POST (URL CEL AST compiled, Body JSON compiled)
      • Retry Policy: MaxRetries=3, Initial=500ms, Multiplier=2.0 (valid bounds)
      • Response Map: 6 properties compiled & verified against HaulQuotaValidation
✅ All 2 REST Reactor configurations validated successfully (0 errors, 0 warnings).
```

### B. Deploying to the Active Mesh (`merge`)
Merge and deploy the configuration to the vessel gateway node via MQTT:

```bash
go run ./cmd/bindu-studio rest-reactor merge \
  --file ./public/console/docs/rest-reactor-tutorial/poc_rest_reactor_config.json \
  --yes \
  --auth console:StrongPass123
```

---

## 6. Step 4: Verification & Live Observability 📊

### A. Inject Real-Time GPS Voyage Telemetry (Simulation Script)
Replay the real-world navigation trajectory of fishing vessel **`NUNEGAL-77`** from the Port of A Coruña, past the Tower of Hercules, into the Atlantic thermal fishing zone across 8 waypoints:

```bash
./public/console/docs/rest-reactor-tutorial/simulate_vessel_gps.sh \
  --node node-dell-2 \
  --vessel NUNEGAL-77 \
  --interval 2
```

**Simulation Waypoints Replayed:**
1. `(43.3715, -8.3960)`: Saliendo del Dique de Abrigo (Puerto de A Coruña)
2. `(43.3850, -8.4065)`: Navegando frente a la Torre de Hércules
3. `(43.4002, -8.4201)`: Saliendo a mar abierto, rumbo Noroeste
4. `(43.4155, -8.4350)`: Aumentando velocidad hacia la zona térmica
5. `(43.4310, -8.4485)`: Entrando a la zona de pesca predicha (Óptima)
6. `(43.4400, -8.4600)`: Maniobra de pesca activa
7. `(43.4350, -8.4750)`: El barco empieza a desviarse ligeramente hacia el oeste
8. `(43.4280, -8.4820)`: Desvío crítico (Alerta de rumbo CEL Reactor)

### B. Trigger Haul Registration (Reactive)
Simulate a net haul confirmation:

```bash
go run ./public/console/docs/rest-reactor-tutorial/publish_mqtt.go \
  -host 127.0.0.1 -port 1883 -user console -pass StrongPass123 \
  -topic "bindu/in" \
  -msg '{"$type":"HaulRegistered","$group":"mesh","$user":"admin","$producer":"NUNEGAL-77","$when":"'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"/galicia/coruna/mar-artabro"},"fields":{"haul_id":"HAUL-2026-09-001","species":"Merluza (Merluccius merluccius)","catch_tonnage":14.2,"deck_temp_c":15.4}}'
```

### C. Import & Launch Operational Dashboards 📊
Import the pre-configured tutorial dashboards into the Bindu mesh:

```bash
# Validate offline
bindu-studio dashboard validate \
  --file ./public/console/docs/rest-reactor-tutorial/poc_dashboards.json

# Import into mesh
bindu-studio dashboard import \
  --file ./public/console/docs/rest-reactor-tutorial/poc_dashboards.json \
  --auth console:SuperSecret1 --all
```

**Included Dashboards:**
1. **🌊 NUNEGAL-77 Ocean Telemetry & SST Trends** (`line`): Live trends of Sea Surface Temperature, Wave Height, and Ocean Currents.
2. **🛰️ REST Reactor Zero-RAG Ingestion Audit** (`table`): Comprehensive trace joining GPS telemetry, real-time Open-Meteo REST egress, and Maritime regulatory quota verification.
3. **⚖️ Catch Biomass vs Fleet Quota Compliance** (`bar`): Haul catch tonnage vs remaining fleet quotas.
4. **🧭 Vessel Navigation & Waypoint Log** (`table`): Live GPS fixes, speed, and heading telemetry.

### D. Verify Ledger Egress in SQLite (`v_messages`)
Open the Console **Queries** tab or run via CLI:

```sql
SELECT 
  datetime("$when") AS timestamp,
  "$type" AS event_type,
  json_extract(raw_json, '$.fields.vessel_id') AS vessel,
  json_extract(raw_json, '$.fields.latitude') AS lat,
  json_extract(raw_json, '$.fields.longitude') AS lon,
  json_extract(raw_json, '$.fields.sea_surface_temp_c') AS sst_c,
  json_extract(raw_json, '$.fields.chlorophyll_mg_m3') AS chlorophyll,
  json_extract(raw_json, '$.fields.fetch_status') AS status
FROM v_messages 
WHERE "$type" = 'RawMarineData'
ORDER BY datetime("$when") DESC LIMIT 8;
```

```sql
SELECT 
  datetime("$when") AS timestamp,
  json_extract(raw_json, '$.fields.haul_id') AS haul,
  json_extract(raw_json, '$.fields.quota_authorized') AS authorized,
  json_extract(raw_json, '$.fields.remaining_season_tons') AS remaining_tons,
  json_extract(raw_json, '$.fields.compliance_code') AS code
FROM v_messages 
WHERE "$type" = 'HaulQuotaValidation'
ORDER BY datetime("$when") DESC LIMIT 5;
```

---

## 7. Summary & Benefits of Native REST Reactor

1. **Zero External Scripts:** Eliminates fragile cron scripts, custom python pollers, and container sprawl on edge nodes.
2. **Deterministic Edge Resilience:** Automatic exponential backoff + jitter absorbs satellite outages without manual watchdog logic.
3. **Strict Schema Governance:** Output is always validated against Bindu `MessageType` definitions before being admitted onto the mesh.
4. **Zero-RAG Data Hydration:** Combines local SQLite historical facts with external API intelligence seamlessly.
5. **Real-time Observability:** Built-in SQL Dashboards visualize external REST data feeds and edge decisions instantly.

---

*AUM 🙏🏾❤️ — Dedicated to the fishermen and engineers of maritime edge intelligence.*

