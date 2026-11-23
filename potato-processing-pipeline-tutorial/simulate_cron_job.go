//go:build ignore

// ==============================================================================
// ⏰ SystemCronJob Awakening Event Simulator for Potato Processing Pipeline
// ==============================================================================
// Simulates the Go Core scheduler firing a timer and publishing SystemCronJob
// to bindu/in to awaken a digital twin after silo quarantine.
// ==============================================================================

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SystemCronJobFields struct {
	ScheduledEventID     string         `json:"scheduled_event_id"`
	Action               string         `json:"action"`
	ScheduledTime        string         `json:"scheduled_time"`
	ActualTime           string         `json:"actual_time"`
	LatencyMs            float64        `json:"latency_ms"`
	RunCount             int            `json:"run_count"`
	MissedDuringDowntime bool           `json:"missed_during_downtime"`
	CustomPayload        map[string]any `json:"custom_payload,omitempty"`
}

type SystemCronJobMessage struct {
	Type     string              `json:"$type"`
	ID       string              `json:"$id"`
	When     string              `json:"$when"`
	Group    string              `json:"$group"`
	Producer string              `json:"$producer"`
	User     string              `json:"$user"`
	NodeDest string              `json:"$node-dest"`
	Fields   SystemCronJobFields `json:"fields"`
}

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 38190, "MQTT Broker Port")
	user := flag.String("user", "console", "MQTT Username")
	pass := flag.String("pass", "StrongPass123", "MQTT Password")
	broker := flag.String("broker", "", "Explicit MQTT broker endpoint (e.g. ssl://127.0.0.1:38190)")

	lotID := flag.String("lot-id", "lot-potato-cold-001", "Target Digital Twin Lot ID ($producer)")
	eventID := flag.String("event-id", "", "Scheduled Timer / Job ID (defaults to silo-quarantine-<lot-id>)")
	action := flag.String("action", "recondition_wakeup", "Scheduler action tag")
	oilTemp := flag.Float64("oil-temp", 178.5, "Target fryer oil temperature setpoint (°C)")
	beltSpeed := flag.Float64("belt-speed", 32.0, "Target conveyor belt speed (mm/s)")
	nodeDest := flag.String("node-dest", "root", "Target execution node")
	flag.Parse()

	finalEventID := *eventID
	if finalEventID == "" {
		finalEventID = fmt.Sprintf("silo-quarantine-%s", *lotID)
	}

	now := time.Now().UTC()
	scheduledTime := now.Add(-30 * time.Second)

	msg := SystemCronJobMessage{
		Type:     "SystemCronJob",
		ID:       fmt.Sprintf("trigger-%s-%d", finalEventID, now.UnixNano()%1000000),
		When:     now.Format(time.RFC3339),
		Group:    "mesh",
		Producer: *lotID, // Preserves the exact partition triplet ($type, $group, $producer) for Zero-RAG SQL
		User:     *user,
		NodeDest: *nodeDest,
		Fields: SystemCronJobFields{
			ScheduledEventID:     finalEventID,
			Action:               *action,
			ScheduledTime:        scheduledTime.Format(time.RFC3339),
			ActualTime:           now.Format(time.RFC3339),
			LatencyMs:            1.42,
			RunCount:             1,
			MissedDuringDowntime: false,
			CustomPayload: map[string]any{
				"lot_id":            *lotID,
				"silo_id":           "silo-01",
				"target_oil_temp":   *oilTemp,
				"target_belt_speed": *beltSpeed,
				"reason":            "Silo quarantine reconditioning period completed; optimal sugar levels restored",
			},
		},
	}

	payload, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println("=================================================================")
	fmt.Printf("⏰ [CRON JOB SIMULATOR] Firing SystemCronJob Awakening for %s\n", *lotID)
	fmt.Printf("   ├─ Scheduled Event ID: %s\n", finalEventID)
	fmt.Printf("   ├─ Digital Twin Lot:   %s ($producer)\n", *lotID)
	fmt.Printf("   ├─ Target Action:      %s\n", *action)
	fmt.Printf("   ├─ Target Fryer Temp:  %.1f °C (Modbus Reg 200 = %d)\n", *oilTemp, int(*oilTemp*10))
	fmt.Printf("   ├─ Target Belt Speed:  %.1f mm/s (Modbus Reg 201 = %d)\n", *beltSpeed, int(*beltSpeed))
	fmt.Println("=================================================================")

	// Connect to MQTT Broker with fallback
	var client mqtt.Client
	var connected bool
	var lastErr error

	brokerEndpoints := []string{}
	if *broker != "" {
		brokerEndpoints = append(brokerEndpoints, *broker)
	} else {
		brokerEndpoints = append(brokerEndpoints,
			fmt.Sprintf("ssl://%s:%d", *host, *port),
			fmt.Sprintf("tcp://%s:%d", *host, *port),
		)
	}

	for _, ep := range brokerEndpoints {
		opts := mqtt.NewClientOptions().
			AddBroker(ep).
			SetClientID(fmt.Sprintf("cron-simulator-%d", time.Now().UnixNano())).
			SetUsername(*user).
			SetPassword(*pass).
			SetTLSConfig(&tls.Config{InsecureSkipVerify: true}).
			SetCleanSession(true).
			SetConnectTimeout(3 * time.Second)

		c := mqtt.NewClient(opts)
		token := c.Connect()
		if token.WaitTimeout(3*time.Second) && token.Error() == nil {
			client = c
			connected = true
			fmt.Printf("🔌 Connected to MQTT broker (%s)\n", ep)
			break
		}
		lastErr = token.Error()
	}

	if !connected {
		log.Fatalf("❌ Failed to connect to MQTT broker (%v): %v", brokerEndpoints, lastErr)
	}
	defer client.Disconnect(250)

	// Publish to bindu/in
	token := client.Publish("bindu/in", 0, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Fatalf("❌ Failed to publish to bindu/in: %v", token.Error())
	}

	fmt.Printf("✅ SystemCronJob awakening event published to bindu/in ($id: %s)!\n\n", msg.ID)
}
