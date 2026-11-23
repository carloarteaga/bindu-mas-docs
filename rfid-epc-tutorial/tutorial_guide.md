# 🏷️ Industrial RFID LLRP Dock Portal & 11-Standard EPC Bit-Decoder Multi-Agent System (MAS)

## 📋 Executive Summary & Industrial Scope

In modern automated logistics hubs, cross-docking distribution centers, and manufacturing supply chains, **RAIN RFID (UHF EPC Gen2 / ISO 18000-63)** is the universal standard for physical asset identification. High-speed dock doors and conveyor portals operate fixed 4-antenna RFID reader arrays (such as Impinj Speedway R420, Zebra FX9600, or Alien F800) streaming thousands of tag reads per second over the **Low Level Reader Protocol (LLRP - ISO/IEC 24730-1)**.

However, traditional SCADA and RFID middleware suffer from critical architectural bottlenecks:
1. **Monolithic Ingestion & GC Overhead**: High tag read bursts (thousands of reads/sec) create severe garbage collection pressure when allocating naive heap objects.
2. **Heterogeneous Tag Standards Chaos**: A single forklift entering a dock door carries mixed EPC standards simultaneously:
   - The wooden pallet is tagged with **SSCC-96** (Serial Shipping Container Code).
   - The individual retail carton cases are tagged with **SGTIN-96** (Serialized Global Trade Item Number).
   - The plastic returnable totes are tagged with **GRAI-96** (Global Returnable Asset Identifier).
   - The reach truck forklift carries a **GIAI-96** (Global Individual Asset Identifier).
   - The forklift operator badge is tagged with **GSRN-96** (Global Service Relation Number).
   - The digital delivery bill of lading is tagged with **GDTI-96** (Global Document Type Identifier).
   - The dock door berth itself has a fixed **SGLN-96** location marker.
3. **Lack of Edge Determinism & Spatial Intelligence**: Raw RFID reads do not indicate motion directionality, velocity, or pallet consolidation. Legacy systems require bulky external software to correlate antenna read sequences.

**Bindu** solves these challenges by deploying a native **Multi-Agent System (MAS)** directly at the edge, combining:
- A high-throughput, zero-allocation Go BPR Plugin (`bindu-rfid-epc-plugin`) connecting to LLRP readers and decoding all **11 EPC Global 96-bit standards** at the bit level.
- Deterministic **CEL Declarative Reactors** for real-time 4-antenna spatial directionality and velocity estimation.
- Cognitive **System One Decision-Native Semantic AI Agents (`TypeSafe AI / Jev`)** for automated delivery manifest reconciliation with calibrated RLCD confidence probabilities and CEL text synthesis.
- Pure SQL **Zero-RAG Real-Time Dashboards** for 4-antenna RF telemetry, noise floor, and VSWR monitoring.

---

## 🏛️ 4-Antenna Portal Physical & Spatial Architecture

The industrial dock portal consists of 4 circular polarized UHF antennas arranged in two opposing pairs:
- **Inbound Gate (Entry Plane)**: Antenna 1 (Left), Antenna 2 (Right).
- **Outbound Gate (Exit Plane)**: Antenna 3 (Left), Antenna 4 (Right).

```
                      Physical Dock Door Portal (Width: 2.5m, Depth: 2.0m)
                     ─────────────────────────────────────────────────────
   Warehouse Interior                                                     Truck Loading Dock
                                   [Inbound Flow -->]
                     +---------------------------------------+
   [Antenna 1] ----> | (•) Inbound Left     (•) Outbound Left| <---- [Antenna 3]
   (Port 1)          |                                       |       (Port 3)
                     |       [ 🚚 Forklift + Pallet ]        |
   [Antenna 2] ----> | (•) Inbound Right   (•) Outbound Right| <---- [Antenna 4]
   (Port 2)          +---------------------------------------+
                                  [<-- Outbound Flow]
                     ─────────────────────────────────────────────────────
```

### Spatial Kinematics & Directionality Calculation
When a forklift passes through the portal:
1. **Inbound Transit**: Tags are detected first on Antennas 1 & 2 with peak RSSI at $t_1$, then detected on Antennas 3 & 4 with peak RSSI at $t_2$.
   $$\Delta t = t_2 - t_1 > 0 \implies \text{Direction} = \text{INBOUND}$$
   $$\text{Estimated Velocity } v = \frac{d_{\text{portal}}}{\Delta t} \quad (d_{\text{portal}} \approx 2.0\,\text{m})$$
2. **Outbound Transit**: Tags are detected first on Antennas 3 & 4, then on Antennas 1 & 2:
   $$\Delta t = t_2 - t_1 < 0 \implies \text{Direction} = \text{OUTBOUND}$$
3. **Static Stray Tag**: A tag sitting motionless in the staging area emits sporadic reads with $|\Delta t| \le 150\,\text{ms}$ or on a single antenna pair with lower RSSI (filtered out by CEL Noise Gating).

---

## 🔬 The 11 GS1 / EPC Global Tag Standards Decoded

The `bindu-rfid-epc-plugin` implements a zero-allocation `BitReader` bit-level extractor that inspects the 8-bit EPC header and partition table to decode all 11 global standards into strongly typed Bindu message types:

| Standard | EPC Header | Partition Bits | Decoded Semantic Fields | Bindu Output MessageType | Typical Industrial Application |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **SGTIN-96** | `0x30` | 3 bits (0-6) | Filter, CompanyPrefix, ItemRef, SerialNumber, GTIN-14 | `EpcSgtinRead` | Retail consumer goods, case packs |
| **SSCC-96** | `0x31` | 3 bits (0-6) | Filter, CompanyPrefix, ExtDigit, SerialRef, SSCC-18 | `EpcSsccRead` | Logistics shipping pallets & containers |
| **SGLN-96** | `0x32` | 3 bits (0-6) | Filter, CompanyPrefix, LocationRef, GLNExt, GLN-13 | `EpcSglnRead` | Dock doors, berths, warehouse aisles |
| **GRAI-96** | `0x33` | 3 bits (0-6) | Filter, CompanyPrefix, AssetType, SerialNumber | `EpcGraiRead` | Returnable totes, plastic crates, kegs |
| **GIAI-96** | `0x34` | 3 bits (0-6) | Filter, CompanyPrefix, IndividualAssetRef | `EpcGiaiRead` | Fixed machines, forklifts, AGVs |
| **GSRN-96** | `0x35` | 3 bits (0-6) | Filter, CompanyPrefix, ServiceRef, GSRN-18 | `EpcGsrnRead` | Driver badges, maintenance technicians |
| **GDTI-96** | `0x3C` | 3 bits (0-6) | Filter, CompanyPrefix, DocumentType, SerialNumber | `EpcGdtiRead` | Customs certificates, bills of lading |
| **CPI-96** | `0x3D` | 3 bits (0-6) | Filter, CompanyPrefix, ComponentPartRef, Serial | `EpcCpiRead` | Aerospace & automotive subassemblies |
| **SGCN-96** | `0x3E` | 3 bits (0-6) | Filter, CompanyPrefix, CouponRef, SerialNumber | `EpcSgcnRead` | High-value rebate vouchers & tickets |
| **ITIP-96** | `0x38` | 3 bits (0-6) | Filter, CompanyPrefix, PieceItemRef, PieceNumber | `EpcItipRead` | Textile rolls, cable reels, timber |
| **GID-96** | `0x35` (Alt) | Fixed bitfields | GeneralManagerNumber, ObjectClass, SerialNumber | `EpcGidRead` | General industrial raw components |

---

## 📦 The Rule of 6 Artifacts in this Solution

Following Bindu's Universal MAS Specification, this package includes all 6 required declarative artifacts:

```
public/console/docs/rfid-epc-tutorial/
├── bindu-rfid-epc-plugin/          # [Artifact 5] Standalone Go BPR Plugin (LLRP + 11 EPC Decoders)
│   ├── bitreader.go
│   ├── llrp.go
│   ├── main.go
│   ├── plugin.json
│   └── README.md
├── poc_message_types.json           # [Artifact 1] 15 Declarative JSON Schemas (11 EPCs + RF Telemetry)
├── poc_cel_reactor_config.json      # [Artifact 3] Deterministic Spatial Transit & RSSI Gating Kinetics
├── poc_semantic_reactor_config.json # [Artifact 2] System One Manifest Reconciliation Agent (TypeSafe AI Jev + CEL)
├── poc_dashboards.json              # [Artifact 4] 3 Declarative Dashboards (Transit, EPC Stream, RF Matrix)
├── rfid_mas_solution.json           # [Artifact 6] MASSolutionSet Manifest & OTA Deployment Script
├── deploy_mas_solution.sh           # Universal OTA Triptych Automation
├── simulate_dock_portal.go          # High-Performance LLRP TCP Server & 4-Antenna Transit Simulator
├── simulate_dock_portal.sh          # Simulator Helper Script
└── tutorial-plan.md                 # Master Engineering Specification
```

---

## 🚀 Step-by-Step Hands-On Guide

### Step 0: Provision Centralized Cloud API Secrets (Zero-Plaintext Wire Guarantee)
Under Bindu's Multi-Agent System (MAS) architecture, secrets like cloud LLM API tokens must **never** be hardcoded in plaintext within distributed deployment manifests or `.bundle.json` files. Staging or replicating plaintext keys exposes credentials across edge gateways, SQLite message ledgers (`v_messages`), and git commits.

Instead, Bindu employs the **Zero-Plaintext Wire Guarantee**:
1. Cloud credentials are encrypted locally in the node's AES-GCM vault (`accessControlConfig.brs`) using the `bindu-studio access-control secrets` CLI.
2. Deployment manifests store only the reference URI: `"api_key_secret": "secret:TYPESAFE_API_KEY"`.
3. The running BPR plugin worker dynamically resolves `secret:TYPESAFE_API_KEY` at runtime directly from the encrypted vault.

```bash
# Provision TYPESAFE_API_KEY into the encrypted vault
go run ./cmd/bindu-studio access-control secrets set TYPESAFE_API_KEY \
  --value "YOUR_ACTUAL_TYPESAFE_API_KEY" \
  --yes

# Verify vault status
go run ./cmd/bindu-studio access-control secrets validate
```

---

### Step 1: Start the LLRP Dock Portal Mock Reader
Open a terminal and start the mock LLRP reader simulator listening on standard TCP port `5084`:

```bash
cd public/console/docs/rfid-epc-tutorial
./simulate_dock_portal.sh
```

**Expected Simulator Output**:
```
=================================================================
📡 BINDU LLRP RFID READER & 4-ANTENNA DOCK PORTAL SIMULATOR
   Listening on TCP 0.0.0.0:5084 (LLRP Standard Port)
   Simulating 4 Antennas (1-2: Inbound, 3-4: Outbound)
   Simulating all 11 GS1 / EPC Global 96-bit Standards
=================================================================
```

---

### Step 2: Deploy the Full MAS Solution via the Universal Protocol
In a second terminal, execute the declarative deployment script:

```bash
cd public/console/docs/rfid-epc-tutorial
./deploy_mas_solution.sh --yes
```

The script automatically executes the **Universal MAS Deployment Cadence**:
1. **Step 0**: Mandatory Offline Preflight Validation Gate:
   - `bindu-studio cel-reactor validate`
   - `bindu-studio semantic-ai-reactor validate`
   - `bindu-studio dashboard validate`
   - `bindu-studio solution validate --strict`
2. **Step 1**: Ingests all 15 JSON schemas into Bindu Root (`message-types merge`).
3. **Step 2 & 3**: Compiles, certifies, and applies `bindu-rfid-epc-plugin` (`plugin certify` + `plugin apply`).
4. **Step 4**: Deploys the CEL spatial kinetics and SLM cognitive manifest reconciler.
5. **Step 5**: Applies the declarative `MASSolutionSet` bundle.
6. **Step 6**: Executes the Universal Runtime OTA Triptych (`baseline accept` $\to$ `desired commit` $\to$ `update wizard`).

---

### Step 3: Inspect Live Decoded EPC Streams in Bindu Console
Open your web browser and navigate to the Bindu Console:
👉 **`http://localhost:8080/console/`**

1. Select the **"Industrial RFID LLRP Dock Portal & 11-Standard EPC Digital Twin"** MAS filter in the top navigation bar.
2. Navigate to **Dashboards**:
   - **🚚 RFID Dock Portal Transit & Directionality Monitor**: Observe real-time synthesized forklift transit events with estimated velocity (e.g. `1.52 m/s`), direction (`INBOUND` / `OUTBOUND`), and consolidated pallet SSCC.
   - **🏷️ 11-Standard EPC Global Tag Stream & Bit-Decoder**: Inspect live stream of decoded SGTIN, SSCC, SGLN, GRAI, GIAI, GSRN, GDTI, CPI, SGCN, ITIP, and GID tags.
   - **📡 4-Antenna RF Physical Layer & Health Telemetry**: Monitor antenna Tx power, sliding average RSSI, noise floor, reflection VSWR, and read rate per second.

---

### Step 4: Verify Zero-RAG SQL Queries Across the Mesh
Using `bindu-studio`, execute read-only SQL queries against the distributed SQLite ledger:

```bash
# Query the latest 10 decoded EPC reads
go run ./cmd/bindu-studio query "SELECT \"$when\", \"$type\", json_extract(raw_json, '$.fields.canonical_urn') AS urn, json_extract(raw_json, '$.fields.peak_rssi_dbm') AS rssi FROM v_messages WHERE \"$type\" LIKE 'Epc%Read' ORDER BY \"$when\" DESC LIMIT 10"

# Query synthesized portal transit velocity and direction
go run ./cmd/bindu-studio query "SELECT \"$when\", json_extract(raw_json, '$.fields.direction') AS dir, json_extract(raw_json, '$.fields.estimated_velocity_mps') AS mps, json_extract(raw_json, '$.fields.pallet_sscc') AS sscc FROM v_messages WHERE \"$type\" = 'PortalTransitEvent' ORDER BY \"$when\" DESC LIMIT 5"
```

---

## 🛡️ Summary of Architectural Innovations

1. **Zero-Alloc Bit-Level EPC Unpacking**: Pure Go bitstream decoding without memory allocations in the hot path.
2. **Deterministic Spatial Synthesis via CEL**: Real-time velocity and direction classification at sub-millisecond latency.
3. **Cognitive Edge Manifest Audit**: Local SLM agents operating asynchronously without cloud latency or vendor lock-in.
4. **Unified Multi-Standard Visibility**: Single pane of glass for all 11 EPC Global identifiers across physical infrastructure.
