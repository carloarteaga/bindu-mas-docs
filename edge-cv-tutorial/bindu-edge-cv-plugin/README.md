# Bindu Edge CV BPR Plugin (`bindu-edge-cv-plugin`)

The **Bindu Edge CV BPR Plugin** is an external Go process managed by Bindu Core that executes multi-camera video stream ingestion, YOLO neural detection, 2D centroid tracking, and semantic event emission over the standard BPR (Bindu Plugin Runtime) protocol.

---

## 🎯 Architectural Highlights

1. **Memory & Crash Isolation:**
   Runs as an independent OS binary communicating via Framed JSON over `stdin/stdout`. If a camera bus disconnects or a hardware driver fails, Bindu Core stays online.
2. **Declarative Mesh Ingestion:**
   Consumes `EdgeCVSet` messages dynamically. Supports zero-downtime hot-reload (`OnEvent`) and cold-boot startup query (`v_messages`).
3. **Multi-Camera Concurrency:**
   Each camera runs on a dedicated capture goroutine (`runCameraStreamLoop`) feeding frames to a shared bounded inference worker pool.
4. **Debounced Semantic Output:**
   Tracks object presence (`min_presence_frames: 3`) and departures (`lost_debounce_frames: 5`), emitting `AppearedClass` and `DisappearedClass` events.

---

## 📁 Project Structure

```
bindu-edge-cv-plugin/
├── plugin.json                 # BPR manifest declaring inputs, outputs & capabilities
├── go.mod                      # Go module linking to bindunet SDK
├── main.go                     # Complete BPR plugin lifecycle & worker pool
├── capture/                    # Native Go capture engine
│   ├── source.go               # FrameSource interface & dynamic factory
│   ├── v4l2_linux.go           # Linux V4L2 USB camera capture (/dev/video0) via go4vl
│   └── rtsp.go                 # Network IP camera RTSP streaming via gortsplib
├── inference/                  # ONNX Runtime YOLO Detector
│   ├── detector.go             # YOLOX neural network session wrapper
│   ├── preprocess.go           # Letterboxing & RGB normalization
│   ├── postprocess.go          # NMS & HSV dominant color extraction
│   └── types.go                # Detection structs and bounding box formats
├── tracking/                   # 2D Spatial & Temporal Tracking
│   └── tracker.go              # Centroid matching, debounce & duration calculation
├── runtime/                    # Packaged AI runtime assets
│   ├── models/                 # YOLOX-S ONNX model & COCO class names
│   └── lib/                    # Native libonnxruntime.so shared library
└── examples/                   # Sample message contracts
    ├── in/
    │   └── EdgeCVSet.json
    └── out/
        ├── AppearedClass.json
        └── DisappearedClass.json
```

---

---

## 📋 System Prerequisites & Host Dependencies

Before running or deploying `bindu-edge-cv-plugin` on target edge gateways (e.g. `node-dell-1`, Raspberry Pi, or any Debian/Ubuntu Linux host), ensure the following system packages are installed:

### 1. Host OS Packages (Mandatory for RTSP Streams)
```bash
# Update package lists and install FFmpeg (required for RTSP H.264 stream demuxing)
sudo apt update && sudo apt install -y ffmpeg

# Optional: Install Video4Linux utilities if capturing from local USB webcams (/dev/video*)
sudo apt install -y v4l-utils
```

> [!IMPORTANT]
> **Why FFmpeg is required on the host:**  
> The RTSP ingestion worker ([`capture/rtsp.go`](capture/rtsp.go)) uses `ffmpeg` in low-delay pipe mode to demux and decode RTSP H.264 streams into continuous JPEG frames without loading heavy C/C++ video decoding toolchains into the Go runtime. If `ffmpeg` is missing from the host's `PATH`, the plugin emits a `CAMERA_STREAM_UNAVAILABLE` ErrorEvent.
>
> **FFmpeg Version & Timeout Options:**  
> In modern FFmpeg (versions $\ge 5.x / 6.x$ on Ubuntu 22.04+/Debian 12+), socket timeouts are configured with `-timeout <microseconds>` (e.g. `-timeout 5000000` for 5 seconds). The legacy `-stimeout` flag is deprecated/unrecognized.

### 2. Embedded AI Assets (Zero Python / Pip Dependencies)
The plugin is completely self-contained and does **NOT** require Python, PyTorch, or PIP:
* **Inference Engine:** `libonnxruntime.so.1.24.1` downloaded on-demand into `runtime/lib/` via `./setup_runtime.sh`.
* **Neural Weights:** `yolox_s.onnx` downloaded into `runtime/models/` via `./setup_runtime.sh` (`coco_classes.txt` is pre-packaged).
* **Automatic Bootstrap:** `build_and_certify_plugin.sh` automatically invokes `./setup_runtime.sh` if assets are missing.
* **Dynamic Linking:** Standard GNU `glibc` on Linux x86_64 / ARM64.

---

## 🛠️ Build, Certification & 1-Click Deployment

### Option A: 1-Click Complete Pipeline (Recommended)
From the repository root:
```bash
./public/console/docs/edge-cv-tutorial/deploy_mas_solution.sh --yes
```

### Option B: Step-by-Step Manual Certification via Bindu Studio
```bash
# 1. Run Doctor Check
go run ./cmd/bindu-studio plugin doctor \
  --source ./public/console/docs/edge-cv-tutorial/bindu-edge-cv-plugin \
  --config ./public/console/docs/edge-cv-tutorial/poc_message_types.json

# 2. Certify and Build Release Bundle
go run ./cmd/bindu-studio plugin certify \
  --source ./public/console/docs/edge-cv-tutorial/bindu-edge-cv-plugin \
  --config ./public/console/docs/edge-cv-tutorial/poc_message_types.json \
  --out ./public/console/docs/edge-cv-tutorial/bundles/bindu-edge-cv-plugin-1.0.0.tar.gz \
  --target linux/amd64

# 3. Apply Plugin to Target Gateway Node (e.g., node-dell-1)
go run ./cmd/bindu-studio plugin apply \
  --bundle ./public/console/docs/edge-cv-tutorial/bundles/bindu-edge-cv-plugin-1.0.0.tar.gz \
  --target node \
  --node node-dell-1

# 4. Commit Desired State and Rollout via OTA
go run ./cmd/bindu-studio runtime baseline accept --yes
go run ./cmd/bindu-studio runtime desired commit --note "Deploy bindu-edge-cv-plugin"
go run ./cmd/bindu-studio runtime update wizard --yes
```
