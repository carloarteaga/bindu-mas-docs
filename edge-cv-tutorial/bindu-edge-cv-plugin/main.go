// Profile: ai-safe
// Bindu Edge Computer Vision BPR Plugin
// Real V4L2 USB and RTSP Multi-Camera Frame Capture, ONNX/YOLO Inference, 2D Tracking & Semantic Emission.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bindu-plugin/bindu-edge-cv-plugin/capture"
	"bindu-plugin/bindu-edge-cv-plugin/inference"
	"bindu-plugin/bindu-edge-cv-plugin/tracking"
	"bindunet/sdk"
)

type GlobalSettings struct {
	InferenceEngine         string   `json:"inference_engine"`
	ConfidenceThreshold     float64  `json:"confidence_threshold"`
	WorkerPoolSize          int      `json:"worker_pool_size"`
	FrameCadenceFPS         int      `json:"frame_cadence_fps"`
	ClassesOfInterest       []string `json:"classes_of_interest"`
	ColorExtractionEnabled  bool     `json:"color_extraction_enabled"`
	SnapshotStorageProvider string   `json:"snapshot_storage_provider"`
	SnapshotBucket          string   `json:"snapshot_bucket"`
}

type EdgeCVSetStatus struct {
	Capture bool `json:"capture"`
}

type EdgeCVConfig struct {
	GatewayNodeID  string               `json:"gateway_node_id"`
	PluginID       string               `json:"plugin_id"`
	GlobalSettings GlobalSettings       `json:"global_settings"`
	Cameras        []capture.CameraSpec `json:"cameras"`
}

type inferenceWork struct {
	cam      capture.CameraSpec
	img      image.Image
	captured time.Time
}

type Plugin struct {
	mu             sync.Mutex
	config         *EdgeCVConfig
	detector       *inference.Detector
	cancel         context.CancelFunc
	running        bool
	captureEnabled bool
	nodeID         string
	pluginRoot     string
	workChan       chan inferenceWork
	trackers       map[string]*tracking.CameraTracker
	cameraFailures map[string]string // cameraID -> error text
	detFailure     string
}

const (
	errCodeConfigQueryFailed     = "CONFIG_QUERY_FAILED"
	errCodeConfigInvalid         = "CONFIG_INVALID"
	errCodeConfigApplyFailed     = "CONFIG_APPLY_FAILED"
	errCodeDetectorInitFailed    = "DETECTOR_INIT_FAILED"
	errCodeCameraConnectFailed   = "CAMERA_CONNECT_FAILED"
	errCodeCameraStreamFailed    = "CAMERA_STREAM_UNAVAILABLE"
	errCodeInferenceFailed       = "INFERENCE_FAILED"
)

func NewPlugin() (*Plugin, error) {
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	pluginRoot := filepath.Dir(exePath)

	return &Plugin{
		pluginRoot:     pluginRoot,
		captureEnabled: true,
		trackers:       make(map[string]*tracking.CameraTracker),
		cameraFailures: make(map[string]string),
	}, nil
}

func (p *Plugin) Handshake() sdk.HandshakeInfo {
	return sdk.HandshakeInfo{
		Inputs:          []string{"EdgeCVSet", "EdgeCVSetStatus"},
		Outputs:         []string{"AppearedClass", "DisappearedClass"},
		Capabilities:    []string{"error:event"},
		Language:        "go",
		SDKVersion:      "",
		ContractVersion: "bpr/v1",
	}
}

func (p *Plugin) OnStart(ctx *sdk.Context) error {
	p.nodeID = ctx.Info().NodeID
	if p.nodeID == "" {
		p.nodeID = "node-dell-1"
	}
	ctx.Logger().Printf("🚀 Plugin %s booting on node %s. Bootstrapping initial EdgeCVSet & EdgeCVSetStatus...", "bindu-edge-cv-plugin", p.nodeID)
	go p.bootstrapConfig(ctx)
	return nil
}

func (p *Plugin) Shutdown(ctx *sdk.Context) error {
	ctx.Logger().Printf("🛑 Plugin %s shutting down...", "bindu-edge-cv-plugin")
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopCaptureStreams(ctx)
	if p.detector != nil {
		_ = p.detector.Close()
		p.detector = nil
	}
	return nil
}

func (p *Plugin) OnTrigger(ctx *sdk.Context, trigger sdk.Trigger) ([]sdk.Message, error) {
	ctx.Logger().Printf("Trigger %q received (idle/cron)", trigger.ID)
	return nil, nil
}

func (p *Plugin) OnEvent(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	switch strings.TrimSpace(event.MessageType) {
	case "EdgeCVSetStatus":
		ctx.Logger().Printf("📥 Dynamic EdgeCVSetStatus event received: %v", event.Payload)
		captureVal := true
		if capRaw, ok := event.Payload["capture"].(bool); ok {
			captureVal = capRaw
		} else if fields, ok := event.Payload["fields"].(map[string]any); ok {
			if capRaw, ok := fields["capture"].(bool); ok {
				captureVal = capRaw
			}
		}

		p.mu.Lock()
		defer p.mu.Unlock()
		p.setCaptureStatus(ctx, captureVal)
		return nil, nil

	case "EdgeCVSet":
		ctx.Logger().Printf("📥 Dynamic EdgeCVSet event received. Reloading configuration...")
		var cfg EdgeCVConfig
		raw, err := json.Marshal(event.Payload)
		if err != nil {
			p.reportError(ctx, "error", errCodeConfigInvalid, fmt.Sprintf("failed to serialize EdgeCVSet event payload: %v", err), nil)
			return nil, err
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			if fields, ok := event.Payload["fields"].(map[string]any); ok {
				rawFields, _ := json.Marshal(fields)
				_ = json.Unmarshal(rawFields, &cfg)
			}
		}

		if cfg.GatewayNodeID == "" && len(cfg.Cameras) == 0 {
			p.reportError(ctx, "error", errCodeConfigInvalid, "received empty EdgeCVSet configuration payload", nil)
			return nil, fmt.Errorf("empty EdgeCVSet config")
		}

		p.mu.Lock()
		defer p.mu.Unlock()
		if err := p.applyConfiguration(ctx, &cfg); err != nil {
			ctx.Logger().Printf("❌ Error applying EdgeCVSet configuration: %v", err)
			p.reportError(ctx, "error", errCodeConfigApplyFailed, fmt.Sprintf("failed to apply EdgeCVSet configuration: %v", err), map[string]any{
				"gateway_node_id": cfg.GatewayNodeID,
				"camera_count":    len(cfg.Cameras),
			})
			return nil, err
		}
		return nil, nil
	default:
		ctx.Logger().Printf("Unrecognized message type %q ignored", event.MessageType)
	}
	return nil, nil
}

func (p *Plugin) setCaptureStatus(ctx *sdk.Context, captureVal bool) {
	prev := p.captureEnabled
	p.captureEnabled = captureVal

	if captureVal {
		ctx.Logger().Printf("▶️ EdgeCVSetStatus: CAPTURE ENABLED (capture=true)")
		if !prev || !p.running {
			if p.config != nil && len(p.config.Cameras) > 0 {
				ctx.Logger().Printf("🚀 Resuming/starting camera capture streams...")
				p.startCaptureStreams(ctx)
			} else {
				ctx.Logger().Printf("⏳ Capture enabled, waiting for EdgeCVSet camera configuration...")
			}
		}
	} else {
		ctx.Logger().Printf("⏸️ EdgeCVSetStatus: CAPTURE PAUSED (capture=false)")
		if p.running || p.cancel != nil {
			ctx.Logger().Printf("🛑 Pausing active camera capture streams and releasing camera handles...")
			p.stopCaptureStreams(ctx)
		}
	}
}

func (p *Plugin) stopCaptureStreams(ctx *sdk.Context) {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.running = false
	ctx.Logger().Printf("🛑 All camera capture streams stopped.")
}

func (p *Plugin) startCaptureStreams(ctx *sdk.Context) {
	if p.running && p.cancel != nil {
		return
	}
	if p.config == nil || len(p.config.Cameras) == 0 {
		return
	}

	runCtx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	workerCount := p.config.GlobalSettings.WorkerPoolSize
	if workerCount <= 0 {
		workerCount = 2
	}
	p.workChan = make(chan inferenceWork, 20)
	p.trackers = make(map[string]*tracking.CameraTracker)

	// Spawn inference worker pool
	for i := 0; i < workerCount; i++ {
		go p.inferenceWorker(runCtx, ctx, i)
	}
	ctx.Logger().Printf("⚙️ Initialized %d inference workers", workerCount)

	// Initialize Trackers and Camera Capture Goroutines
	fps := p.config.GlobalSettings.FrameCadenceFPS
	if fps <= 0 {
		fps = 1
	}

	activeCount := 0
	for _, cam := range p.config.Cameras {
		if !cam.Enabled {
			ctx.Logger().Printf("Camera [%s] disabled in config, skipping", cam.CameraID)
			continue
		}
		activeCount++
		camSpec := cam

		tracker := tracking.NewCameraTracker(camSpec, tracking.TrackCallbacks{
			OnAppeared: func(t *tracking.TrackedObject, avgConf float64) {
				ctx.Logger().Printf("👁️ [AppearedClass] Verified track %s on %s (class: %s, conf: %.2f, area: %.3f)",
					t.TrackID, t.CameraID, t.ClassName, avgConf, t.MaxBBoxArea)

				if err := ctx.Emit("AppearedClass", map[string]any{
					"camera_id":      t.CameraID,
					"station_id":     t.StationID,
					"track_id":       t.TrackID,
					"class_name":     t.ClassName,
					"confidence":     avgConf,
					"dominant_color": t.DominantColor,
					"bbox_area":      t.MaxBBoxArea,
					"bbox_centroid":  []float64{t.Centroid[0], t.Centroid[1]},
				}); err != nil {
					ctx.Logger().Printf("❌ Failed to emit AppearedClass: %v", err)
					p.reportError(ctx, "warn", "EMIT_FAILED", fmt.Sprintf("failed to emit AppearedClass: %v", err), nil)
				}
			},
			OnDisappeared: func(t *tracking.TrackedObject, dwellDurationSeconds float64, avgConf float64) {
				ctx.Logger().Printf("💨 [DisappearedClass] Track %s left %s (dwell: %.1fs, area: %.3f)",
					t.TrackID, t.CameraID, dwellDurationSeconds, t.MaxBBoxArea)

				if err := ctx.Emit("DisappearedClass", map[string]any{
					"camera_id":              t.CameraID,
					"station_id":             t.StationID,
					"track_id":               t.TrackID,
					"class_name":             t.ClassName,
					"dwell_duration_seconds": dwellDurationSeconds,
					"final_dominant_color":   t.DominantColor,
					"avg_confidence":         avgConf,
					"max_bbox_area":          t.MaxBBoxArea,
				}); err != nil {
					ctx.Logger().Printf("❌ Failed to emit DisappearedClass: %v", err)
				}
			},
		})
		p.trackers[camSpec.CameraID] = tracker

		go p.runCameraCaptureLoop(runCtx, ctx, camSpec, fps)
	}

	ctx.Logger().Printf("📹 Activated %d camera stream(s) at %d FPS cadence", activeCount, fps)
	p.running = true
}

func (p *Plugin) applyConfiguration(ctx *sdk.Context, cfg *EdgeCVConfig) error {
	p.stopCaptureStreams(ctx)

	p.config = cfg
	modelPath, classesPath, libPath := inference.ResolveRuntimePaths(p.pluginRoot)
	ctx.Logger().Printf("🧠 Initializing ONNX Detector with Model=%s, Classes=%s, Lib=%s", modelPath, classesPath, libPath)

	if p.detector == nil {
		det, err := inference.NewDetector(modelPath, classesPath, libPath)
		if err != nil {
			ctx.Logger().Printf("❌ Failed to load ONNX detector: %v", err)
			p.reportError(ctx, "error", errCodeDetectorInitFailed, fmt.Sprintf("failed to load ONNX runtime model: %v", err), map[string]any{
				"model_path":   modelPath,
				"classes_path": classesPath,
				"lib_path":     libPath,
			})
			return fmt.Errorf("detector init: %w", err)
		}
		p.detector = det
		ctx.Logger().Printf("✅ ONNX Detector loaded and initialized successfully")
	}

	if p.captureEnabled {
		p.startCaptureStreams(ctx)
	} else {
		ctx.Logger().Printf("⏸️ EdgeCVSet configuration applied, but frame capture is currently PAUSED (capture=false)")
	}
	return nil
}

func (p *Plugin) runCameraCaptureLoop(runCtx context.Context, ctx *sdk.Context, spec capture.CameraSpec, targetFPS int) {
	for {
		select {
		case <-runCtx.Done():
			ctx.Logger().Printf("Camera capture loop stopped for [%s]", spec.CameraID)
			return
		default:
		}

		ctx.Logger().Printf("🎥 Opening camera [%s] (%s: %s)...", spec.CameraID, spec.SourceKind, spec.SourceRef)
		source, err := capture.OpenCameraSource(spec, targetFPS)
		if err != nil {
			ctx.Logger().Printf("❌ Could not open camera [%s]: %v (retrying in 5s...)", spec.CameraID, err)
			p.handleCameraFailure(ctx, spec.CameraID, spec.SourceRef, err)
			select {
			case <-runCtx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		ctx.Logger().Printf("✅ Camera [%s] active. Capturing at %d FPS.", spec.CameraID, targetFPS)
		interval := time.Second / time.Duration(targetFPS)
		ticker := time.NewTicker(interval)

		streamActive := true
		frameCount := 0
		consecutiveFailures := 0
		for streamActive {
			select {
			case <-runCtx.Done():
				ticker.Stop()
				source.Close()
				ctx.Logger().Printf("Camera loop stopped for [%s]", spec.CameraID)
				return
			case <-ticker.C:
				frameCtx, cancel := context.WithTimeout(runCtx, 4*time.Second)
				img, capturedAt, err := source.CaptureFrame(frameCtx)
				cancel()

				if err != nil {
					consecutiveFailures++
					p.handleCameraFailure(ctx, spec.CameraID, spec.SourceRef, err)
					if consecutiveFailures >= 3 {
						ctx.Logger().Printf("⚠️ [cam %s] %d consecutive capture failures, cycling camera source connection...", spec.CameraID, consecutiveFailures)
						streamActive = false
					}
					continue
				}

				consecutiveFailures = 0
				// Frame successfully captured -> clear camera failure if present
				p.handleCameraRecovery(ctx, spec.CameraID)
				frameCount++

				// Queue work non-blockingly
				select {
				case p.workChan <- inferenceWork{cam: spec, img: img, captured: capturedAt}:
				default:
					ctx.Logger().Printf("⚠️ [cam %s] Inference worker queue full, frame #%d dropped", spec.CameraID, frameCount)
				}
			}
		}
		ticker.Stop()
		source.Close()
		time.Sleep(2 * time.Second)
	}
}

func (p *Plugin) inferenceWorker(runCtx context.Context, ctx *sdk.Context, workerID int) {
	for {
		select {
		case <-runCtx.Done():
			return
		case work, ok := <-p.workChan:
			if !ok {
				return
			}

			if p.detector == nil {
				continue
			}

			p.mu.Lock()
			cfg := p.config
			tracker := p.trackers[work.cam.CameraID]
			p.mu.Unlock()

			if cfg == nil || tracker == nil {
				continue
			}

			opts := inference.InferenceOptions{
				ConfidenceThreshold:    cfg.GlobalSettings.ConfidenceThreshold,
				ClassesOfInterest:      cfg.GlobalSettings.ClassesOfInterest,
				ColorExtractionEnabled: cfg.GlobalSettings.ColorExtractionEnabled,
			}
			if opts.ConfidenceThreshold <= 0 {
				opts.ConfidenceThreshold = 0.40
			}
			if len(opts.ClassesOfInterest) == 0 {
				opts.ClassesOfInterest = []string{"person"}
			}

			start := time.Now()
			dets, err := p.detector.Detect(work.img, opts)
			elapsed := time.Since(start)

			if err != nil {
				ctx.Logger().Printf("❌ Worker %d inference error on [%s]: %v", workerID, work.cam.CameraID, err)
				p.reportError(ctx, "warn", errCodeInferenceFailed, fmt.Sprintf("ONNX inference failed: %v", err), map[string]any{
					"camera_id": work.cam.CameraID,
				})
				continue
			}

			if len(dets) > 0 {
				summary := make([]string, 0, len(dets))
				for _, d := range dets {
					summary = append(summary, fmt.Sprintf("%s(%.2f)", d.ClassName, d.Confidence))
				}
				ctx.Logger().Printf("🎯 [%s] Detected %d object(s) in %v: %s", work.cam.CameraID, len(dets), elapsed, strings.Join(summary, ", "))
			}

			tracker.ProcessDetections(dets, work.captured)
		}
	}
}

func (p *Plugin) handleCameraFailure(ctx *sdk.Context, cameraID, sourceRef string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	errText := err.Error()
	if prev, exists := p.cameraFailures[cameraID]; exists && prev == errText {
		return // Deduplicate: already reported
	}

	p.cameraFailures[cameraID] = errText
	ctx.Logger().Printf("⚠️ Camera [%s] stream failure: %v", cameraID, err)
	p.reportError(ctx, "error", errCodeCameraStreamFailed, fmt.Sprintf("camera [%s] stream unavailable: %v", cameraID, err), map[string]any{
		"camera_id":  cameraID,
		"source_ref": sourceRef,
		"error":      errText,
	})
}

func (p *Plugin) handleCameraRecovery(ctx *sdk.Context, cameraID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.cameraFailures[cameraID]; exists {
		delete(p.cameraFailures, cameraID)
		ctx.Logger().Printf("✅ Camera [%s] stream recovered and streaming frames successfully", cameraID)
	}
}

func (p *Plugin) reportError(ctx *sdk.Context, severity, code, message string, details map[string]any) {
	report := sdk.ErrorReport{
		Severity: strings.TrimSpace(severity),
		Code:     strings.TrimSpace(code),
		Message:  strings.TrimSpace(message),
		Details:  details,
	}
	if report.Details == nil {
		report.Details = make(map[string]any)
	}
	report.Details["plugin"] = "bindu-edge-cv-plugin"
	if nodeID := strings.TrimSpace(p.nodeID); nodeID != "" {
		report.Details["node_id"] = nodeID
	}

	if err := ctx.ReportError(report); err != nil {
		ctx.Logger().Printf("⚠️ Failed to deliver structured error report (code=%s): %v", code, err)
	}
}

func (p *Plugin) bootstrapConfig(ctx *sdk.Context) {
	nodeID := p.nodeID
	if nodeID == "" {
		nodeID = ctx.Info().NodeID
	}
	if nodeID == "" {
		nodeID = "node-dell-1"
	}

	// 1. Query EdgeCVSetStatus (Check if capture state was persisted)
	ctx.Logger().Printf("🔍 Checking persisted EdgeCVSetStatus for node=%s...", nodeID)
	statusSQL := `SELECT fields FROM v_messages 
		WHERE "$type" = 'EdgeCVSetStatus' 
		ORDER BY "$when" DESC LIMIT 1`

	statusRows, err := ctx.QueryReadOnly(context.Background(), statusSQL)
	if err == nil && len(statusRows) > 0 {
		var status EdgeCVSetStatus
		if fMap, ok := statusRows[0]["fields"].(map[string]any); ok {
			raw, _ := json.Marshal(fMap)
			_ = json.Unmarshal(raw, &status)
		} else if fStr, ok := statusRows[0]["fields"].(string); ok {
			_ = json.Unmarshal([]byte(fStr), &status)
		}
		p.mu.Lock()
		p.captureEnabled = status.Capture
		p.mu.Unlock()
		ctx.Logger().Printf("📋 Found persisted EdgeCVSetStatus: capture=%v", status.Capture)
	} else {
		// Rule 3: If no EdgeCVSetStatus exists, default to capture=true
		p.mu.Lock()
		p.captureEnabled = true
		p.mu.Unlock()
		ctx.Logger().Printf("ℹ️ No persisted EdgeCVSetStatus found. Defaulting capture=true.")
	}

	// 2. Query EdgeCVSet (Cameras & Model topology)
	ctx.Logger().Printf("🔍 Querying persisted EdgeCVSet configuration for gateway=%s...", nodeID)
	sql := `SELECT fields FROM v_messages 
		WHERE "$type" = 'EdgeCVSet' 
		AND json_extract(fields, '$.gateway_node_id') = ? 
		ORDER BY "$when" DESC LIMIT 1`

	rows, err := ctx.QueryReadOnly(context.Background(), sql, nodeID)
	if err != nil {
		ctx.Logger().Printf("❌ Startup EdgeCVSet query failed: %v", err)
		p.reportError(ctx, "error", errCodeConfigQueryFailed, fmt.Sprintf("startup EdgeCVSet SQL query failed: %v", err), map[string]any{
			"node_id": nodeID,
		})
		return
	}
	if len(rows) == 0 {
		ctx.Logger().Printf("⏳ No persisted EdgeCVSet found for gateway %s (waiting for incoming config event)...", nodeID)
		return
	}

	var cfg EdgeCVConfig
	if fieldsRaw, ok := rows[0]["fields"].(map[string]any); ok {
		raw, _ := json.Marshal(fieldsRaw)
		_ = json.Unmarshal(raw, &cfg)
	} else if fieldsStr, ok := rows[0]["fields"].(string); ok {
		_ = json.Unmarshal([]byte(fieldsStr), &cfg)
	}

	if len(cfg.Cameras) > 0 {
		ctx.Logger().Printf("📋 Loaded EdgeCVSet with %d camera(s) for node %s. Initializing...", len(cfg.Cameras), nodeID)
		p.mu.Lock()
		defer p.mu.Unlock()
		if err := p.applyConfiguration(ctx, &cfg); err != nil {
			ctx.Logger().Printf("❌ Failed to apply startup EdgeCVSet: %v", err)
			p.reportError(ctx, "error", errCodeConfigApplyFailed, fmt.Sprintf("failed to apply startup configuration: %v", err), nil)
		}
	} else {
		ctx.Logger().Printf("⚠️ persisted EdgeCVSet has 0 cameras declared")
	}
}

func main() {
	plugin, err := NewPlugin()
	if err != nil {
		log.Fatalf("Failed to initialize plugin: %v", err)
	}
	if err := sdk.Run(plugin); err != nil {
		fmt.Fprintf(os.Stderr, "Plugin runtime exited: %v\n", err)
		os.Exit(1)
	}
}
