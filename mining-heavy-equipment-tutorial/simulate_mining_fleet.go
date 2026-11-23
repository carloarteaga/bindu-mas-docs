// Profile: ai-safe
// Bindu Mining & Heavy Equipment Fleet Vibration Simulator
// Demonstrates non-invasive triaxial BLE beacon sampling, drive-by circular buffer bursts,
// true utilization state transitions, mechanical abuse shocks, and bearing defect degradation.

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type VibrationSampleFields struct {
	MachineID         string  `json:"machine_id"`
	MachineModel      string  `json:"machine_model"`
	MachineType       string  `json:"machine_type"`
	OperatingState    string  `json:"operating_state"`
	VibrationRMS      float64 `json:"vibration_rms"`
	PeakG             float64 `json:"peak_g"`
	CrestFactor       float64 `json:"crest_factor"`
	Kurtosis          float64 `json:"kurtosis"`
	SampleTimestamp   string  `json:"sample_timestamp"`
	BatteryMillivolts int     `json:"battery_millivolts"`
	BeaconID          string  `json:"beacon_id"`
	IngestMode        string  `json:"ingest_mode"`
}

type BinduMessageEnvelope struct {
	Type     string `json:"$type"`
	ID       string `json:"$id"`
	When     string `json:"$when"`
	Group    string `json:"$group"`
	Producer string `json:"$producer"`
	User     string `json:"$user"`
	NodeDest string `json:"$node-dest"`
	Fields   any    `json:"fields"`
}

func main() {
	burstMode := flag.Bool("burst", false, "Simulate drive-by store-and-forward batch burst (30 1-minute samples)")
	abuseScenario := flag.Bool("abuse", false, "Simulate severe mechanical shock impact (Komatsu HD785, peak_g=10.4g)")
	defectScenario := flag.Bool("defect", false, "Simulate incipient hydraulic pump bearing defect (Epiroc HB-02, Crest=6.0, Kurtosis=4.85)")
	allScenarios := flag.Bool("all", false, "Simulate complete multi-machine operational sequence")
	dryRun := flag.Bool("dry-run", false, "Print generated JSON event payloads to stdout without publishing to MQTT")
	brokerURL := flag.String("broker", "tls://127.0.0.1:1888", "MQTT broker endpoint")
	clientCert := flag.String("cert", "", "Path to client certificate file (mTLS)")
	clientKey := flag.String("key", "", "Path to client private key file (mTLS)")
	caCert := flag.String("ca", "", "Path to CA certificate file")
	baseDir := flag.String("base-dir", "", "Bindu base directory for resolving TLS certificates")
	flag.Parse()

	// Default to all scenarios if no specific flag chosen
	if !*burstMode && !*abuseScenario && !*defectScenario {
		*allScenarios = true
	}

	fmt.Println("=================================================================")
	fmt.Println("🚜 BINDU HEAVY EQUIPMENT VIBRATION & FLEET TELEMETRY SIMULATOR")
	fmt.Println("=================================================================")

	var client mqtt.Client
	if !*dryRun {
		var err error
		client, err = initMQTTClient(*brokerURL, *baseDir, *clientCert, *clientKey, *caCert)
		if err != nil {
			fmt.Printf("⚠️ MQTT Connection Warning: %v\n", err)
			fmt.Println("👉 Falling back to DRY-RUN mode (printing generated messages to stdout)...")
			*dryRun = true
		} else {
			defer client.Disconnect(250)
			fmt.Println("✅ Connected to Bindu Sovereign Mesh MQTT Broker (topic: bindu/in)")
		}
	}

	publishEvent := func(msg BinduMessageEnvelope) {
		data, err := json.MarshalIndent(msg, "", "  ")
		if err != nil {
			fmt.Printf("❌ Serialization error: %v\n", err)
			return
		}

		if *dryRun {
			fmt.Printf("\n📦 [DRY-RUN SIMULATION] Message: %s (Producer: %s)\n", msg.Type, msg.Producer)
			fmt.Println(string(data))
		} else {
			token := client.Publish("bindu/in", 1, false, data)
			token.Wait()
			if token.Error() != nil {
				fmt.Printf("❌ Publish error: %v\n", token.Error())
			} else {
				fmt.Printf("  • Published %s for %s at %s\n", msg.Type, msg.Producer, msg.When)
			}
		}
	}

	now := time.Now().UTC()

	// 1. SCENARIO A: Normal Operation & Warm-up (CAT-390F-01 Excavator)
	if *allScenarios {
		fmt.Println("\n--- Scenario 1: CAT-390F-01 Normal Digging (Nominal Vibration) ---")
		msg := BinduMessageEnvelope{
			Type:     "MachineVibrationSample",
			ID:       fmt.Sprintf("vib-cat-%d", now.UnixNano()),
			When:     now.Format(time.RFC3339),
			Group:    "mesh",
			Producer: "CAT-390F-01",
			User:     "simulator",
			NodeDest: "root",
			Fields: VibrationSampleFields{
				MachineID:         "CAT-390F-01",
				MachineModel:      "CAT 390F Hydraulic Excavator",
				MachineType:       "EXCAVATOR",
				OperatingState:    "WORKING",
				VibrationRMS:      2.12,
				PeakG:             6.45,
				CrestFactor:       3.04,
				Kurtosis:          3.18,
				SampleTimestamp:   now.Format(time.RFC3339),
				BatteryMillivolts: 3620,
				BeaconID:          "BLE-CAT-01",
				IngestMode:        "REALTIME",
			},
		}
		publishEvent(msg)
	}

	// 2. SCENARIO B: Structural Shock & Abuse (Komatsu HD785 Haul Truck)
	if *allScenarios || *abuseScenario {
		fmt.Println("\n--- Scenario 2: KOMATSU-HD785-03 Severe Impact Shock (Boulder Drop in Bed) ---")
		impactTime := now.Add(2 * time.Minute)
		msg := BinduMessageEnvelope{
			Type:     "MachineVibrationSample",
			ID:       fmt.Sprintf("vib-komatsu-%d", impactTime.UnixNano()),
			When:     impactTime.Format(time.RFC3339),
			Group:    "mesh",
			Producer: "KOMATSU-HD785-03",
			User:     "simulator",
			NodeDest: "root",
			Fields: VibrationSampleFields{
				MachineID:         "KOMATSU-HD785-03",
				MachineModel:      "Komatsu HD785 Haul Truck",
				MachineType:       "HAUL_TRUCK",
				OperatingState:    "WORKING",
				VibrationRMS:      2.15,
				PeakG:             10.42, // Breaches 8.5g -> CEL Gating triggers EquipmentAbuseIncident
				CrestFactor:       4.85,
				Kurtosis:          3.95,
				SampleTimestamp:   impactTime.Format(time.RFC3339),
				BatteryMillivolts: 3590,
				BeaconID:          "BLE-KOM-03",
				IngestMode:        "REALTIME",
			},
		}
		publishEvent(msg)
	}

	// 3. SCENARIO C: Incipient Hydraulic Pump Bearing Defect (Epiroc HB-02 Rock Breaker)
	if *allScenarios || *defectScenario {
		fmt.Println("\n--- Scenario 3: EPIROC-HB-02 Incipient Bearing Defect (Crest=6.0, Kurtosis=4.85) ---")
		defectTime := now.Add(4 * time.Minute)
		msg := BinduMessageEnvelope{
			Type:     "MachineVibrationSample",
			ID:       fmt.Sprintf("vib-epiroc-%d", defectTime.UnixNano()),
			When:     defectTime.Format(time.RFC3339),
			Group:    "mesh",
			Producer: "EPIROC-HB-02",
			User:     "simulator",
			NodeDest: "root",
			Fields: VibrationSampleFields{
				MachineID:         "EPIROC-HB-02",
				MachineModel:      "Epiroc HB Heavy Hydraulic Breaker",
				MachineType:       "ROCK_BREAKER",
				OperatingState:    "WORKING",
				VibrationRMS:      2.30,
				PeakG:             13.80, // High peak with moderate RMS -> Crest Factor = 6.0
				CrestFactor:       6.00,  // Breaches 4.8
				Kurtosis:          4.85,  // Breaches 4.2 -> CEL triggers BearingDefectSuspected -> Awakens SLM
				SampleTimestamp:   defectTime.Format(time.RFC3339),
				BatteryMillivolts: 3540,
				BeaconID:          "BLE-EPI-02",
				IngestMode:        "REALTIME",
			},
		}
		publishEvent(msg)
	}

	// 4. SCENARIO D: Drive-By Store-and-Forward Burst (30 1-Minute Samples)
	if *allScenarios || *burstMode {
		fmt.Println("\n--- Scenario 4: Drive-By Burst Ingestion (Supervisor Truck Downloads 30-Minute Ring Buffer) ---")
		baseTime := now.Add(-30 * time.Minute)
		for i := 1; i <= 30; i++ {
			sampleTime := baseTime.Add(time.Duration(i) * time.Minute)
			state := "WORKING"
			rms := 1.95 + (rand.Float64() * 0.3)
			if i <= 10 {
				state = "IDLE" // First 10 minutes were engine idling!
				rms = 0.42 + (rand.Float64() * 0.08)
			}
			peak := rms * (2.9 + (rand.Float64() * 0.4))
			crest := peak / rms
			kurt := 3.0 + (rand.Float64() * 0.3)

			msg := BinduMessageEnvelope{
				Type:     "MachineVibrationSample",
				ID:       fmt.Sprintf("vib-cat-burst-%02d-%d", i, sampleTime.UnixNano()),
				When:     sampleTime.Format(time.RFC3339), // Preserves physical event time!
				Group:    "mesh",
				Producer: "CAT-390F-01",
				User:     "drive-by-gateway",
				NodeDest: "root",
				Fields: VibrationSampleFields{
					MachineID:         "CAT-390F-01",
					MachineModel:      "CAT 390F Hydraulic Excavator",
					MachineType:       "EXCAVATOR",
					OperatingState:    state,
					VibrationRMS:      float64(int(rms*100)) / 100,
					PeakG:             float64(int(peak*100)) / 100,
					CrestFactor:       float64(int(crest*100)) / 100,
					Kurtosis:          float64(int(kurt*100)) / 100,
					SampleTimestamp:   sampleTime.Format(time.RFC3339),
					BatteryMillivolts: 3615,
					BeaconID:          "BLE-CAT-01",
					IngestMode:        "DRIVE_BY_BURST",
				},
			}
			publishEvent(msg)
		}
		fmt.Println("  └── ✅ Successfully emitted 30-minute historical ring buffer!")
	}

	fmt.Println("\n=================================================================")
	fmt.Println("🎉 Simulation Complete! Check Web Console Dashboards to view ledger:")
	fmt.Println("  • Fleet Health & State Table")
	fmt.Println("  • Structural Abuse Shock Incidents")
	fmt.Println("  • Predictive Maintenance Diagnosis (SLM Work Order)")
	fmt.Println("=================================================================")
}

func initMQTTClient(brokerStr, baseDir, certPath, keyPath, caPath string) (mqtt.Client, error) {
	u, err := url.Parse(brokerStr)
	if err != nil {
		return nil, fmt.Errorf("invalid broker URL: %w", err)
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerStr)
	opts.SetClientID(fmt.Sprintf("mining-fleet-sim-%d", time.Now().UnixNano()%10000))
	opts.SetCleanSession(true)
	opts.SetConnectTimeout(4 * time.Second)

	if u.Scheme == "tls" || u.Scheme == "ssl" || strings.HasSuffix(brokerStr, ":1888") {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}

		if certPath == "" && keyPath == "" && baseDir != "" {
			certPath = filepath.Join(baseDir, "runtime", "tls", "root.crt")
			keyPath = filepath.Join(baseDir, "runtime", "tls", "root.key")
		}

		if certPath != "" && keyPath != "" {
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err == nil {
				tlsConfig.Certificates = []tls.Certificate{cert}
			}
		}

		opts.SetTLSConfig(tlsConfig)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(4 * time.Second) {
		return nil, fmt.Errorf("connection timed out")
	}
	if token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}
