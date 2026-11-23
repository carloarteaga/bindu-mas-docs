//go:build ignore

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

type genericMessage struct {
	Type     string         `json:"$type"`
	ID       string         `json:"$id"`
	Group    string         `json:"$group"`
	Producer string         `json:"$producer"`
	User     string         `json:"$user"`
	When     string         `json:"$when"`
	Fields   map[string]any `json:"fields"`
}

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 1885, "MQTT Broker Port")
	user := flag.String("user", "console", "MQTT Username")
	pass := flag.String("pass", "StrongPass123", "MQTT Password")
	interactive := flag.Bool("interactive", false, "Pause between simulation steps")
	flag.Parse()

	fmt.Println("================================================================================")
	fmt.Println("👁️ BINDU EDGE CV: Multi-Camera Presence & Visual Authority Simulator")
	fmt.Println("================================================================================")
	fmt.Printf("MQTT Broker: ssl://%s:%d | User: %s\n", *host, *port, *user)
	fmt.Println("Scenario:    Carlos arrives at Workstation, observed concurrently by:")
	fmt.Println("             1. cam-usb-root  (/dev/video0 on Root - Direct Frontal View)")
	fmt.Println("             2. cam-tapo-room (192.168.12.177 - Room Ambient Angled View)")
	fmt.Println("Goal:        Prove CEL Reactor overlap arbitration suppresses false double-counting")
	fmt.Println("             and outputs 1 authoritative WorkstationPresenceSession.")
	fmt.Println("--------------------------------------------------------------------------------")

	// 1. Connect to MQTT Broker
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	if *user != "" {
		opts.SetUsername(*user)
		opts.SetPassword(*pass)
	}
	opts.SetClientID(fmt.Sprintf("bindu-edge-cv-sim-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintf(os.Stderr, "❌ Error connecting to MQTT broker: %v\n", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)
	fmt.Println("✅ Connected to Bindu MQTT Secure Broker.")

	publishMsg := func(msg genericMessage, stepDesc string) {
		payload, err := json.Marshal(msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal JSON: %v\n", err)
			return
		}
		token := client.Publish("bindu/in", 1, false, payload)
		token.Wait()
		if token.Error() != nil {
			fmt.Fprintf(os.Stderr, "❌ Error publishing: %v\n", token.Error())
		} else {
			fmt.Printf("  📡 [%s] Published %s ($id: %s, $producer: %s)\n", stepDesc, msg.Type, msg.ID, msg.Producer)
		}
	}

	stepWait := func(seconds int, label string) {
		if *interactive {
			fmt.Printf("\n[Press Enter to continue: %s...]", label)
			fmt.Scanln()
		} else {
			fmt.Printf("   ⏳ Waiting %d seconds (%s)...\n", seconds, label)
			time.Sleep(time.Duration(seconds) * time.Second)
		}
	}

	// STEP 1: Publish EdgeCVSet Configuration Message
	fmt.Println("\n--- STEP 1: Ingesting EdgeCVSet Declarative Multi-Camera Mesh Config ---")
	edgeCVConfigMsg := genericMessage{
		Type:     "EdgeCVSet",
		ID:       fmt.Sprintf("cfg-edge-cv-%d", time.Now().UnixNano()),
		Group:    "config",
		Producer: "root",
		User:     "admin",
		When:     time.Now().UTC().Format(time.RFC3339),
		Fields: map[string]any{
			"gateway_node_id": "root",
			"plugin_id":       "bindu-edge-cv-plugin",
			"global_settings": map[string]any{
				"inference_engine":          "onnx_yolox_nano",
				"confidence_threshold":      0.50,
				"worker_pool_size":          2,
				"frame_cadence_fps":         5,
				"classes_of_interest":       []string{"person"},
				"color_extraction_enabled":  true,
				"snapshot_storage_provider": "minio_local",
				"snapshot_bucket":           "cv-snapshots",
			},
			"cameras": []map[string]any{
				{
					"camera_id":            "cam-usb-root",
					"station_id":           "carlos-dev-workstation",
					"name":                 "Local USB Webcam on Root (V4L2)",
					"source_kind":          "usb",
					"source_ref":           "/dev/video0",
					"adjacent_camera_ids":  []string{"cam-tapo-room"},
					"min_presence_frames":  3,
					"lost_debounce_frames": 5,
					"enabled":              true,
				},
				{
					"camera_id":            "cam-tapo-room",
					"station_id":           "room-ambient-overview",
					"name":                 "TP-Link Tapo IP Camera at Angle (RTSP TCP/IP)",
					"source_kind":          "tcp_ip",
					"source_ref":           "rtsp://binducv:abubindu@192.168.12.177:554/stream2",
					"rtsp_transport":       "tcp",
					"adjacent_camera_ids":  []string{"cam-usb-root"},
					"min_presence_frames":  3,
					"lost_debounce_frames": 5,
					"enabled":              true,
				},
			},
		},
	}
	publishMsg(edgeCVConfigMsg, "Config Sync")

	stepWait(2, "Synchronizing Camera Stream Orchestrator")

	// STEP 2: Tapo Angled IP Camera Sighting (Person Enters Room)
	fmt.Println("\n--- STEP 2: [T+0s] cam-tapo-room Sighting (Angled Room View) ---")
	tapoTrackID := fmt.Sprintf("track-tapo-%s", uuid.NewString()[:6])
	tapoAppeared := genericMessage{
		Type:     "AppearedClass",
		ID:       fmt.Sprintf("sight-tapo-%d", time.Now().UnixNano()),
		Group:    "mesh",
		Producer: "cam-tapo-room",
		User:     "bindu-edge-cv-plugin",
		When:     time.Now().UTC().Format(time.RFC3339),
		Fields: map[string]any{
			"camera_id":      "cam-tapo-room",
			"station_id":     "room-ambient-overview",
			"track_id":       tapoTrackID,
			"class_name":     "person",
			"confidence":     0.74,
			"dominant_color": "NAVY_BLUE",
			"bbox_area":      0.068, // Small area (angled / distant view)
			"bbox_centroid":  []float64{0.32, 0.45},
		},
	}
	publishMsg(tapoAppeared, "Tapo Room Sighting")

	stepWait(2, "Carlos moves to workstation desk")

	// STEP 3: USB Frontal Webcam Sighting (Carlos Sits Down in Front of Workstation)
	fmt.Println("\n--- STEP 3: [T+2s] cam-usb-root Sighting (Direct Frontal Workstation View) ---")
	usbTrackID := fmt.Sprintf("track-usb-%s", uuid.NewString()[:6])
	usbAppeared := genericMessage{
		Type:     "AppearedClass",
		ID:       fmt.Sprintf("sight-usb-%d", time.Now().UnixNano()),
		Group:    "mesh",
		Producer: "cam-usb-root",
		User:     "bindu-edge-cv-plugin",
		When:     time.Now().UTC().Format(time.RFC3339),
		Fields: map[string]any{
			"camera_id":      "cam-usb-root",
			"station_id":     "carlos-dev-workstation",
			"track_id":       usbTrackID,
			"class_name":     "person",
			"confidence":     0.96,
			"dominant_color": "NAVY_BLUE",
			"bbox_area":      0.392, // Large area (close frontal view = Primary Authority)
			"bbox_centroid":  []float64{0.50, 0.51},
		},
	}
	publishMsg(usbAppeared, "USB Frontal Sighting")

	stepWait(6, "Carlos works at computer (Focus dwell period)")

	// STEP 4: Session Exit & Overlap Arbitration
	fmt.Println("\n--- STEP 4: [T+8s] Departure & Deterministic CEL Authority Arbitration ---")
	// 4a. cam-tapo-room emits DisappearedClass
	tapoDisappeared := genericMessage{
		Type:     "DisappearedClass",
		ID:       fmt.Sprintf("exit-tapo-%d", time.Now().UnixNano()),
		Group:    "mesh",
		Producer: "cam-tapo-room",
		User:     "bindu-edge-cv-plugin",
		When:     time.Now().UTC().Format(time.RFC3339),
		Fields: map[string]any{
			"camera_id":              "cam-tapo-room",
			"station_id":             "room-ambient-overview",
			"track_id":               tapoTrackID,
			"class_name":             "person",
			"dwell_duration_seconds": 8.5,
			"final_dominant_color":   "NAVY_BLUE",
			"avg_confidence":         0.73,
			"max_bbox_area":          0.072,
		},
	}
	publishMsg(tapoDisappeared, "Tapo Departure")

	time.Sleep(500 * time.Millisecond)

	// 4b. cam-usb-root emits DisappearedClass (Direct Desk Authority)
	usbDisappeared := genericMessage{
		Type:     "DisappearedClass",
		ID:       fmt.Sprintf("exit-usb-%d", time.Now().UnixNano()),
		Group:    "mesh",
		Producer: "cam-usb-root",
		User:     "bindu-edge-cv-plugin",
		When:     time.Now().UTC().Format(time.RFC3339),
		Fields: map[string]any{
			"camera_id":              "cam-usb-root",
			"station_id":             "carlos-dev-workstation",
			"track_id":               usbTrackID,
			"class_name":             "person",
			"dwell_duration_seconds": 7.0,
			"final_dominant_color":   "NAVY_BLUE",
			"avg_confidence":         0.96,
			"max_bbox_area":          0.395,
		},
	}
	publishMsg(usbDisappeared, "USB Desk Departure")

	fmt.Println("\n================================================================================")
	fmt.Println("🎉 Simulation Sequence Complete!")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Expected CEL Reactor Evaluation Results:")
	fmt.Println("  1. cam-usb-root AuthorityScore:  0.395 × 0.96 = 0.3792 (PRIMARY WINNER 🏆)")
	fmt.Println("  2. cam-tapo-room AuthorityScore: 0.072 × 0.73 = 0.0525 (SUPPORTING SIGHTING)")
	fmt.Println("  3. Emitted Message: 'WorkstationPresenceSession' on carlos-dev-workstation")
	fmt.Println("     - primary_camera_id:    'cam-usb-root'")
	fmt.Println("     - supporting_camera_ids: ['cam-tapo-room']")
	fmt.Println("     - dwell_duration_seconds: 7.0")
	fmt.Println("================================================================================")
}
