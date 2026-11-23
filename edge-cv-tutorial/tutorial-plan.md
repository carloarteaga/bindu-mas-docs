# **Bindu Edge CV: Multi-Camera Visual Authority Arbitration & Desk Presence Counting**
## **Engineering Analysis & Master Tutorial Plan**

---

## **1. Executive Summary & Problem Formulation**

In real-world computer vision deployments (workstation monitoring, retail cashier counters, gas station islands, or warehouse assembly benches), multiple cameras observe the same physical space from different vantage points with **overlapping Fields of View (FOV)**.

```
                     ┌────────────────────────────────────────┐
                     │    DEVELOPER WORKSTATION (CARLOS)      │
                     │                                        │
      [ Camera 1 ] ──► [ Desk Chair: 👤 PERSON (CARLOS) ] ◄── [ Camera 2 ]
   (Local USB on Root: (Primary Workstation Target)    (Tapo IP at Angle:
    Full Upper Body,                                    Side/Partial View,
    BBox: 450x380,                                      BBox: 120x90,
    Area: 0.38, Conf: 0.95)                             Area: 0.07, Conf: 0.68)
```

### **The Double-Counting Crisis in Traditional Vision Systems**
When Carlos sits down at his development desk to write code:
1. **Camera 1 (`cam-usb-root` on `/dev/video0`):** USB webcam directly facing the workstation. It captures Carlos front and center with high confidence ($0.95$), large bounding box area ($0.38$), and tracks his presence continuously for $45\text{ minutes}$ ($2700\,\text{s}$).
2. **Camera 2 (`cam-tapo-room` on `192.168.12.177`):** TP-Link Tapo IP camera mounted at an angle in the room. It captures a partial side-profile/shoulder of Carlos at the edge of its frame with moderate confidence ($0.68$) and small bounding box area ($0.07$).

**The Consequence of Naive Counting:**  
Without cross-camera authority arbitration, standard vision systems evaluate each camera independently and report that **2 different people** worked at the desk, causing **100% false count inflation**.

> [!IMPORTANT]
> **The Role of the AI Architect (AI-Native Design Pattern):**  
> The overarching objective of this tutorial is not merely to show human engineers how to configure computer vision. Its primary purpose is to establish a standardized, deterministic **Semantic Design Pattern** that enables **Autonomous AI Agents (such as Agy)** to author, deploy, and orchestrate complex multi-camera vision networks across physical hardware simply by declaring spatial adjacency lists and operational intent.

---

### **1.1 The Bindu Semantic Solution: Overlap Authority Arbitration**

Bindu solves this problem deterministically by separating **Raw Edge Vision Sensing** from **Semantic Authority Arbitration**:

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                        LAYER 1: LOCAL EDGE SENSING (BPR Plugin)                        │
│                                                                                        │
│  [USB /dev/video0 Loop] ──┐                                                            │
│  [Tapo RTSP IP Loop   ] ──┼─► [Shared ONNX/YOLO Pool] ──► [2D Image-Space Tracker]     │
│                           │                                       │                    │
│                                                                   ▼                    │
│                                                 EMIT: AppearedClass / DisappearedClass │
└───────────────────────────────────────────────────────────────────┬────────────────────┘
                                                                    │ (MQTT bindu/in)
┌───────────────────────────────────────────────────────────────────▼────────────────────┐
│              LAYER 2: MULTI-CAMERA AUTHORITY ARBITRATION (Bindu Core)                  │
│                                                                                        │
│    [AppearedClass / DisappearedClass Events] ──► [SQLite Ledger (Zero-RAG Context)]    │
│                                                                   │                    │
│  ┌────────────────────────────────────────────────────────────────┴───────────────┐    │
│  │ CEL Overlap Arbitration Reactor                                                │    │
│  │ 1. Verify Camera Adjacency: cam-tapo-room ∈ adjacent_camera_ids(cam-usb-root)  │    │
│  │ 2. Temporal Concurrency: |t_usb - t_tapo| < 15 seconds                         │    │
│  │ 3. Target Class Match: class == "person"                                       │    │
│  │ 4. Compute Visual Authority Score:                                             │    │
│  │      Score(USB)  = Area(0.38) × Conf(0.95) × Centering(1.0) = 0.361 ──► 🏆     │    │
│  │      Score(Tapo) = Area(0.07) × Conf(0.68) × Centering(0.4) = 0.019 ──► 🚫     │    │
│  └────────────────────────────────────────────────────────────────┬───────────────┘    │
│                                                                   ▼                    │
│                                     EMIT: WorkstationPresenceSession (Single Truth)    │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

1. **The BPR Plugin is Lean & Modular:** It ingests both the local USB stream and the Tapo IP RTSP stream, runs inference across a shared ONNX/YOLO worker pool, tracks persons in 2D image coordinates, and emits raw events:
   * **`AppearedClass`**: Emitted when Carlos sits down and is confirmed in the camera's FOV.
   * **`DisappearedClass`**: Emitted when Carlos stands up and leaves the desk, reporting total `dwell_duration_seconds`.
2. **Bindu Core Arbitrates Authority:** A deterministic **CEL Reactor** intercepts overlapping sightings, designates `cam-usb-root` as the **Primary Workstation Authority**, suppresses duplicate counts from the supporting Tapo camera, and emits a single, authoritative **`WorkstationPresenceSession`** event.

---

## **2. Core Architectural Principles & Guardrails**

1. **3-Layer Separation of Concerns:**  
   * **Semantics:** MessageTypes (`EdgeCVSet`, `EdgeCVSetStatus`, `AppearedClass`, `DisappearedClass`, `WorkstationPresenceSession`).
   * **Runtime:** Memory-isolated BPR plugin process (`bindu-edge-cv-plugin`) communicating via `stdin/stdout` JSON.
   * **Operations:** Declarative MAS Solution Bundle deployed via `bindu-studio solution apply`.
2. **Zero Fragile Calibration:**  
   No manual homography matrices, 3D metric calibrations, or bird's-eye stitching. Camera spatial relationships are declared via a simple **Adjacency List** (`adjacent_camera_ids: ["cam-tapo-room"]`).
3. **Deterministic Authority Formula:**
   $$\text{AuthorityScore} = \text{BBoxArea} \times \text{Confidence} \times \text{CenteringWeight}$$
   $$\text{CenteringWeight} = 1.0 - 2.0 \times \sqrt{(c_x - 0.5)^2 + (c_y - 0.5)^2}$$
   *Where $(c_x, c_y)$ is the normalized bounding box centroid $[0.0..1.0]$.*
4. **No Domain Pollution:**  
   QR codes and barcodes are eliminated from the vision tracking core; they belong in dedicated discrete plugins.

---

## **3. Declarative Configuration Specification: `EdgeCVSet`**

`EdgeCVSet` unifies heterogeneous physical camera transports into a single declarative contract configured for Carlos's live lab environment:

```json
{
  "$type": "EdgeCVSet",
  "$node-dest": "root",
  "$group": "config",
  "$producer": "root",
  "$user": "admin",
  "$when": "2026-09-28T12:00:00Z",
  "fields": {
    "gateway_node_id": "root",
    "plugin_id": "bindu-edge-cv-plugin",
    "global_settings": {
      "inference_engine": "onnx_yolox_nano",
      "confidence_threshold": 0.50,
      "worker_pool_size": 2,
      "frame_cadence_fps": 5,
      "classes_of_interest": ["person"],
      "color_extraction_enabled": true,
      "snapshot_storage_provider": "minio_local",
      "snapshot_bucket": "cv-snapshots"
    },
    "cameras": [
      {
        "camera_id": "cam-usb-root",
        "station_id": "carlos-dev-workstation",
        "name": "Local USB Webcam on Root (V4L2)",
        "source_kind": "usb",
        "source_ref": "/dev/video0",
        "adjacent_camera_ids": ["cam-tapo-room"],
        "min_presence_frames": 3,
        "lost_debounce_frames": 5,
        "enabled": true
      },
      {
        "camera_id": "cam-tapo-room",
        "station_id": "room-ambient-overview",
        "name": "TP-Link Tapo IP Camera at Angle (RTSP TCP/IP)",
        "source_kind": "tcp_ip",
        "source_ref": "rtsp://binducv:abubindu@192.168.12.177:554/stream2",
        "rtsp_transport": "tcp",
        "adjacent_camera_ids": ["cam-usb-root"],
        "min_presence_frames": 3,
        "lost_debounce_frames": 5,
        "enabled": true
      }
    ]
  }
}
```

### **3.1 Live Reference Hardware Profile**

| Camera ID | Hardware Model | Transport / Ingestion Ref | Protocol / Codec | Station Mapping | Physical Role in Desk Setup |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`cam-usb-root`** | Standard USB UVC Webcam | `/dev/video0` | Linux V4L2 (`go4vl` raw YUYV/MJPEG) | `carlos-dev-workstation` | **Primary Authority:** Direct front view of Carlos at computer desk |
| **`cam-tapo-room`** | TP-Link Tapo C-Series IP Camera | `rtsp://binducv:abubindu@192.168.12.177:554/stream2` | RTSP over TCP (`gortsplib` H.264 Substream 360p) | `room-ambient-overview` | **Supporting View:** Angled side-view capturing partial shoulder/profile |

---

### **3.2 Declarative Runtime Capture Control: `EdgeCVSetStatus`**

The `EdgeCVSetStatus` message contract provides dynamic runtime lifecycle control over camera stream capture loops and ONNX neural inference without stopping or restarting the BPR plugin process.

```json
{
  "$type": "EdgeCVSetStatus",
  "$node-dest": "root",
  "$group": "config",
  "$producer": "root",
  "$user": "admin",
  "$when": "2026-10-03T12:00:00Z",
  "fields": {
    "capture": true
  }
}
```

* **Contract & Semantic Behavior:**
  1. `capture == true`: Active frame capture & ONNX YOLO inference across all configured cameras.
  2. `capture == false`: Pause capture, cancel stream loops, close video source handles, free CPU/GPU resources.
  3. **Bootstrap State:** If no `EdgeCVSetStatus` event exists in `v_messages` upon plugin startup, capture defaults to `true`. If `capture == false` is persisted, the plugin initializes dormant and awaits an enabling event.
  4. **Retention:** `retentionMode: count`, `retentionValue: 1` (`persist: true`).

---

## **4. Semantic Message Contracts (`MessageTypes`)**

### **4.1 `AppearedClass` (Raw Detection Onset)**
```json
{
  "name": "AppearedClass",
  "description": "Emitted when a tracked person is verified and enters a camera field of view",
  "emoji": "👁️",
  "retentionMode": "days",
  "retentionValue": 7,
  "routing": ["local", "parent"],
  "properties": [
    { "name": "camera_id", "type": "string", "required": true, "description": "Observing camera identifier" },
    { "name": "station_id", "type": "string", "required": true, "description": "Assigned station identifier (e.g. carlos-dev-workstation)" },
    { "name": "track_id", "type": "string", "required": true, "description": "Local 2D camera tracking session ID" },
    { "name": "class_name", "type": "string", "required": true, "description": "Detected class (person)" },
    { "name": "confidence", "type": "number", "required": true, "description": "Detection confidence score (0.0 to 1.0)" },
    { "name": "dominant_color", "type": "string", "required": true, "description": "Extracted clothing color (BLACK, BLUE, WHITE, RED)" },
    { "name": "bbox_area", "type": "number", "required": true, "description": "Normalized bounding box area (w * h) [0.0..1.0]" },
    { "name": "bbox_centroid", "type": "array", "required": true, "description": "Normalized centroid coordinates [cx, cy]" },
    { "name": "snapshot_uri", "type": "media-storage", "required": false, "description": "MinIO snapshot frame URI" }
  ]
}
```

---

### **4.2 `DisappearedClass` (Raw Detection Departure)**
```json
{
  "name": "DisappearedClass",
  "description": "Emitted when a tracked person leaves the camera FOV or stands up from desk",
  "emoji": "💨",
  "retentionMode": "days",
  "retentionValue": 7,
  "routing": ["local", "parent"],
  "properties": [
    { "name": "camera_id", "type": "string", "required": true, "description": "Observing camera identifier" },
    { "name": "station_id", "type": "string", "required": true, "description": "Assigned station identifier" },
    { "name": "track_id", "type": "string", "required": true, "description": "Local 2D camera tracking session ID" },
    { "name": "class_name", "type": "string", "required": true, "description": "Detected class name (person)" },
    { "name": "dwell_duration_seconds", "type": "number", "required": true, "description": "Total duration person remained at desk in seconds" },
    { "name": "final_dominant_color", "type": "string", "required": true, "description": "Consolidated clothing color" },
    { "name": "avg_confidence", "type": "number", "required": true, "description": "Average detection confidence over entire session" },
    { "name": "max_bbox_area", "type": "number", "required": true, "description": "Maximum bounding box area observed" }
  ]
}
```

---

### **4.3 `WorkstationPresenceSession` (Authoritative Consolidated Event)**
```json
{
  "name": "WorkstationPresenceSession",
  "description": "Authoritative consolidated desk presence session emitted after multi-camera overlap arbitration",
  "emoji": "💻",
  "retentionMode": "days",
  "retentionValue": 30,
  "routing": ["local", "parent"],
  "properties": [
    { "name": "session_id", "type": "string", "required": true, "description": "Unique consolidated session identifier" },
    { "name": "station_id", "type": "string", "required": true, "description": "Physical workstation (carlos-dev-workstation)" },
    { "name": "primary_camera_id", "type": "string", "required": true, "description": "Winning camera declared as Primary Authority (cam-usb-root)" },
    { "name": "supporting_camera_ids", "type": "array", "required": true, "description": "Adjacent cameras that observed person concurrently (cam-tapo-room)" },
    { "name": "target_class", "type": "string", "required": true, "description": "Class identifier (person)" },
    { "name": "clothing_color", "type": "string", "required": true, "description": "Validated dominant clothing color" },
    { "name": "dwell_duration_seconds", "type": "number", "required": true, "description": "Total authoritative focus/work time at desk" },
    { "name": "authority_score", "type": "number", "required": true, "description": "Computed primary visual authority score" },
    { "name": "session_status", "type": "string", "required": true, "enum": ["COMPLETED", "IN_PROGRESS", "ABORTED"] }
  ]
}
```

---

### **4.4 `EdgeCVSetStatus` (Runtime Dynamic Capture Control)**
```json
{
  "name": "EdgeCVSetStatus",
  "description": "Dynamic control and runtime capture state for Bindu Edge Computer Vision BPR Plugin",
  "emoji": "⏯️",
  "retentionMode": "count",
  "retentionValue": 1,
  "routing": ["local", "child"],
  "properties": [
    { "name": "capture", "type": "boolean", "required": true, "description": "Whether camera frame capture and vision analysis is actively enabled" }
  ]
}
```

---

## **5. Multi-Camera Overlap Arbitration Logic (CEL Reactor)**

When Carlos stands up from the desk, a `DisappearedClass` event arrives. The **CEL Overlap Arbitration Reactor** queries recent sightings from adjacent cameras in SQLite (`v_messages`):

### **5.1 SQLite Zero-RAG Overlap Query**
```sql
SELECT 
    "$id" AS candidate_msg_id,
    json_extract(fields, '$.camera_id') AS cand_camera,
    json_extract(fields, '$.station_id') AS cand_station,
    json_extract(fields, '$.track_id') AS cand_track,
    CAST(json_extract(fields, '$.dwell_duration_seconds') AS REAL) AS cand_dwell,
    CAST(json_extract(fields, '$.max_bbox_area') AS REAL) AS cand_area,
    CAST(json_extract(fields, '$.avg_confidence') AS REAL) AS cand_conf,
    "$when" AS cand_when
FROM v_messages
WHERE "$type" = 'DisappearedClass'
  AND json_extract(fields, '$.camera_id') IN ('cam-tapo-room', 'cam-usb-root')
  AND json_extract(fields, '$.class_name') = 'person'
  AND "$when" >= datetime(:incoming_when, '-30 seconds')
  AND "$when" <= datetime(:incoming_when, '+30 seconds');
```

---

### **5.2 The Pure CEL Authority Formula (`poc_cel_reactor_config.json`)**

In Bindu, the visual authority score is evaluated **purely within Common Expression Language (CEL)** in sub-microseconds without any external script or model invocation:

```cel
// Sub-microsecond CEL Authority Evaluation in Bindu Runtime:
double(payload.max_bbox_area) * double(payload.avg_confidence) * (
  has(payload.bbox_centroid) && size(payload.bbox_centroid) >= 2 
    ? (1.0 - 2.0 * math.sqrt(
        (double(payload.bbox_centroid[0]) - 0.5) * (double(payload.bbox_centroid[0]) - 0.5) + 
        (double(payload.bbox_centroid[1]) - 0.5) * (double(payload.bbox_centroid[1]) - 0.5)
      ))
    : 1.0
)
```

1. **Self-Score vs Candidate-Score:**
   * **`cam-usb-root` Score:** $0.38 \times 0.95 \times 1.0 = \mathbf{0.361} \longrightarrow$ 🏆 **Primary Authority Winner!**
   * **`cam-tapo-room` Score:** $0.07 \times 0.68 \times 0.4 = \mathbf{0.019} \longrightarrow$ 🚫 **Suppressed Supporting View.**
2. **Verdict:** `cam-usb-root` emits **`WorkstationPresenceSession`** with `supporting_camera_ids: ["cam-tapo-room"]`. Exactly **1 session** is counted!

---

## **6. Analytical SQL Queries for Carlos's Workstation**

With `WorkstationPresenceSession` as the single source of truth, Carlos can analyze his desk focus time directly in the Bindu Console:

### **Query 1: How many times did Carlos come to the computer today & total work time?**
```sql
SELECT 
    station_id AS "Workstation",
    COUNT(*) AS "Total Desk Visits",
    ROUND(SUM(dwell_duration_seconds) / 60.0, 1) AS "Total Work Minutes",
    ROUND(AVG(dwell_duration_seconds) / 60.0, 1) AS "Avg Session Minutes",
    ROUND(MAX(dwell_duration_seconds) / 60.0, 1) AS "Longest Focus Session (Min)"
FROM v_messages 
WHERE "$type" = 'WorkstationPresenceSession'
  AND "$when" >= date('now', 'start of day')
GROUP BY station_id;
```

### **Query 2: Is Carlos currently at the computer right now? (Live Desk Occupancy)**
```sql
SELECT 
    json_extract(fields, '$.station_id') AS workstation,
    json_extract(fields, '$.camera_id') AS camera,
    json_extract(fields, '$.dominant_color') AS shirt_color,
    "$when" AS session_started_at
FROM v_messages
WHERE "$type" = 'AppearedClass'
  AND json_extract(fields, '$.class_name') = 'person'
  AND "$id" NOT IN (
      SELECT json_extract(fields, '$.appearance_id') 
      FROM v_messages 
      WHERE "$type" = 'DisappearedClass'
  )
ORDER BY "$when" DESC 
LIMIT 1;
```

### **Query 3: Multi-Camera Deduplication Audit (Duplicates Suppressed)**
```sql
SELECT 
    primary_camera_id,
    COUNT(*) AS total_sessions,
    SUM(json_array_length(fields, '$.supporting_camera_ids')) AS angle_cameras_suppressed
FROM v_messages
WHERE "$type" = 'WorkstationPresenceSession'
GROUP BY primary_camera_id;
```

---

## **7. Architecture Sequence Diagram**

```mermaid
sequenceDiagram
    autonumber
    participant USB as USB Cam (/dev/video0)
    participant Tapo as Tapo IP (192.168.12.177)
    participant BPR as Edge CV BPR Plugin
    participant Core as Bindu Mesh Core (bindu/in)
    participant DB as SQLite Ledger (v_messages)
    participant CEL as Overlap Arbitration Reactor
    participant Console as Bindu Console UI

    Note over USB,Tapo: Carlos sits down at the computer
    USB->>BPR: Ingest front view (Area: 0.38, Conf: 0.95)
    Tapo->>BPR: Ingest angled side view (Area: 0.07, Conf: 0.68)

    BPR->>Core: Emit AppearedClass (cam-usb-root, person, Area: 0.38)
    BPR->>Core: Emit AppearedClass (cam-tapo-room, person, Area: 0.07)

    Note over USB,Tapo: Carlos works for 25 minutes, then stands up
    BPR->>Core: Emit DisappearedClass (cam-usb-root, dwell: 1500s, max_area: 0.38, avg_conf: 0.95)
    BPR->>Core: Emit DisappearedClass (cam-tapo-room, dwell: 1480s, max_area: 0.07, avg_conf: 0.68)
    
    Core->>DB: Persist raw DisappearedClass events

    Note over CEL: CEL Reactor triggers on DisappearedClass (cam-usb-root)
    CEL->>DB: Query concurrent sightings on adjacent cam-tapo-room
    DB-->>CEL: Return concurrent sighting on cam-tapo-room
    Note over CEL: USB Score (0.361) > Tapo Score (0.019) ──► USB Wins Primary Authority!
    CEL->>Core: Emit WorkstationPresenceSession (station: carlos-dev-workstation, dwell: 1500s, duplicates_suppressed: 1)
    Core->>Console: Update Live Workstation Dashboard (1 Visit Counted)
```

---

## **8. Step-by-Step Hands-On Live Testing Protocol**

To verify this tutorial live with your real hardware:

1. **Step 1 — Start the Pipeline:**
   Deploy the MAS Solution bundle or run `./simulate_edge_cv.sh`.
2. **Step 2 — Sit at the Computer:**
   Sit down in your desk chair in front of the USB camera for 30–60 seconds.
3. **Step 3 — Stand Up & Walk Away:**
   Step out of the frame of both cameras for 10 seconds.
4. **Step 4 — Verify Single-Session Count in Console:**
   Open the Bindu Console at `http://localhost:8080/console/` $\rightarrow$ **Dashboards** $\rightarrow$ **💻 Carlos Desk Presence & Focus Monitor**.
   * Verify that **Total Desk Visits = 1** (not 2).
   * Verify that `cam-usb-root` is recorded as `primary_camera_id` and `cam-tapo-room` is listed under `supporting_camera_ids`.

---

## **9. Tutorial Implementation Roadmap & Directory Artifacts**

Location: `public/console/docs/edge-cv-tutorial/`

```
public/console/docs/edge-cv-tutorial/
├── tutorial-plan.md                       # Master engineering & architecture specification
├── tutorial_guide.md                      # Hands-on operator guide with step-by-step CLI commands
├── poc_message_types.json                 # MessageTypes (EdgeCVSet, EdgeCVSetStatus, AppearedClass, DisappearedClass, WorkstationPresenceSession)
├── poc_edge_cv_config.json                # Sample EdgeCVSet with USB on Root and Tapo RTSP stream
├── poc_edge_cv_status.json                # Sample EdgeCVSetStatus runtime capture control envelope
├── poc_cel_reactor_config.json            # CELReactorSet implementing Overlap Authority Arbitration
├── poc_dashboards.json                    # Real-time workstation occupancy & desk session dashboards
├── solution_edge_cv.json                  # MASSolutionSet manifest
├── bundles/
│   └── solution_edge_cv.bundle.json       # 1-Click self-contained MAS Solution Bundle
├── bindu-edge-cv-plugin/                  # Complete Go BPR Plugin Source Tree
│   ├── plugin.json                        # BPR plugin manifest (declares inputs: EdgeCVSet, EdgeCVSetStatus)
│   ├── go.mod                             # Go module configuration
│   ├── main.go                            # Multi-camera ingestion & tracking lifecycle with dynamic capture pause/resume
│   ├── README.md                          # Plugin engineering guide
│   └── examples/                          # Sample in/out message envelopes
│       ├── in/
│       │   ├── EdgeCVSet.json
│       │   └── EdgeCVSetStatus.json
│       └── out/
│           ├── AppearedClass.json
│           └── DisappearedClass.json
├── build_and_certify_plugin.sh            # Script to compile & certify BPR plugin with Bindu Studio
├── publish_capture_status.go              # Go MQTT publisher for dynamic EdgeCVSetStatus messages
├── SetCaptureStatus.sh                    # CLI control simulator script to start/pause capture via MQTT
├── simulate_edge_cv.go                    # Go simulation publisher generating live/synthetic test sequences
├── simulate_edge_cv.sh                    # End-to-end bash runner executing test scenarios
├── deploy_mas_solution.sh                 # 1-Click solution deployer (bindu-studio solution apply)
└── test_frames/                           # Sample test images (carlos_usb_front.jpg, carlos_tapo_side.jpg)
```

---

## **10. Architectural Declaration of Principles: Mastering Hardware to the Bare Metal while Exposing Pure Semantics**

This tutorial is far more than a standard developer "How-To"; it stands as a **definitive architectural manifesto** for edge artificial intelligence.

Bindu proves that enterprise systems do not need to choose between low-level hardware control and high-level AI autonomy:
1. **Mastery Down to the Screws (Bare-Metal Control):**  
   Bindu commands raw Linux video device drivers (`/dev/video0` V4L2 memory buffers), negotiates low-level H.264 RTSP network sockets over TCP (`192.168.12.177:554`), and manages bounded multi-threaded CGo/ONNX worker pools with zero memory leaks.
2. **Elevation to Pure Semantics (AI-First Autonomy):**  
   Instead of exposing fragile C++ pointers, frame buffers, or coordinate matrices to upper layers, Bindu immediately encapsulates physical reality into **pristine, immutable semantic events** (`AppearedClass`, `DisappearedClass`, `WorkstationPresenceSession`).
3. **The AI Stewardship Paradigm:**  
   Because all physical relationships are abstracted into declarative JSON contracts (`EdgeCVSet`), **Autonomous AI Agents (such as Agy)** can design, deploy, audit, and optimize complex multi-camera computer vision networks simply by declaring operational intent. Human engineers and AI agents no longer write fragile video glue-code—they declare the semantic graph, and Bindu manifests it across the physical mesh.

---

## **11. The BPR Vision Core Engine: Hosting YOLO, Ingesting Streams & Emitting Semantics in Go**

The Bindu Plugin Runtime (**BPR**) is the computational engine that bridges raw physical hardware (cameras, GPUs, neural accelerators) with the semantic mesh.

```
+─────────────────────────────────────────────────────────────────────────────────────────────────────────+
|                                    BINDU PLUGIN RUNTIME (BPR) ARCHITECTURE                              |
|                                                                                                         |
|  Bindu Core Process (Host)                                                                              |
|     │                                                                                                   |
|     │ Framed JSON over OS Stdin / Stdout (sdk.RunPlugin)                                                |
|     ▼                                                                                                   |
|  [ bindu-edge-cv-plugin ]  (Isolated OS Process in Go)                                                  |
|     ├── 1. Handshake & Capabilities: In: [EdgeCVSet], Out: [AppearedClass, DisappearedClass]            |
|     ├── 2. Config Bootstrap: Query v_messages for EdgeCVSet or Hot-Reload via OnEvent                   |
|     ├── 3. Camera Capture Loops (Native Goroutines):                                                    |
|     │      ├── cam-usb-root  ──► Linux V4L2 (/dev/video0) via go4vl (5 FPS)                            |
|     │      └── cam-tapo-room ──► RTSP TCP (192.168.12.177:554/stream2) via gortsplib (5 FPS)           |
|     ├── 4. Non-Blocking Channel Pipeline: chan CapturedFrame ──► chan FrameTask                         |
|     ├── 5. Bounded Inference Pool (CGo ONNX Runtime):                                                   |
|     │      ├── libonnxruntime.so (Native C-API, Zero-Python)                                            |
|     │      └── yolox_s.onnx / yolox_nano.onnx (Letterbox 640x640, NMS, Conf: 0.50)                    |
|     ├── 6. 2D Tracking & Debounce Engine (min_presence_frames: 3, lost_debounce_frames: 5)              |
|     └── 7. Semantic Event Emission: ctx.Emit("AppearedClass") / ctx.Emit("DisappearedClass")             |
+─────────────────────────────────────────────────────────────────────────────────────────────────────────+
```

---

### **11.1 BPR Runtime Protocol & Process Isolation**

1. **Memory Isolation:**  
   The vision plugin runs as a dedicated external operating system process (`bindu-edge-cv-plugin`). If an OpenCV driver crashes or a camera bus disconnects violently, Bindu Core and MQTT networking remain 100% operational.
2. **Zero HTTP/gRPC Overhead:**  
   Communication between Bindu Core and the plugin is handled strictly via **Framed JSON over standard input/output (`stdin`/`stdout`)** using `bindunet/sdk`.
3. **Plugin Manifest (`plugin.json`):**
   ```json
   {
     "id": "bindu-edge-cv-plugin",
     "name": "Bindu Edge CV Plugin",
     "version": "1.0.0",
     "runtime": "bpr",
     "in": [
       "EdgeCVSet",
       "EdgeCVSetStatus"
     ],
     "out": [
       "AppearedClass",
       "DisappearedClass"
     ],
     "capabilities": [
       "error:event"
     ]
   }
   ```
4. **Go Entrypoint Contract:**
   ```go
   package main

   import (
       "bindunet/sdk"
       "log"
   )

   func main() {
       plugin, err := NewPlugin()
       if err != nil {
           log.Fatalf("failed to initialize plugin: %v", err)
       }
       if err := sdk.RunPlugin(plugin); err != nil {
           log.Fatalf("plugin execution terminated: %v", err)
       }
   }
   ```

---

### **11.2 Consuming Declarative `EdgeCVSet` & `EdgeCVSetStatus` Controls**

The plugin is 100% declarative and receives both its topology parameters and runtime capture control state through Bindu message contracts:

#### **A. Cold Boot Startup Query (Zero-RAG Local Context):**
On boot, before any network event arrives, the plugin queries SQLite via REST to retrieve its last persisted capture status (`EdgeCVSetStatus`) and configuration (`EdgeCVSet`):
```sql
-- 1. Check if capture is enabled or paused (defaults to true if no record exists)
SELECT fields FROM v_messages 
WHERE "$type" = 'EdgeCVSetStatus' 
ORDER BY "$when" DESC 
LIMIT 1;

-- 2. Fetch active camera topology and model parameters
SELECT fields FROM v_messages 
WHERE "$type" = 'EdgeCVSet' 
  AND json_extract(fields, '$.gateway_node_id') = ? 
ORDER BY "$when" DESC 
LIMIT 1;
```
*If no configuration exists yet, the plugin emits a structured warning (`CV_PLUGIN_CONFIG_MISSING`) and enters a non-blocking "warm-wait" state until a configuration event arrives. If `capture == false`, the plugin initializes dormant without spawning camera loops.*

#### **B. Real-Time Dynamic Hot-Reload & Control (`OnEvent`):**
When an updated `EdgeCVSet` or `EdgeCVSetStatus` is published to the mesh:
```go
func (p *Plugin) OnEvent(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    switch event.MessageType {
    case "EdgeCVSet":
        cfg, err := cvapp.DecodeConfigPayload(event.Payload)
        if err != nil {
            return nil, err
        }
        // Gracefully drain existing camera loops and re-initialize vision engine
        return nil, p.startWithConfig(ctx, cfg)

    case "EdgeCVSetStatus":
        status, err := cvapp.DecodeStatusPayload(event.Payload)
        if err != nil {
            return nil, err
        }
        // Dynamically start or pause frame capture loops across all cameras
        p.setCaptureStatus(ctx, status.Capture)
        return nil, nil
    }
    return nil, nil
}
```

---

### **11.3 Multi-Camera Concurrency Pipeline in Native Go**

The vision engine eliminates Python's Global Interpreter Lock (GIL) by implementing native Go channels and goroutines:

```
[Camera 1: USB /dev/video0] ──► runCameraLoop() ──┐
                                                  ├──► chan CapturedFrame ──► chan FrameTask ──► [Worker 1: ONNX YOLO] ──┐
[Camera 2: Tapo RTSP TCP  ] ──► runCameraLoop() ──┘                                           └── [Worker 2: ONNX YOLO] ──┴─► [Tracking & Debounce]
```

1. **Independent Camera Loops (`runCameraLoop`):**
   * One dedicated goroutine per enabled camera running at its configured cadence (e.g. 5 FPS).
   * **USB Webcam (`cam-usb-root`):** Ingested via native Linux V4L2 buffers (`go4vl`), capturing YUYV/MJPEG frames directly from `/dev/video0`.
   * **Tapo IP Camera (`cam-tapo-room`):** Ingested via RTSP over TCP (`gortsplib`) decoding H.264 video packets into Go `image.Image` buffers.
2. **Bounded Shared Inference Pool (`cvinfer.WorkerPool`):**
   * Camera frames are pushed into a bounded channel `chan FrameTask`.
   * A fixed worker pool of $N$ Go goroutines (configured via `worker_pool_size: 2`) continuously pulls frame tasks and executes neural inference.
   * If cameras generate frames faster than the inference pool can process, frames are intelligently dropped or sample-debounced without growing memory.

---

### **11.4 YOLO Model Hosting via Native ONNX Runtime C-API in Go**

Inference is executed using **ONNX Runtime's native C-API** through CGo, delivering bare-metal execution speed with zero Python runtime overhead.

1. **Runtime Assets Directory Structure:**
   ```
   bindu-cv-plugin/runtime/
   ├── lib/
   │   └── libonnxruntime.so          # Native C shared library (v1.24.1)
   └── models/
       ├── yolox_s.onnx               # Trained YOLOX-s / YOLOX-Nano neural weights
       └── coco_classes.txt           # 80 standard COCO labels (0: person, 2: car, etc.)
   ```

2. **Inference Pipeline Steps:**
   * **Pre-Processing (Go):** Letterbox resize input frame to $640 \times 640$, transpose HWC to CHW format, and normalize RGB pixel values $[0.0..1.0]$.
   * **ONNX Execution (C-API):** Pass tensor to `OrtRun()`, utilizing multi-threaded CPU instructions (AVX2/NEON) or hardware accelerators.
   * **Post-Processing (Go):** Parse output tensor, compute bounding boxes $[x_1, y_1, x_2, y_2]$, filter candidates by `confidence_threshold >= 0.50`, and apply Non-Maximum Suppression (NMS) with IoU threshold $0.45$.

3. **2D Centroid Tracking & Debounce Logic:**
   * **Appearance Verification:** A detected person must be tracked across `min_presence_frames: 3` consecutive frames before emitting `AppearedClass`.
   * **Loss Debounce:** When a person turns around or is temporarily occluded, the tracker holds state for `lost_debounce_frames: 5` before confirming departure.
   * **Color Extraction:** Samples the dominant RGB cluster within the bounding box torso region and classifies it (e.g. `NAVY_BLUE`, `DARK`, `WHITE`).

---

### **11.5 Semantic Event Emission via BPR Protocol**

When tracking thresholds are satisfied, the plugin emits structured events back to Bindu Core:

```go
// 1. Emitting AppearedClass on verified onset:
ctx.Emit("AppearedClass", map[string]any{
    "camera_id":      "cam-usb-root",
    "station_id":     "carlos-dev-workstation",
    "track_id":       "track-usb-501",
    "class_name":     "person",
    "confidence":     0.96,
    "dominant_color": "NAVY_BLUE",
    "bbox_area":      0.392,
    "bbox_centroid":  []float64{0.50, 0.51},
})

// 2. Emitting DisappearedClass on departure:
ctx.Emit("DisappearedClass", map[string]any{
    "camera_id":              "cam-usb-root",
    "station_id":             "carlos-dev-workstation",
    "track_id":               "track-usb-501",
    "class_name":             "person",
    "dwell_duration_seconds": 1500.0,
    "final_dominant_color":   "NAVY_BLUE",
    "avg_confidence":         0.95,
    "max_bbox_area":          0.395,
})
```

---

### **11.6 Complete Build, Certification & Deployment Workflow**

Here is the exact operational procedure to build, certify, and install the Edge CV BPR Plugin on any Linux gateway node.

#### **Step 1: Download Target ONNX Runtime Library**
```bash
# Downloads official ONNX Runtime v1.24.1 into bindu-cv-plugin/runtime/lib/
./scripts/download-bindu-cv-onnxruntime.sh --version 1.24.1
```

#### **Step 2: Prepare & Export YOLOX Model Weights**
```bash
# Exports official YOLOX checkpoint to ONNX format (yolox_s.onnx / yolox_nano.onnx)
./scripts/prepare-bindu-cv-yolox-model.sh
```

#### **Step 3: Certify Plugin Bundle with Bindu Studio**
```bash
# 1. Run Pre-flight Health Check
go run ./cmd/bindu-studio plugin doctor \
  --source ./bindu-cv-plugin \
  --config ./bindu-cv/bindu-cv-message-types.json

# 2. Certify and Package for Target Architecture (e.g. linux/amd64 or linux/arm64)
./scripts/certify-bindu-cv-plugin-linux.sh --target linux/amd64
```
*This produces a cryptographically validated, self-contained plugin tarball at:*  
`dist/bindu-cv-plugin/certified/linux-amd64/bindu-cv-plugin-1.0.0-linux-amd64.tar.gz`

#### **Step 4: Import & Register Plugin to Gateway Node**
```bash
# Import the certified bundle into the cluster registry
go run ./cmd/bindu-studio plugin import \
  --bundle ./dist/bindu-cv-plugin/certified/linux-amd64/bindu-cv-plugin-1.0.0-linux-amd64.tar.gz

# Assign the plugin to the target gateway node (e.g. 'root')
go run ./cmd/bindu-studio plugin register \
  --plugin bindu-cv-plugin \
  --target node \
  --node root
```

#### **Step 5: Rollout to Live Node via Runtime Update Wizard**
```bash
# Stage the desired assignment in mesh desired state
go run ./cmd/bindu-studio runtime desired commit --note "Deploy Edge CV Multi-Camera Plugin"

# Roll out and restart plugin process on target node
go run ./cmd/bindu-studio runtime update wizard --nodes root --yes
```

#### **Step 6: Real-Time Stream Tuning & Verification (`bindu-cv-tune`)**
To test camera feeds, verify bounding box detection, and inspect frame rates in real-time before production rollout:
```bash
# Launch interactive TUI/GUI visual stream tuner
go run -tags "gocv go4vl" ./bindu-cv/cmd/bindu-cv-tune \
  --config ./public/console/docs/edge-cv-tutorial/poc_edge_cv_config.json
```

---

## **12. Centralized Secrets Architecture & RTSP Security Isolation**

### **12.1 Zero-Plaintext Wire Guarantee for Media Storage**
- When `global_settings.snapshot_storage_provider` is set to `"minio_local"`, snapshot binary uploads are decoupled from the event ledger and MQTT mesh.
- The storage provider is defined in `MediaStorageSet` with `"secret_key": "secret:minio_media_secret"`.
- The secret `minio_media_secret` is sealed into the encrypted vault (`accessControlConfig.brs`) using `bindu-studio access-control secrets set minio_media_secret --value "bindu-media-123"`.
- Zero credentials are exposed on the wire or in SQLite message ledgers.

### **12.2 RTSP Stream Credential Governance**
- **Dedicated Low-Privilege Accounts:** Surveillance cameras should always be configured with dedicated RTSP viewer accounts (e.g. `binducv`), restricting access to video streaming only without configuration or PTZ modification privileges.
- **Network Segmentation:** Physical cameras must reside in an isolated IoT/surveillance VLAN without routing to external networks.

