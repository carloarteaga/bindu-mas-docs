// Profile: ai-safe
// Bindu BLE Presence & Spatial Geofence BPR Plugin
// Supervised edge worker scanning physical Bluetooth Low Energy advertising frames (tinygo.org/x/bluetooth)
// and emitting BLEBeaconSeen and BLEBeaconLost semantic events enriched with spatial context.

package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"bindunet/sdk"
	"tinygo.org/x/bluetooth"
)

const (
	appleCompanyID = 0x004c
)

type BLEPlugin struct {
	mu                sync.Mutex
	gatewayNodeID     string
	adapterName       string
	scannerBackend    string
	presenceWindowMS  int
	absenceWindowMS   int
	referenceZone     string
	referenceLocagram string
	zoneCenterX       float64
	zoneCenterY       float64

	engine   *PresenceEngine
	adapter  *bluetooth.Adapter
	scanning bool
	stopChan chan struct{}
}

func NewBLEPlugin() *BLEPlugin {
	nodeID := os.Getenv("BINDU_NODE_ID")
	if nodeID == "" {
		nodeID = "node-dell-2"
	}

	backend := os.Getenv("BLE_SCANNER_BACKEND")
	if backend == "" {
		backend = "tinygo"
	}

	return &BLEPlugin{
		gatewayNodeID:     nodeID,
		adapterName:       "hci0",
		scannerBackend:    backend,
		presenceWindowMS:  1000,
		absenceWindowMS:   5000,
		referenceZone:     "operation-room",
		referenceLocagram: "/root/node-dell-2/operation-room",
		zoneCenterX:       6.0,
		zoneCenterY:       4.0,
		stopChan:          make(chan struct{}),
	}
}

func (p *BLEPlugin) Handshake() sdk.HandshakeInfo {
	return sdk.HandshakeInfo{
		Inputs: []string{
			"BLEPluginConfig",
		},
		Outputs: []string{
			"BLEBeaconSeen",
			"BLEBeaconLost",
		},
		Capabilities: []string{
			"ble-scanner",
			"presence-detection",
			"spatial-where-geofence",
		},
		Language:        "go",
		SDKVersion:      "bpr-sdk-go/0.1.0",
		ContractVersion: "bpr/v1",
	}
}

func (p *BLEPlugin) OnStart(ctx *sdk.Context) error {
	ctx.Logger().Printf("[BLE Plugin] Starting BLE presence scanner on gateway %s (Backend: %s)...",
		p.gatewayNodeID, p.scannerBackend)

	// Initialize the Presence State Machine
	p.engine = NewPresenceEngine(p.presenceWindowMS, p.absenceWindowMS, func(trans PresenceTransition) {
		p.handleTransition(ctx, trans)
	})

	// Spawn the 1Hz absence sweeper ticker
	go p.runSweeperTicker(ctx)

	// Start Bluetooth Scanner
	go p.runScanner(ctx)

	return nil
}

func (p *BLEPlugin) OnTrigger(ctx *sdk.Context, trigger sdk.Trigger) ([]sdk.Message, error) {
	ctx.Logger().Printf("[BLE Plugin] Trigger received: %s (%s)", trigger.ID, trigger.Reason)
	return nil, nil
}

func (p *BLEPlugin) OnEvent(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	switch event.MessageType {
	case "BLEPluginConfig":
		return p.handleConfig(ctx, event)
	default:
		ctx.Logger().Printf("[BLE Plugin] Unhandled message type: %s", event.MessageType)
		return nil, nil
	}
}

func (p *BLEPlugin) handleConfig(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fields := event.Payload
	if gw, ok := fields["gateway_node_id"].(string); ok && gw != "" {
		p.gatewayNodeID = gw
	}
	if b, ok := fields["scanner_backend"].(string); ok && b != "" {
		p.scannerBackend = b
	}
	if pw, ok := fields["presence_window_ms"].(float64); ok && pw > 0 {
		p.presenceWindowMS = int(pw)
	}
	if aw, ok := fields["absence_window_ms"].(float64); ok && aw > 0 {
		p.absenceWindowMS = int(aw)
	}
	if rz, ok := fields["reference_zone"].(string); ok && rz != "" {
		p.referenceZone = rz
	}
	if rl, ok := fields["reference_locagram"].(string); ok && rl != "" {
		p.referenceLocagram = rl
	}
	if x, ok := fields["zone_center_x"].(float64); ok {
		p.zoneCenterX = x
	}
	if y, ok := fields["zone_center_y"].(float64); ok {
		p.zoneCenterY = y
	}

	ctx.Logger().Printf("[BLE Plugin] Configuration updated: PresenceWindow=%dms, AbsenceWindow=%dms, Zone=%s (%s)",
		p.presenceWindowMS, p.absenceWindowMS, p.referenceZone, p.referenceLocagram)

	return nil, nil
}

func (p *BLEPlugin) handleTransition(ctx *sdk.Context, trans PresenceTransition) {
	nowStr := time.Now().UTC().Format(time.RFC3339)

	if trans.Type == "SEEN" {
		payload := map[string]any{
			"beacon_id":           trans.BeaconID,
			"protocol":            trans.Protocol,
			"gateway_node_id":     p.gatewayNodeID,
			"zone_id":             p.referenceZone,
			"status":              "SEEN",
			"seen_at":             trans.SeenAt.Format(time.RFC3339),
			"rssi_raw":            trans.RSSIRaw,
			"rssi_filtered":       trans.RSSIFiltered,
			"distance_estimate_m": trans.DistanceEstimateM,
			"tx_power":            trans.TxPower,
			"ibeacon_uuid":        trans.UUID,
			"ibeacon_major":       trans.Major,
			"ibeacon_minor":       trans.Minor,
			"observation_count":   trans.ObservationCount,
			"scanner_backend":     p.scannerBackend,
		}

		ctx.Logger().Printf("🟢 [BLE SEEN] Tag: %s | RSSI: %.1f dBm | Dist: %.2fm | Zone: %s",
			trans.BeaconID, trans.RSSIFiltered, trans.DistanceEstimateM, p.referenceZone)

		if err := ctx.Emit("BLEBeaconSeen", payload); err != nil {
			ctx.Logger().Printf("[BLE Plugin] Emit error for BLEBeaconSeen: %v", err)
		}
	} else if trans.Type == "LOST" {
		payload := map[string]any{
			"beacon_id":          trans.BeaconID,
			"protocol":           trans.Protocol,
			"gateway_node_id":    p.gatewayNodeID,
			"zone_id":            p.referenceZone,
			"status":             "LOST",
			"last_seen_at":       trans.LastSeenAt.Format(time.RFC3339),
			"lost_at":            trans.LostAt.Format(time.RFC3339),
			"absence_window_ms":  trans.AbsenceWindowMS,
			"last_rssi_filtered": trans.RSSIFiltered,
			"reason":             trans.Reason,
			"ibeacon_uuid":       trans.UUID,
			"ibeacon_major":      trans.Major,
			"ibeacon_minor":      trans.Minor,
		}

		ctx.Logger().Printf("🔴 [BLE LOST] Tag: %s | Reason: %s (Silence > %dms) | Last Seen: %s",
			trans.BeaconID, trans.Reason, trans.AbsenceWindowMS, trans.LastSeenAt.Format(time.RFC3339))

		_ = nowStr
		if err := ctx.Emit("BLEBeaconLost", payload); err != nil {
			ctx.Logger().Printf("[BLE Plugin] Emit error for BLEBeaconLost: %v", err)
		}
	}
}

func (p *BLEPlugin) runSweeperTicker(ctx *sdk.Context) {
	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case t := <-ticker.C:
			if p.engine != nil {
				p.engine.Sweep(t)
			}
		}
	}
}

func (p *BLEPlugin) runScanner(ctx *sdk.Context) {
	if p.scannerBackend == "mock" {
		p.runMockScanner(ctx)
		return
	}

	adapter := bluetooth.DefaultAdapter
	p.adapter = adapter

	if err := adapter.Enable(); err != nil {
		ctx.Logger().Printf("⚠️ [BLE Plugin] bluetooth.Enable() failed: %v. Falling back to mock scanner.", err)
		p.runMockScanner(ctx)
		return
	}

	ctx.Logger().Printf("📡 [BLE Plugin] Bluetooth radio adapter enabled on %s. Starting scan...", p.gatewayNodeID)
	p.scanning = true

	err := adapter.Scan(func(_ *bluetooth.Adapter, result bluetooth.ScanResult) {
		for _, mData := range result.AdvertisementPayload.ManufacturerData() {
			if mData.CompanyID != appleCompanyID || len(mData.Data) < 23 {
				continue
			}
			// iBeacon prefix check: 0x02, 0x15
			if mData.Data[0] != 0x02 || mData.Data[1] != 0x15 {
				continue
			}

			uuid := formatUUID(mData.Data[2:18])
			major := uint16(mData.Data[18])<<8 | uint16(mData.Data[19])
			minor := uint16(mData.Data[20])<<8 | uint16(mData.Data[21])
			txPower := int(int8(mData.Data[22]))

			beaconID := fmt.Sprintf("ibeacon:%s:%d:%d", uuid, major, minor)

			obs := RawObservation{
				BeaconID: beaconID,
				Protocol: "ibeacon",
				RSSI:     int(result.RSSI),
				TxPower:  txPower,
				UUID:     uuid,
				Major:    major,
				Minor:    minor,
				SeenAt:   time.Now().UTC(),
			}

			p.engine.Ingest(obs)
		}
	})

	if err != nil {
		ctx.Logger().Printf("⚠️ [BLE Plugin] Scan error: %v", err)
	}
}

func (p *BLEPlugin) runMockScanner(ctx *sdk.Context) {
	ctx.Logger().Printf("🧪 [BLE Plugin] Running in mock scanner fallback mode")
	<-p.stopChan
}

func formatUUID(b []byte) string {
	if len(b) != 16 {
		return hex.EncodeToString(b)
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (p *BLEPlugin) Shutdown(ctx *sdk.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.stopChan)
	if p.adapter != nil && p.scanning {
		_ = p.adapter.StopScan()
		p.scanning = false
	}
	ctx.Logger().Printf("[BLE Plugin] Scanner stopped cleanly")
	return nil
}

func main() {
	plugin := NewBLEPlugin()
	if err := sdk.Run(plugin); err != nil {
		log.Fatalf("BLE plugin terminated: %v", err)
	}
}
