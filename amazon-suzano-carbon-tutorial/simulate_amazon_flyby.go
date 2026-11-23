// Profile: ai-safe
// Bindu Amazon Precision Forestry & Airborne Fly-By Carbon Telemetry Simulator
// Demonstrates drone airborne data-mule harvesting from sub-canopy solar BLE stations,
// multispectral canopy foliage evaluation, deterministic bio-chemical stoichiometric deduction,
// and real-time deforestation threat alerting.

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SubCanopySoilFields struct {
	StationID        string  `json:"station_id"`
	ZoneID           string  `json:"zone_id"`
	HectareID        string  `json:"hectare_id"`
	SoilMoisturePct  float64 `json:"soil_moisture_pct"`
	AmbientTempC     float64 `json:"ambient_temp_c"`
	UnderstoryCO2Ppm float64 `json:"understory_co2_ppm"`
	ParSolarFluxUmol float64 `json:"par_solar_flux_umol"`
	BatteryMV        int     `json:"battery_mv"`
	SampleTimestamp  string  `json:"sample_timestamp"`
}

type CanopyFoliageFields struct {
	DroneID                 string  `json:"drone_id"`
	ZoneID                  string  `json:"zone_id"`
	HectareID               string  `json:"hectare_id"`
	FoliageDensityIndex     float64 `json:"foliage_density_index"`
	CanopyHealthFactor      float64 `json:"canopy_health_factor"`
	IllegalClearingDetected bool    `json:"illegal_clearing_detected"`
	ClearingAreaM2          float64 `json:"clearing_area_m2"`
	CanopyColorVariance     float64 `json:"canopy_color_variance"`
	InspectionNotes         string  `json:"inspection_notes"`
	InspectionTimestamp     string  `json:"inspection_timestamp"`
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
	threatOnly := flag.Bool("threat-only", false, "Simulate only the illegal clearing / deforestation anomaly")
	pristineOnly := flag.Bool("pristine-only", false, "Simulate only pristine primary forest hectares")
	dryRun := flag.Bool("dry-run", false, "Print generated JSON event payloads to stdout without publishing to MQTT")
	brokerURL := flag.String("broker", "tls://127.0.0.1:1888", "MQTT broker endpoint")
	clientCert := flag.String("cert", "", "Path to client certificate file (mTLS)")
	clientKey := flag.String("key", "", "Path to client private key file (mTLS)")
	caCert := flag.String("ca", "", "Path to CA certificate file")
	baseDir := flag.String("base-dir", "", "Bindu base directory for resolving TLS certificates")
	flag.Parse()

	fmt.Println("=================================================================")
	fmt.Println("🌳 BINDU AMAZON SUZANO PRECISION FORESTRY SIMULATOR")
	fmt.Println("   Airborne Data-Mule Fly-By & Stoichiometric Carbon Deduction")
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

	type HectareProfile struct {
		HectareID        string
		StationID        string
		SoilMoisture     float64
		AmbientTemp      float64
		UnderstoryCO2    float64
		ParFlux          float64
		BatteryMV        int
		FoliageDensity   float64
		CanopyHealth     float64
		IllegalClearing  bool
		ClearingAreaM2   float64
		ColorVariance    float64
		Notes            string
		IsDeforestation  bool
	}

	hectares := []HectareProfile{
		{
			HectareID:       "hectare-poly-AMZ-0841",
			StationID:       "SOL-STA-0841",
			SoilMoisture:    72.5,
			AmbientTemp:     26.2,
			UnderstoryCO2:   415.0,
			ParFlux:         145.0,
			BatteryMV:       3850,
			FoliageDensity:  0.94,
			CanopyHealth:    0.92,
			IllegalClearing: false,
			ClearingAreaM2:  0.0,
			ColorVariance:   0.18,
			Notes:           "Pristine primary Terra Firme forest. Optimal sub-canopy moisture and complete emergent crown coverage.",
			IsDeforestation: false,
		},
		{
			HectareID:       "hectare-poly-AMZ-0842",
			StationID:       "SOL-STA-0842",
			SoilMoisture:    84.0,
			AmbientTemp:     25.5,
			UnderstoryCO2:   422.0,
			ParFlux:         110.0,
			BatteryMV:       3920,
			FoliageDensity:  0.96,
			CanopyHealth:    0.95,
			IllegalClearing: false,
			ClearingAreaM2:  0.0,
			ColorVariance:   0.22,
			Notes:           "Riparian/Varzea flood basin forest. High biological respiration and intense photosynthetic biomass accretion.",
			IsDeforestation: false,
		},
		{
			HectareID:       "hectare-poly-AMZ-0843",
			StationID:       "SOL-STA-0843",
			SoilMoisture:    65.0,
			AmbientTemp:     27.8,
			UnderstoryCO2:   408.0,
			ParFlux:         180.0,
			BatteryMV:       3780,
			FoliageDensity:  0.91,
			CanopyHealth:    0.89,
			IllegalClearing: false,
			ClearingAreaM2:  0.0,
			ColorVariance:   0.15,
			Notes:           "Upland plateau forest with dense multi-strata understory. Normal seasonal transpiration balance.",
			IsDeforestation: false,
		},
		{
			HectareID:       "hectare-poly-AMZ-0844",
			StationID:       "SOL-STA-0844",
			SoilMoisture:    68.0,
			AmbientTemp:     27.0,
			UnderstoryCO2:   412.0,
			ParFlux:         165.0,
			BatteryMV:       3810,
			FoliageDensity:  0.88,
			CanopyHealth:    0.86,
			IllegalClearing: false,
			ClearingAreaM2:  0.0,
			ColorVariance:   0.19,
			Notes:           "Regenerating 30-year secondary forest. Rapid carbon capture trajectory and healthy pioneer species mix.",
			IsDeforestation: false,
		},
		{
			HectareID:       "hectare-poly-AMZ-0845",
			StationID:       "SOL-STA-0845",
			SoilMoisture:    42.0,
			AmbientTemp:     34.5,
			UnderstoryCO2:   380.0,
			ParFlux:         1250.0,
			BatteryMV:       4100,
			FoliageDensity:  0.35,
			CanopyHealth:    0.38,
			IllegalClearing: true,
			ClearingAreaM2:  1450.0,
			ColorVariance:   0.45,
			Notes:           "CRITICAL ALERT: Detected active illegal clearing / logging gap! Soil desiccating under direct sunlight.",
			IsDeforestation: true,
		},
	}

	droneNodeID := "DRONE-AMZ-01"
	zoneID := "AMAZON-SECTOR-B"
	now := time.Now().UTC()

	for i, h := range hectares {
		if *threatOnly && !h.IsDeforestation {
			continue
		}
		if *pristineOnly && h.IsDeforestation {
			continue
		}

		sampleTime := now.Add(time.Duration(i*2) * time.Minute).Format(time.RFC3339)
		fmt.Printf("\n🚁 Drone '%s' performing Fly-By over '%s' (Station: %s)...\n", droneNodeID, h.HectareID, h.StationID)

		// 1. Emit SubCanopySoilTelemetry (harvested via vertical BLE fly-by from the solar station)
		soilMsg := BinduMessageEnvelope{
			Type:     "SubCanopySoilTelemetry",
			ID:       fmt.Sprintf("MSG-SOIL-%s-%d", h.StationID, time.Now().UnixNano()),
			When:     sampleTime,
			Group:    "mesh",
			Producer: h.StationID,
			User:     "drone-agent",
			NodeDest: "root",
			Fields: SubCanopySoilFields{
				StationID:        h.StationID,
				ZoneID:           zoneID,
				HectareID:        h.HectareID,
				SoilMoisturePct:  h.SoilMoisture,
				AmbientTempC:     h.AmbientTemp,
				UnderstoryCO2Ppm: h.UnderstoryCO2,
				ParSolarFluxUmol: h.ParFlux,
				BatteryMV:        h.BatteryMV,
				SampleTimestamp:  sampleTime,
			},
		}
		publishEvent(soilMsg)

		// 2. Emit CanopyFoliageInspection (captured by drone camera and classified by on-board VLM)
		canopyMsg := BinduMessageEnvelope{
			Type:     "CanopyFoliageInspection",
			ID:       fmt.Sprintf("MSG-CANOPY-%s-%d", h.HectareID, time.Now().UnixNano()),
			When:     sampleTime,
			Group:    "mesh",
			Producer: droneNodeID,
			User:     "drone-agent",
			NodeDest: "root",
			Fields: CanopyFoliageFields{
				DroneID:                 droneNodeID,
				ZoneID:                  zoneID,
				HectareID:               h.HectareID,
				FoliageDensityIndex:     h.FoliageDensity,
				CanopyHealthFactor:      h.CanopyHealth,
				IllegalClearingDetected: h.IllegalClearing,
				ClearingAreaM2:          h.ClearingAreaM2,
				CanopyColorVariance:     h.ColorVariance,
				InspectionNotes:         h.Notes,
				InspectionTimestamp:     sampleTime,
			},
		}
		publishEvent(canopyMsg)

		if !*dryRun {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Println("\n=================================================================")
	fmt.Println("✨ Simulation cycle complete.")
	fmt.Println("👉 Inspect results in Bindu Console: http://127.0.0.1:8080/console/dashboards")
	fmt.Println("=================================================================")
}

func initMQTTClient(brokerURL, baseDir, certPath, keyPath, caPath string) (mqtt.Client, error) {
	u, err := url.Parse(brokerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid broker url: %w", err)
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(fmt.Sprintf("suzano-flyby-sim-%d", time.Now().UnixNano()))
	opts.SetCleanSession(true)
	opts.SetKeepAlive(30 * time.Second)

	if strings.ToLower(u.Scheme) == "tls" || strings.ToLower(u.Scheme) == "ssl" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}

		if certPath != "" && keyPath != "" {
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err == nil {
				tlsConfig.Certificates = []tls.Certificate{cert}
			}
		} else if baseDir != "" {
			stdCert := filepath.Join(baseDir, "tls", "client.crt")
			stdKey := filepath.Join(baseDir, "tls", "client.key")
			cert, err := tls.LoadX509KeyPair(stdCert, stdKey)
			if err == nil {
				tlsConfig.Certificates = []tls.Certificate{cert}
			}
		}

		opts.SetTLSConfig(tlsConfig)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}
