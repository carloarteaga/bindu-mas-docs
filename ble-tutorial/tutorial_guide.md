# 📡 Bluetooth Low Energy (BLE) Spatial Presence & Reference Plane Digital Twin Tutorial

## 📋 1. Executive Summary & Industrial Scope

In modern smart hospitals, automated warehouses, cleanrooms, and industrial shop floors, **Bluetooth Low Energy (BLE 4.2 / 5.x)** is the dominant radio frequency standard for real-time asset tracking and personnel safety. Battery-operated BLE tags and badges emit omnidirectional advertising bursts in the 2.4 GHz ISM band using standard advertising specifications such as **Apple iBeacon**, **Eddystone (UID/TLM/URL)**, or proprietary telemetry packets.

However, transforming raw 2.4 GHz radio frequency advertising packets into reliable digital twin location insights presents significant engineering challenges:
1. **Multipath Fading & RSSI Noise**: Radio signal strength indication (RSSI) fluctuates drastically due to human bodies, metal machinery, reflections, and antenna orientation. Raw RSSI cannot be used directly without statistical filtering.
2. **Boundary Chattering & False Transitions**: A tag located near the physical entrance of an operating room will rapidly flicker between "present" and "absent" without state debouncing.
3. **Decoupled Spatial Context**: Most IoT systems emit isolated payloads (e.g. `{"mac": "AA:BB:CC", "rssi": -72}`) without binding the observation to the building's physical coordinate system.
4. **Heavy Engine Intrusion**: Legacy systems embed radio drivers and custom business logic directly into the central server, compromising platform stability.

**Bindu** solves these challenges cleanly within its Multi-Agent System (MAS) architecture:
- **Clean Isolation via BPR Plugins (Level 1)**: The edge scanner runs as an external, supervised OS process (`bindu-ble-presence-plugin`) interfacing directly with the Linux Bluetooth controller (`hci0`) via `tinygo.org/x/bluetooth`, communicating with the Bindu node strictly over `stdin/stdout` JSON envelopes.
- **Hierarchical Reference Plane ($where) (Level 0)**: Bindu's spatial model defines an `operation-room` reference node anchored under the physical gateway node (`node-dell-2`), establishing a branch-local Cartesian coordinate plane ($12\,\text{m} \times 8\,\text{m}$).
- **State Machine with Dwell & Absence Filters**: A deterministic state engine enforces a **1000 ms dwell window** before declaring a beacon `SEEN` and a **5000 ms absence timeout** before declaring it `LOST`.
- **Pure SQL Zero-RAG Observability (Level 2)**: Real-time Dashboards query the immutable `v_messages` event stream directly using standard SQL, rendering both a tabular transition audit log and an interactive 2D operational map without external databases.

---

## 🏛️ 2. Physical & Spatial Architecture

The physical setup consists of an edge gateway scanner (`node-dell-2`) monitoring an enclosed medical surgical suite (**Operation Room 101**):

```
                        Operation Room 101 (Width: 12.0m, Height: 8.0m)
     (0, 8.0) ┌────────────────────────────────────────────────────────┐ (12.0, 8.0)
              │                                                        │
              │                                                        │
              │                     [ 🩺 Mobile Ultrasound Twin ]      │
              │                       Beacon: fda50693-...             │
              │                       Target Coords: (6.0m, 4.0m)      │
              │                       Status: SEEN                     │
              │                                                        │
              │   📡 Edge Scanner Gateway                              │
              │      node-dell-2 (Local Plane Root)                    │
              │      Position: (1.0m, 1.0m)                            │
              │                                           🚪 Doorway   │
     (0, 0.0) └───────────────────────────────────────────════════════┘ (12.0, 0.0)
```

### Reference Node Hierarchy & The `$where` Contract

Bindu models spatial structures as a declarative directed acyclic graph (DAG) in `mesh.json`:

1. **Gateway Node (`node-dell-2`)**: The physical hardware machine equipped with a Bluetooth interface. It acts as the **Plane Root** with a local Cartesian coordinate system (origin: $x=0, y=0$, unit: meters).
2. **Reference Node (`operation-room`)**: A logical spatial entity whose parent is `node-dell-2`. It is assigned a polygon boundary:
   $$\text{Coordinates} = [[0, 0], [12, 0], [12, 8], [0, 8], [0, 0]]$$
3. **Semantic Ingress Stamping**: Every message emitted by the plugin (`BLEBeaconSeen` and `BLEBeaconLost`) is stamped with:
   - `zone_id`: `"operation-room"`
   - `where_locagram`: `"node-dell-2/operation-room"`
   - `coordinates_x`: $6.0$
   - `coordinates_y`: $4.0$
   - `zone_center_x`: $6.0$, `zone_center_y`: $4.0$

This allows downstream consumers, CEL reactors, and Dashboards to project the tag's digital twin directly onto the room geometry.

---

## 🔬 3. Radio Frequency & State Machine Mathematics

### 3.1 Log-Distance Path Loss Model
The plugin estimates real-time distance from the edge scanner using the Log-Distance Path Loss formula:

$$d = 10^{\frac{P_{\text{ref}} - \text{RSSI}_{\text{filtered}}}{10 \cdot n}}$$

Where:
- $P_{\text{ref}}$: Measured RSSI at 1 meter (calibrated iBeacon TX Power byte, typically $-59\,\text{dBm}$).
- $\text{RSSI}_{\text{filtered}}$: Exponentially smoothed signal strength.
- $n$: Path loss exponent ($n \approx 2.0$ for free space, $n \approx 2.5$ for indoor hospital rooms).

### 3.2 Exponential Moving Average (EMA) RSSI Smoothing
To suppress multipath radio spikes, incoming RSSI readings are smoothed across consecutive packets:

$$\overline{\text{RSSI}}_t = \alpha \cdot \text{RSSI}_t + (1 - \alpha) \cdot \overline{\text{RSSI}}_{t-1} \quad (\alpha = 0.35)$$

### 3.3 Presence & Absence Debounce State Machine

```
              ┌────────────────────────────────────────────────────────┐
              │                                                        │
              ▼                                                        │
      ┌───────────────┐  First Packet   ┌───────────────────┐          │ Absence
      │    UNKNOWN    │ ──────────────> │ CANDIDATE_SEEN    │          │ > 5000ms
      │  (No History) │                 │ (Dwell Timer On)  │          │
      └───────────────┘                 └───────────────────┘          │
              ▲                                   │                    │
              │                                   │ Dwell >= 1000ms    │
              │                                   ▼                    │
              │  Absence Timeout        ┌───────────────────┐          │
              │  Emits BLEBeaconLost    │      PRESENT      │          │
              └──────────────────────── │   (Tag Active)    │ ─────────┘
                                        └───────────────────┘
```

1. **First Packet Received**: The beacon enters the `CANDIDATE_SEEN` buffer.
2. **Dwell Debounce ($\ge 1000\,\text{ms}$)**: If continuous advertising frames persist for at least $1000\,\text{ms}$, the state transitions to `PRESENT` and the plugin emits a `BLEBeaconSeen` semantic event.
3. **Continuous Tracking**: As long as advertising packets arrive, the beacon's `last_seen` timestamp updates.
4. **Absence Sweeper ($> 5000\,\text{ms}$)**: A background sweeper checks active beacons every second. If no packet is received for $5000\,\text{ms}$, the state transitions to `LOST`, and the plugin emits a `BLEBeaconLost` semantic event.

---

## 📦 4. The 6 Standard Artifacts in this Solution

Following the **Multi-Agent System (MAS) Solution Authoring Contract** in [`public/console/docs/AGENTS.md`](../AGENTS.md):

```
public/console/docs/ble-tutorial/
├── poc_message_types.json           # [Artifact 1] Level 0 Schemas: BLEBeaconSeen, BLEBeaconLost, BLEPluginConfig
├── bindu-ble-presence-plugin/       # [Artifact 5] Level 1 Go BPR Plugin (tinygo.org/x/bluetooth + state engine)
│   ├── main.go                      # Supervised entrypoint scanning physical radio & emitting JSON envelopes
│   ├── presence_engine.go           # Mathematical filtering, dwell debounce, and absence sweeper
│   └── plugin.json                  # BPR plugin manifest declared with inputs and outputs
├── poc_dashboards.json              # [Artifact 4] Level 2 Dashboards: Tabular Transition Log & 2D Operational Map
├── ble_mas_solution.json            # [Artifact 6] Level 3 MASSolutionSet Manifest
├── deploy_mas_solution.sh           # Level 4 Universal 5-Phase Deployment Automation
├── simulator/                       # Interactive Radio Simulator on Root
│   ├── simulate_ble_beacons.go      # Go transmitter broadcasting genuine iBeacon frames via adapter.Advertise()
│   └── simulate_ble_beacons.sh      # Executable wrapper script
├── ble_tutorial_plan.md             # Master Architecture and Verification Plan
└── tutorial_guide.md                # This Comprehensive Hands-On Guide
```

---

## 📜 5. Level 0 Schemas: `poc_message_types.json`

The solution defines three certified Level 0 schemas adhering to the **Bottom-Up Routing Contract** and **Sliding Count Retention**:

| Message Type | Directionality | Routing | Retention | Semantic Purpose |
| :--- | :--- | :--- | :--- | :--- |
| `BLEBeaconSeen` | Bottom-Up | `["local", "parent"]` | `count: 50` | Emitted when a beacon stabilizes inside the room |
| `BLEBeaconLost` | Bottom-Up | `["local", "parent"]` | `count: 50` | Emitted when beacon advertising ceases for 5000ms |
| `BLEPluginConfig` | Top-Down | `["local", "child"]` | `count: 1` | Runtime tuning (dwell window, absence threshold, RSSI cutoffs) |

> [!IMPORTANT]
> **Directionality Rule**: Telemetry and sensor streams MUST use `"routing": ["local", "parent"]`. Never include `"child"`, as child nodes do not need sensor data from other nodes. Including `"child"` causes mesh flooding and infinite rejection error loops.

---

## 🛠️ 6. Level 1 Plugin: `bindu-ble-presence-plugin`

The plugin is implemented in pure Go using `bindunet/sdk` and `tinygo.org/x/bluetooth`:
- **Hardware Access**: Connects to the host's physical Bluetooth adapter (`hci0`). If running in virtual environments, it supports `--backend dry-run` for software emulation.
- **iBeacon Frame Decoder**: Extracts Apple Company ID (`0x004c`), indicator byte (`0x02`), length (`0x15`), 16-byte proximity UUID, 2-byte Major, 2-byte Minor, and 1-byte TX Power.
- **Envelope Stamping**: Formats outgoing semantic events with spatial coordinates and zone IDs for direct ingestion into Bindu's local plane.

---

## 📊 7. Level 2 Real-Time Dashboards

The solution includes two Zero-RAG SQL dashboards in `poc_dashboards.json`:

### 7.1 Tabular Ingress & Dwell Audit Trail (`type: "table"`)
Monitors all presence transitions in chronological order:
```sql
SELECT 
  "$when" AS timestamp, 
  "$type" AS event_type, 
  json_extract(raw_json, '$.fields.beacon_id') AS beacon_id, 
  json_extract(raw_json, '$.fields.scanner_node_id') AS scanner, 
  json_extract(raw_json, '$.fields.zone_id') AS zone, 
  CAST(json_extract(raw_json, '$.fields.rssi_dbm') AS REAL) AS rssi, 
  CAST(json_extract(raw_json, '$.fields.estimated_distance_m') AS REAL) AS distance_m, 
  CAST(json_extract(raw_json, '$.fields.dwell_duration_seconds') AS REAL) AS dwell_s, 
  json_extract(raw_json, '$.fields.status') AS status 
FROM v_messages 
WHERE "$type" IN ('BLEBeaconSeen', 'BLEBeaconLost') 
ORDER BY "$when" DESC 
LIMIT 30;
```

### 7.2 Operation Room 2D Spatial Occupancy Map (`type: "map"`)
Renders the real-time location of active beacons projected onto the `operation-room` Cartesian grid:
```sql
SELECT 
  "$when" AS timestamp, 
  json_extract(raw_json, '$.fields.beacon_id') AS entity_id, 
  json_extract(raw_json, '$.fields.status') AS status, 
  json_extract(raw_json, '$.fields.zone_id') AS zone, 
  CAST(json_extract(raw_json, '$.fields.coordinates_x') AS REAL) AS pos_x, 
  CAST(json_extract(raw_json, '$.fields.coordinates_y') AS REAL) AS pos_y, 
  json_extract(raw_json, '$.fields.where_locagram') AS geom 
FROM v_messages 
WHERE "$type" IN ('BLEBeaconSeen', 'BLEBeaconLost') 
ORDER BY "$when" DESC 
LIMIT 20;
```

---

## 🚀 8. Step-by-Step Hands-On Deployment Guide

### Step 1: Run the Universal 5-Phase Deployment Pipeline
Execute the automated deployment script from your terminal:

```bash
cd public/console/docs/ble-tutorial
./deploy_mas_solution.sh --yes
```

**What happens during this automated pipeline:**
1. **Phase 1 (Ontology)**: Merges `BLEBeaconSeen`, `BLEBeaconLost`, and `BLEPluginConfig` schemas into Root.
2. **Phase 2 (Plugin Certification)**: Compiles `bindu-ble-presence-plugin` with `linux/amd64` native binaries and packages `bindu-ble-presence-plugin.tar.gz`.
3. **Phase 3 (Spatial Reference Plane)**:
   - Registers reference node `operation-room` under `node-dell-2`.
   - Establishes `node-dell-2` as the Cartesian origin.
   - Assigns the $12\,\text{m} \times 8\,\text{m}$ room polygon.
4. **Phase 4 (Solution Application)**: Preflights dashboards, stages the plugin on `node-dell-2`, imports dashboards, and registers `MASSolutionSet`.
5. **Phase 5 (Universal Runtime OTA Triptych)**: Executes `baseline accept -> desired commit -> update wizard` to deploy and start the plugin on `node-dell-2`.

---

### Step 2: Start the BLE Radio Simulator on Root
The simulator transmits genuine iBeacon advertising frames in the 2.4 GHz band.

```bash
cd public/console/docs/ble-tutorial/simulator
./simulate_ble_beacons.sh interactive
```

**Interactive CLI Console**:
```
==================================================
📡 Bindu BLE Beacon Interactive Simulator
Commands:
  1 | on    - Start radio broadcast (Asset in room)
  2 | off   - Stop radio broadcast (Asset left room)
  3 | walk  - Walk-in for 15s then walk-out
  q | quit  - Exit simulator
==================================================
ble-sim>
```

---

### Step 3: Simulate Tag Arrival ("Walk-In")
In the simulator console, type `on` (or `1`):

```bash
ble-sim> on
```

**Observed Engine Behavior**:
1. The physical Bluetooth adapter begins broadcasting iBeacon frames:
   - UUID: `fda50693-a4e2-4fb1-afcf-c6eb07647825`
   - Major: `100`, Minor: `1`
   - Measured Power: `-59 dBm`
2. `node-dell-2` captures the advertising burst over `hci0`.
3. The plugin starts the dwell debounce timer ($1000\,\text{ms}$).
4. Once $1000\,\text{ms}$ has elapsed, the plugin emits `BLEBeaconSeen`:
   ```json
   {
     "$type": "BLEBeaconSeen",
     "$where": "node-dell-2/operation-room",
     "fields": {
       "beacon_id": "beacon-fda50693-100-1",
       "status": "SEEN",
       "zone_id": "operation-room",
       "coordinates_x": 6.0,
       "coordinates_y": 4.0,
       "rssi_dbm": -62.5,
       "estimated_distance_m": 1.48,
       "dwell_duration_seconds": 1.05
     }
   }
   ```
5. The message is routed to Root via MQTT.

---

### Step 4: Simulate Tag Departure ("Walk-Out")
In the simulator console, type `off` (or `2`):

```bash
ble-sim> off
```

**Observed Engine Behavior**:
1. The radio transmitter halts advertising immediately.
2. The edge plugin's background absence sweeper checks the beacon table every second.
3. At $t = 5.0\,\text{s}$, the absence threshold expires.
4. The plugin emits `BLEBeaconLost`:
   ```json
   {
     "$type": "BLEBeaconLost",
     "$where": "node-dell-2/operation-room",
     "fields": {
       "beacon_id": "beacon-fda50693-100-1",
       "status": "LOST",
       "zone_id": "operation-room",
       "last_rssi_dbm": -62.5,
       "dwell_duration_seconds": 22.4
     }
   }
   ```
5. The digital twin transitions to departed.

---

### Step 5: Verify Live in Console & SQL

#### Pure SQL Query via Bindu Console:
Navigate to **Bindu Console -> Messages -> SQL Query**:
```sql
SELECT 
  "$when", 
  "$type", 
  json_extract(raw_json, '$.fields.beacon_id') AS beacon, 
  json_extract(raw_json, '$.fields.status') AS state, 
  json_extract(raw_json, '$.fields.dwell_duration_seconds') AS duration_sec
FROM v_messages 
WHERE "$type" IN ('BLEBeaconSeen', 'BLEBeaconLost') 
ORDER BY "$when" DESC;
```

#### Dashboards View:
Open **Bindu Console -> Dashboards**:
- Launch **📡 BLE Beacon Ingress & Dwell Audit Trail** to view arrival and departure transitions with filtered RSSI and estimated distances.
- Launch **🗺️ Operation Room BLE Beacon Spatial Occupancy Map** to observe the ultrasound twin's coordinates positioned inside the $12\,\text{m} \times 8\,\text{m}$ Operation Room polygon.

---

## 🏆 9. Summary & Architecture Takeaways

By completing this tutorial, you have learned:
1. How to capture genuine physical 2.4 GHz Bluetooth advertising frames using a sandboxed Go BPR plugin.
2. How to eliminate radio frequency multipath noise and boundary jitter using statistical filtering, dwell debouncing, and absence sweepers.
3. How to anchor local Cartesian reference planes (`operation-room` under `node-dell-2`) to enrich telemetry messages with native spatial coordinates (`$where`).
4. How to author and certify Level 0 schemas, Level 1 plugins, Level 2 dashboards, and Level 3 solutions following the Universal 5-Phase Deployment Pipeline.
