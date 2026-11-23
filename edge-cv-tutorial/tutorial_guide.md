# 👁️ Tutorial Guide: Edge Computer Vision Multi-Camera Authority Arbitration & Workstation Presence

Welcome to the **Bindu Edge Computer Vision (Edge CV)** tutorial guide. This tutorial demonstrates how to build, deploy, and operate an intelligent multi-camera computer vision pipeline in Bindu with zero Python runtime dependencies, deterministic visual authority arbitration, and single-transaction Multi-Agent System (MAS) packaging.

---

## 🎯 Executive Summary & The AI Architect Paradigm

In traditional computer vision systems, multi-camera tracking is notoriously fragile, requiring complex homography calibrations, planar coordinate warps, and high-latency Python orchestration servers.

Bindu reimagines Multi-Camera Computer Vision through the **Bindu 3-Layer Architecture**:
1. **Semantics:** Declarative message contracts (`EdgeCVSet`, `EdgeCVSetStatus`, `AppearedClass`, `DisappearedClass`, `WorkstationPresenceSession`).
2. **Runtime:** Native sub-microsecond Common Expression Language (**CEL**) reactors performing spatial overlap arbitration and deduplication.
3. **Operations:** Self-contained MAS Solution Bundles (`.bundle.json`) deployed via `bindu-studio solution apply` in an atomic transaction.

```
+-----------------------------------------------------------------------------------+
|                            THE AI ARCHITECT PATTERN                               |
|                                                                                   |
|  "The goal of this tutorial is not merely to show a human how to configure       |
|   two cameras, but to establish the semantic design pattern for autonomous AI     |
|   agents (like Antigravity / Agy) to deploy camera networks by simply             |
|   reasoning over declarative spatial adjacency lists and CEL authority rules."   |
+-----------------------------------------------------------------------------------+
```

---

## 📹 Distributed Mesh Topology & Hardware Environment

This tutorial runs across a distributed Bindu mesh topology:

* **Edge Gateway (`node-dell-1` - Dell Inspiron 11):** Hosts the native BPR Plugin [`bindu-edge-cv-plugin`](./bindu-edge-cv-plugin/main.go) with embedded ONNX Runtime C++ engine, performing real-time video capture and YOLO detection at the edge using the network Tapo RTSP stream at 1 FPS (freeing 100% of workstation CPU).
* **Central Node (`root`):** Hosts the central CEL Reactor [`workstation-presence-overlap-arbitrator`](./poc_cel_reactor_config.json) and the HTMX Real-time Dashboards.

| Camera ID | Hardware / Model | Connection / Transport | Executing Node | Role & Placement | Primary Authority Weight |
|---|---|---|---|---|---|
| **`cam-tapo-room`** | TP-Link Tapo C200 / C210 | RTSP TCP/IP (`192.168.12.177:554/stream2`) | **`node-dell-1`** | Room ambient overview & presence tracking (1 FPS) | **Authoritative Edge Vision** |
| **`cam-usb-root`** *(optional simulated/aux)* | USB 2.0 HD Webcam | Direct `/dev/video0` (V4L2 Pure Go) | **`node-1`** | Front-facing direct desk view (multi-camera overlap scenarios) | **Secondary / Overlap Sighting** |

### 📋 Edge Gateway Host Prerequisites & Installation
Before running on physical hardware, ensure the target edge gateway has the required system utilities:

```bash
# 1. Install FFmpeg (Mandatory for RTSP H.264 stream demuxing in pipe mode)
sudo apt update && sudo apt install -y ffmpeg

# 2. Install V4L2 utilities (Optional: only if using local USB webcams on /dev/video*)
sudo apt install -y v4l-utils
```

> [!NOTE]
> **Zero External Python / Pip Dependencies:**  
> The Go BPR plugin [`bindu-edge-cv-plugin`](./bindu-edge-cv-plugin/README.md) is compiled to a standalone native binary linking to `libonnxruntime.so.1.24.1` and `yolox_s.onnx` (downloaded on-demand via `./setup_runtime.sh`). It requires **no Python, PyTorch, or PIP environment**.
>
> **FFmpeg Socket Timeout Contract:**  
> For modern FFmpeg ($\ge 5.x/6.x$), socket timeouts on RTSP connections are specified using `-timeout <microseconds>` (e.g. `-timeout 5000000` for 5s). Legacy `-stimeout` is unrecognized.
>
> **RTSP Security & Camera Isolation Best Practices:**  
> - **Dedicated Viewer Account:** Always configure cameras with a low-privilege RTSP viewer account (e.g. `binducv`), never the camera's master administrator account.
> - **VLAN / Network Isolation:** Connect IP cameras to an isolated surveillance VLAN / subnet inaccessible from the public internet.
> - **Zero-Plaintext Wire Guarantee for Media:** Out-of-band detection snapshots uploaded to MinIO/S3 are authenticated using the centralized access control vault key (`minio_media_secret`) stored in `accessControlConfig.brs`.

---

## ⏯️ Dynamic Video Capture & Inference Control (`EdgeCVSetStatus`)

The plugin supports declarative runtime capture control via the `EdgeCVSetStatus` message type:

```json
{
  "$type": "EdgeCVSetStatus",
  "$node-dest": "node-dell-1",
  "$group": "config",
  "$producer": "node-dell-1",
  "$user": "admin",
  "$when": "2026-10-03T12:00:00Z",
  "fields": {
    "capture": true
  }
}
```

### Operational Lifecycle Contract:
1. **`capture: true`**: Actively captures video frames from physical/network cameras and executes ONNX YOLO inference.
2. **`capture: false`**: Pauses capture immediately, cancels video demuxing/stream loops, closes device handles, and stops CPU consumption.
3. **Retention**: Exactly `count: 1` per node/producer.
4. **Bootstrapping**:
   * If **no `EdgeCVSetStatus` event exists** in the ledger on plugin boot, the plugin defaults to `capture = true` and starts immediately.
   * If an `EdgeCVSetStatus` event was previously persisted, the plugin initializes respecting that state (`capture: false` stays paused until commanded to start).

### 🕹️ Dynamic Control via Simulator Script (`SetCaptureStatus.sh`)
You can dynamically toggle camera capture and AI inference across any mesh node using the dedicated CLI simulator script:

```bash
# Resume/Start capture and inference on root node
./public/console/docs/edge-cv-tutorial/SetCaptureStatus.sh true

# Pause capture and free CPU resources on root node
./public/console/docs/edge-cv-tutorial/SetCaptureStatus.sh false

# Target a remote gateway node (e.g. node-dell-1)
./public/console/docs/edge-cv-tutorial/SetCaptureStatus.sh start --node node-dell-1
./public/console/docs/edge-cv-tutorial/SetCaptureStatus.sh stop --node node-dell-1
```

---

## 📐 Visual Overlap Arbitration Mathematics

When Carlos arrives at his desk, both cameras observe him concurrently. Rather than emitting two disjoint presence sessions (false double-counting), the system calculates a real-time **Authority Score**:

$$\text{AuthorityScore} = \text{BBoxArea} \times \text{AverageConfidence}$$

* **`cam-usb-root` Score:** $0.395 \times 0.96 = \mathbf{0.3792}$ 🏆 (*Primary Winner*)
* **`cam-tapo-room` Score:** $0.072 \times 0.73 = \mathbf{0.0525}$ (*Supporting Sighting*)

The CEL reactor identifies the spatial overlap within a 45-second sliding window and consolidates the event into a single authoritative `WorkstationPresenceSession`:

```json
{
  "session_id": "session-cam-usb-root-track-501",
  "station_id": "carlos-dev-workstation",
  "primary_camera_id": "cam-usb-root",
  "supporting_camera_ids": ["cam-tapo-room"],
  "target_class": "person",
  "clothing_color": "NAVY_BLUE",
  "dwell_duration_seconds": 7.0,
  "authority_score": 0.3792,
  "session_status": "COMPLETED"
}
```

---

## 🚀 Universal MAS Solution & Plugin Deployment Protocol

Deploying an end-to-end MAS Solution that orchestrates edge BPR plugins consists of the following steps:

### Step 0: Provision Centralized Storage Secret in Access Control Vault
The Edge CV plugin persists camera detection snapshots via `minio_local`, which references `"secret_key": "secret:minio_media_secret"`. Before deployment, ensure the storage secret is sealed into the encrypted vault (`accessControlConfig.brs`):

```bash
# Provision MinIO storage secret in the encrypted access control vault
go run ./cmd/bindu-studio access-control secrets set minio_media_secret --value "bindu-media-123" --base-dir .
```

### Phase 1: Declarative Solution Bundle Application (Root)
Synchronize all schemas, CEL reactors, dashboards, and solution manifests in a single atomic transaction:

```bash
go run ./cmd/bindu-studio solution apply \
  --file public/console/docs/edge-cv-tutorial/bundles/solution_edge_cv.bundle.json \
  --auth console:StrongPass123 \
  --yes
```

This synchronizes:
* ✅ 5 Message Type Schemas (`EdgeCVSet`, `EdgeCVSetStatus`, `AppearedClass`, `DisappearedClass`, `WorkstationPresenceSession`)
* ✅ 1 Sub-microsecond CEL Overlap Reactor (`workstation-presence-overlap-arbitrator`)
* ✅ 2 Console Dashboards (`carlos-workstation-presence-monitor`, `multi-camera-vision-stream-audit`)
* ✅ 1 Solution Manifest in the central cluster registry.

---

### Phase 2: Native BPR Plugin Certification & Cluster Import
Compile, attest, and stage the edge plugin binary:

```bash
# 1. Compile & certify for target gateway architecture (e.g. linux/amd64)
./public/console/docs/edge-cv-tutorial/build_and_certify_plugin.sh

# 2. Import certified package into root cluster registry
go run ./cmd/bindu-studio plugin import \
  dist/plugins/bindu-edge-cv-plugin-1.0.0-linux-amd64.tar.gz \
  --auth console:StrongPass123
```

---

### Phase 3: Runtime OTA Rollout to Edge Gateways (The Runtime Triptych)
To physically deliver the staged plugin binary to remote edge nodes (e.g. `node-dell-1`) and trigger process activation, execute the mandatory **Runtime Triptych**:

```bash
# 1. Acknowledge baseline state
go run ./cmd/bindu-studio runtime baseline accept --yes

# 2. Commit desired state with staged plugins
go run ./cmd/bindu-studio runtime desired commit --note "Deploy bindu-edge-cv-plugin@1.0.0"

# 3. Rollout OTA update wizard to edge nodes
go run ./cmd/bindu-studio runtime update wizard --yes --auth console:StrongPass123
```

> [!TIP]
> **Automated 1-Click Script:** All three phases above are automated end-to-end in:
> ```bash
> ./public/console/docs/edge-cv-tutorial/deploy_mas_solution.sh --yes
> ```

---

## 🧪 Testing with the Multi-Camera Simulator

Run the automated simulation script to emit a realistic presence sequence:

```bash
./public/console/docs/edge-cv-tutorial/simulate_edge_cv.sh
```

Or execute the Go telemetry publisher directly:

```bash
go run ./public/console/docs/edge-cv-tutorial/simulate_edge_cv.go \
  -host=127.0.0.1 \
  -port=1885 \
  -user=console \
  -pass=StrongPass123
```

### What happens during the simulation:
1. **[T+0s]** Syncs `EdgeCVSet` declarative camera topology to MQTT `bindu/in`.
2. **[T+2s]** `cam-tapo-room` detects Carlos entering the room (`AppearedClass`).
3. **[T+4s]** `cam-usb-root` detects Carlos sitting directly in front of the desk (`AppearedClass`).
4. **[T+10s]** Carlos stands up:
   * `cam-tapo-room` emits `DisappearedClass` (Score: $0.0525$).
   * `cam-usb-root` emits `DisappearedClass` (Score: $0.3792$).
5. **CEL Reactor** intercepts both events, recognizes the spatial adjacency correlation, and produces exactly **ONE** `WorkstationPresenceSession`!

---

## 📊 Verification & SQL Queries

Open the Bindu Console at `http://localhost:8080/console/` or execute SQL queries via `/api/v1/messages/sql`:

### 1. Authoritative Workstation Focus Sessions
```sql
SELECT 
  "$when" AS timestamp,
  json_extract(raw_json, '$.fields.station_id') AS workstation,
  json_extract(raw_json, '$.fields.primary_camera_id') AS primary_camera,
  json_extract(raw_json, '$.fields.supporting_camera_ids') AS supporting_cameras,
  ROUND(CAST(json_extract(raw_json, '$.fields.dwell_duration_seconds') AS REAL) / 60.0, 2) AS focus_minutes,
  json_extract(raw_json, '$.fields.clothing_color') AS clothing,
  json_extract(raw_json, '$.fields.authority_score') AS authority_score
FROM v_messages 
WHERE "$type" = 'WorkstationPresenceSession'
ORDER BY "$when" DESC;
```

### 2. Multi-Camera Overlap Audit
```sql
SELECT 
  "$when" AS timestamp,
  json_extract(raw_json, '$.fields.camera_id') AS camera,
  json_extract(raw_json, '$.fields.class_name') AS detected_class,
  CAST(json_extract(raw_json, '$.fields.confidence') AS REAL) AS confidence,
  CAST(json_extract(raw_json, '$.fields.bbox_area') AS REAL) AS bbox_area,
  json_extract(raw_json, '$.fields.dominant_color') AS color
FROM v_messages 
WHERE "$type" IN ('AppearedClass', 'DisappearedClass')
ORDER BY "$when" DESC 
LIMIT 20;
```

---

## 🛠️ Production Guide: Building, Certifying & Deploying the Real Go BPR Plugin

Beyond simulation, here is the complete step-by-step workflow to compile, certify, and install the real **Go Computer Vision BPR Plugin (`bindu-edge-cv-plugin`)** on your physical Linux node:

```
+───────────────────────────────────────────────────────────────────────────────────+
|                 PRODUCTION BPR PLUGIN LIFECYCLE PIPELINE                          |
|                                                                                   |
|  1. Install Host Prerequisites (sudo apt install -y ffmpeg v4l-utils)             |
|  2. Certify & Build Release Bundle (build_and_certify_plugin.sh)                  |
|  3. Apply Plugin to Node (bindu-studio plugin apply)                              |
|  4. Synchronize Mesh Desired State (bindu-studio runtime desired commit)          |
|  5. Automated Zero-Touch Rollout (bindu-studio runtime update wizard)             |
|  6. Deploy MAS Solution & Dashboards (deploy_mas_solution.sh)                     |
+───────────────────────────────────────────────────────────────────────────────────+
```

### 1. Install Edge Gateway Host Dependencies
On the target Linux machine (e.g. `node-dell-1`):
```bash
sudo apt update && sudo apt install -y ffmpeg v4l-utils
```

### 2. Certify and Package the Plugin with Bindu Studio
From the workstation / build environment:
```bash
# Automated Doctor check and amd64 release bundle packaging:
./public/console/docs/edge-cv-tutorial/build_and_certify_plugin.sh
```
*Outputs certified plugin bundle: `public/console/docs/edge-cv-tutorial/bundles/bindu-edge-cv-plugin-1.0.0.tar.gz`.*

### 3. Apply Plugin to Target Gateway Node
```bash
go run ./cmd/bindu-studio plugin apply \
  --bundle ./public/console/docs/edge-cv-tutorial/bundles/bindu-edge-cv-plugin-1.0.0.tar.gz \
  --target node \
  --node node-dell-1
```

### 4. Live Runtime Rollout via OTA
```bash
# Accept baseline and commit target desired configuration
go run ./cmd/bindu-studio runtime baseline accept --yes
go run ./cmd/bindu-studio runtime desired commit --note "Deploy bindu-edge-cv-plugin to node-dell-1"

# Rollout to remote node (autonomous download, candidate activation & restart)
go run ./cmd/bindu-studio runtime update wizard --yes
```

### 5. Automated 1-Click Complete Pipeline (MAS Solution)
Alternatively, to deploy the entire multi-camera pipeline (MessageTypes, Plugin, CEL Reactor, and Dashboards) in a single command:
```bash
./public/console/docs/edge-cv-tutorial/deploy_mas_solution.sh --yes
```

---

## 📁 Artifact Index

All solution artifacts are available in [`public/console/docs/edge-cv-tutorial/`](./):

* [`tutorial-plan.md`](./tutorial-plan.md): Master architectural specification and design document.
* [`poc_message_types.json`](./poc_message_types.json): Declarative message contracts.
* [`poc_edge_cv_config.json`](./poc_edge_cv_config.json): Physical USB and Tapo RTSP stream orchestration configuration.
* [`poc_cel_reactor_config.json`](./poc_cel_reactor_config.json): CEL Overlap Arbitration Reactor.
* [`poc_dashboards.json`](./poc_dashboards.json): Operational dashboards for presence tracking.
* [`solution_edge_cv.json`](./solution_edge_cv.json): MAS Solution manifest.
* [`bundles/solution_edge_cv.bundle.json`](./bundles/solution_edge_cv.bundle.json): Self-contained 1-click deployment bundle.
* [`bindu-edge-cv-plugin/`](./bindu-edge-cv-plugin/): Complete Go source tree for the BPR computer vision plugin.
* [`setup_runtime.sh`](./setup_runtime.sh): Automated downloader for ONNX Runtime shared library and YOLOX-S weights.
* [`build_and_certify_plugin.sh`](./build_and_certify_plugin.sh): Script to compile, doctor-check, and certify the BPR plugin bundle.
* [`deploy_mas_solution.sh`](./deploy_mas_solution.sh): 1-Click solution deployer.
* [`simulate_edge_cv.go`](./simulate_edge_cv.go): Go MQTT telemetry publisher.
* [`simulate_edge_cv.sh`](./simulate_edge_cv.sh): End-to-end test runner.

