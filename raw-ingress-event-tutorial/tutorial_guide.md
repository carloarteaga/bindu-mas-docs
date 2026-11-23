# 📡 Heterogeneous Raw MQTT Ingress & Dynamic CEL Normalization Tutorial

## 📋 Executive Summary & Problem Space

In modern Internet of Things (IoT), Industrial Automation (IIoT), and Supply Chain Tracking, **MQTT** is the universal protocol for telemetry transport. However, hardware manufacturers (Zebra, Hopeland, Chafon, Chainway, Impinj, and PLC vendors) each structure their MQTT payloads differently:
- Some publish nested JSON arrays of tag readings (Hopeland HH340/380).
- Others publish single flat JSON objects with proprietary field names (Zebra IoT Connector with `idHex` and `peakRssi`).
- Microcontrollers and industrial PLCs often publish compact **raw binary frames** (e.g., Marktrace MR6034E 18-byte frames or Modbus byte sequences) to conserve bandwidth.

### The Traditional Dilemma: Daemon Sprawl vs Storage Bloat

Traditional edge software architectures handle this diversity in two flawed ways:
1. **The Daemon Sprawl Anti-Pattern**: Writing and maintaining separate Go/Python/Node.js gateway services for each hardware model. This creates massive operational overhead, container bloat, and deployment friction on resource-constrained edge gateways.
2. **The Database WAL Bloat Anti-Pattern**: Storing untyped messages directly into SQLite. Persisting high-frequency raw reads (thousands of unparsed JSON/binary packets per second) saturates the disk write-ahead log (WAL), explodes database file size, and floods mesh networks when amateur solutions broadcast raw streams to parent and child nodes.

---

## 🏛️ The Bindu Ingress Architecture: Ephemeral Memory Routing + CEL Kinetics

Bindu eliminates both anti-patterns through an **Event-Driven Horn Clause Deductive Engine**:

```
 [ Hardware / Sensors ]              [ Bindu Core Dispatcher ]                  [ RAM Heap: CEL Reactors ]                 [ WAL Ledger: v_messages ]
 ──────────────────────              ─────────────────────────                  ─────────────────────────                 ─────────────────────────
  Hopeland (JSON Array)     ───┐
  Zebra (JSON 'SIMPLE')     ───┼──>  bindu/in/raw/<subpath>  ──>  RawIngressEvent  ──>  Horn Clause Deductions  ──> Output ──>  INSERT INTO v_messages
  Chafon (JSON Object)      ───┤                                  • persist: false       P1 ∧ P2 ∧ ... → Q                      • Type: RawTagRead
  Marktrace (Binary Bytes)  ───┘                                  • routing: local       Sub-microsecond Go                     • Retention: Count 50
                                                                  • Category: system     Zero disk allocations                  • Zero unparsed bloat!
```

### Architectural Guarantees:
1. **Zero-Plaintext Untyped Ingress**: Any sensor or reader can publish to `bindu/in/raw/<subpath>` with TLS and ACL authentication. No code changes required on the reader.
2. **Ephemeral Memory Boxing (`RawIngressEvent`)**: The dispatcher automatically packages the raw payload into a built-in `RawIngressEvent`. Crucially:
   - `persist: false`: The event exists **strictly in Go RAM memory** and is **never written to disk**.
   - `routing: ["local"]`: It is confined to the local node heap, **never flooding parent or child nodes**.
3. **Formal Horn Clause Transformations ($P_1 \land P_2 \land \dots \rightarrow Q$)**: Declarative Google CEL reactors intercept the in-memory `RawIngressEvent`, validate the topic subpath and payload structure, and deduce the normalized business entity `RawTagRead`.
4. **Binary Protocol Unpacking in Memory**: Bindu's CEL binary extensions (`hex_decode`, `bytes_slice`, `uint8`, `hex_encode`) decode raw byte frames into structured fields in under 15 microseconds, eliminating external C/Go decoders.
5. **Clean SQLite Ledger**: Only the resulting, strongly-typed `RawTagRead` messages are recorded into SQLite with deterministic retention policies (`retentionMode: "count"`, `retentionValue: 50`).

---

## 🏭 Supported Hardware Profiles in this Tutorial

| Profile | Payload Format | Incoming Topic | Characteristic Fields | CEL Normalization Technique |
|---|---|---|---|---|
| **Hopeland HH340/380** | JSON Array | `bindu/in/raw/rfid/hopeland/dock_door_1` | `device_id`, `antenna`, `tags: [{epc, rssi, read_count}]` | Top-level `payload.antenna`, unpacks `tags[0].epc` and `double(tags[0].rssi)` |
| **Zebra FX9600** | JSON (IoT Connector) | `bindu/in/raw/rfid/zebra/dock_north` | `type: "SIMPLE"`, `data.idHex`, `data.antenna`, `data.peakRssi` | Maps `data.idHex` to `epc`, `data.peakRssi` to `rssi`, `data.reads` to `read_count` |
| **Chafon CF6D2 / Chainway** | JSON Object | `bindu/in/raw/rfid/chafon/gate_01` | `device_id`, `antenna`, `tag: {epc, rssi}` | Maps `tag.epc` to `epc`, `tag.rssi` to `rssi`, assigns default `read_count: 1` |
| **Marktrace MR6034E** | Binary Frame (18B) | `bindu/in/raw/rfid/marktrace/portal_west` | `0xBB 0x02` header, Byte 4: Antenna, Bytes 5..16: EPC, Byte 17: RSSI | Uses `hex_decode(payload.raw_data)`, `uint8(..., 4)`, `hex_encode(bytes_slice(..., 5, 17))` |

---

## 🔬 Declarative Artifacts Walkthrough

The tutorial solution is composed of the following declarative JSON configurations in `public/console/docs/raw-ingress-event-tutorial/`:

### 1. `poc_message_types.json` — The Business Schemas
Defines two semantic message types:
- **`RawTagRead`**: The unified, normalized tag reading event emitted by all CEL normalizers.
  - Fields: `epc` (string), `antenna` (integer), `rssi` (number), `read_count` (integer), `reader_vendor` (string), `reader_id` (string), `dock_portal` (string).
  - Retention: `count: 50` (sliding real-time window, zero storage leak).
  - Routing: `["local"]`.
- **`ReaderHealthAlert`**: Incident notification generated when antenna RF signal drops below operational threshold (< -75 dBm).

### 2. `poc_cel_reactor_config.json` — The 5 CEL Reactors
- **`hopeland_rfid_normalizer`**: Unpacks Hopeland JSON array events.
- **`zebra_rfid_normalizer`**: Unpacks Zebra IoT Connector `SIMPLE` mode events.
- **`chafon_rfid_normalizer`**: Unpacks Chafon single tag objects.
- **`marktrace_binary_rfid_normalizer`**: Unpacks 18-byte raw binary frames using CEL binary extensions:
  ```cel
  // Trigger condition: verify subpath, binary content type, and frame size
  payload.subpath.startsWith('rfid/marktrace') && payload.content_type == 'binary' && size(hex_decode(payload.raw_data)) >= 18

  // Transformations:
  epc:           hex_encode(bytes_slice(hex_decode(payload.raw_data), 5, 17))
  antenna:       int(uint8(hex_decode(payload.raw_data), 4))
  rssi:          double(uint8(hex_decode(payload.raw_data), 17)) - 256.0
  read_count:    1
  reader_vendor: 'Marktrace'
  reader_id:     'MR6034E-01'
  dock_portal:   string(payload.subpath)
  ```
- **`weak_signal_health_guard`**: Intercepts `RawTagRead` with `rssi < -75.0 dBm` and synthesizes `ReaderHealthAlert`.

### 3. `poc_dashboards.json` — Real-Time Visualizations
- **`raw-ingress-telemetry-monitor`**: Live table of all normalized RFID tag events across all vendors.
- **`reader-health-incident-monitor`**: Operational board tracking weak signal alerts and marginal antenna connections.

### 4. `raw_ingress_mas_solution.json` — Level 0 Solution Manifest
The declarative solution pack tying schemas, reactors, dashboards, and mesh nodes into an auditable artifact.

---

## 🚀 Step-by-Step Hands-On Deployment Guide

### Step 1: Preflight Verification & Dry-Run Certification

Before deploying any logic to the cluster, Bindu requires executing preflight dry-runs:

```bash
# 1. Validate CEL Reactor syntax against the tutorial schemas:
go run ./cmd/bindu-studio cel-reactor validate \
    --message-types public/console/docs/raw-ingress-event-tutorial/poc_message_types.json \
    --file public/console/docs/raw-ingress-event-tutorial/poc_cel_reactor_config.json

# 2. Simulate Hopeland JSON normalizer in-process:
go run ./cmd/bindu-studio cel-reactor dry-run \
    --message-types public/console/docs/raw-ingress-event-tutorial/poc_message_types.json \
    --file public/console/docs/raw-ingress-event-tutorial/poc_cel_reactor_config.json \
    --reactor-id hopeland_rfid_normalizer \
    --payload '{"subpath": "rfid/hopeland/hh340_01", "content_type": "json", "payload": {"device_id": "HH340-A01", "antenna": 2, "tags": [{"epc": "E280110C2000000213456789", "rssi": -45.5, "read_count": 12}]}}'

# 3. Simulate Marktrace Binary Frame normalizer (CEL Binary Extensions):
go run ./cmd/bindu-studio cel-reactor dry-run \
    --message-types public/console/docs/raw-ingress-event-tutorial/poc_message_types.json \
    --file public/console/docs/raw-ingress-event-tutorial/poc_cel_reactor_config.json \
    --reactor-id marktrace_binary_rfid_normalizer \
    --payload '{"subpath": "rfid/marktrace/portal_west", "content_type": "binary", "raw_data": "bb02220002e280116060000204b768820ac9"}'

# 4. Validate Dashboard definitions:
go run ./cmd/bindu-studio dashboard validate \
    --file public/console/docs/raw-ingress-event-tutorial/poc_dashboards.json
```

### Step 2: Deploy Solution to the Cluster

Execute the universal deployment script:

```bash
./public/console/docs/raw-ingress-event-tutorial/deploy_mas_solution.sh --yes
```

This automates the 5 deployment phases:
1. Merges `RawTagRead` and `ReaderHealthAlert` into the live message catalog.
2. Executes preflight validation gates.
3. Merges the 5 CEL reactors and 2 dashboards via MQTT.
4. Exports, validates, and applies the MAS solution bundle.
5. Executes the runtime OTA triptych (`baseline accept` -> `desired commit` -> `update wizard`).

---

## 📡 Running the Multi-Reader Simulator

The tutorial includes a high-performance Go simulator (`simulate_raw_readers.go`) capable of streaming realistic traffic from all 4 reader profiles.

### Mode A: Simulate All Readers Simultaneously (Interleaved)
```bash
./public/console/docs/raw-ingress-event-tutorial/simulate_raw_readers.sh all 20 500ms
```
*Emits 20 readings, rotating through Hopeland, Zebra, Chafon, and Marktrace binary frames every 500ms.*

### Mode B: Simulate a Specific Hardware Vendor
```bash
# Hopeland HH340 only:
./public/console/docs/raw-ingress-event-tutorial/simulate_raw_readers.sh hopeland 10 300ms

# Zebra FX9600 IoT Connector only:
./public/console/docs/raw-ingress-event-tutorial/simulate_raw_readers.sh zebra 10 300ms

# Chafon CF6D2 only:
./public/console/docs/raw-ingress-event-tutorial/simulate_raw_readers.sh chafon 10 300ms

# Marktrace MR6034E Raw Binary Frames only:
./public/console/docs/raw-ingress-event-tutorial/simulate_raw_readers.sh marktrace 10 300ms
```

### Mode C: Continuous Simulation
Pass `0` as the count to stream continuously until interrupted with `Ctrl+C`:
```bash
./public/console/docs/raw-ingress-event-tutorial/simulate_raw_readers.sh all 0 250ms
```

---

## 📊 Verifying the Invariant: Zero SQLite Bloat

To prove that untyped traffic is never stored to SQLite while normalized tag events are recorded, query the running engine via the native REST endpoint:

```bash
# 1. Query for RawIngressEvent (Must return exactly 0 rows):
curl -s -X POST http://127.0.0.1:8080/api/v1/messages/sql \
  -u console:SuperSecret1 \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT COUNT(*) AS total_raw FROM v_messages WHERE \"$type\" = '\''RawIngressEvent'\''"}'

# 2. Query for Normalized RawTagRead events (Hopeland, Zebra, Chafon, Marktrace):
curl -s -X POST http://127.0.0.1:8080/api/v1/messages/sql \
  -u console:SuperSecret1 \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT \"$when\", json_extract(raw_json, '\''$.fields.reader_vendor'\'') AS vendor, json_extract(raw_json, '\''$.fields.epc'\'') AS epc, CAST(json_extract(raw_json, '\''$.fields.antenna'\'') AS INTEGER) AS antenna, CAST(json_extract(raw_json, '\''$.fields.rssi'\'') AS REAL) AS rssi FROM v_messages WHERE \"$type\" = '\''RawTagRead'\'' ORDER BY \"$when\" DESC LIMIT 10"}'

# 3. Query for ReaderHealthAlert events (Weak signal warnings):
curl -s -X POST http://127.0.0.1:8080/api/v1/messages/sql \
  -u console:SuperSecret1 \
  -H "Content-Type: application/json" \
  -d '{"query": "SELECT \"$when\", json_extract(raw_json, '\''$.fields.alert_code'\'') AS alert, json_extract(raw_json, '\''$.fields.details'\'') AS details FROM v_messages WHERE \"$type\" = '\''ReaderHealthAlert'\'' ORDER BY \"$when\" DESC LIMIT 5"}'
```

---

## 🛡️ Best Practices & Platform Guardrails

1. **Always Use `persist: false` for Raw Ingress**: Untyped ingress events are transient transformation vectors. Never set `persist: true` on `RawIngressEvent`.
2. **Strictly Confine Routing to `["local"]`**: Telemetry and raw ingress streams must never route to `"child"`. Adding `"child"` to raw ingress causes mesh flooding and infinite rejection error loops.
3. **Use Count-Based Retention on Normalized Telemetry**: Normalized telemetry streams (`RawTagRead`) must use `retentionMode: "count"` (20–100 items), never calendar days (`days: 14` or `days: 30`) which bloat edge storage.
4. **Leverage Binary CEL Extensions for Modbus & Serial Protocols**: When interfacing with PLCs, RS-485 to MQTT bridges, or microcontrollers, use `hex_decode`, `bytes_slice`, `uint8`, and `uint16_be` directly in CEL rather than writing external microservices.
