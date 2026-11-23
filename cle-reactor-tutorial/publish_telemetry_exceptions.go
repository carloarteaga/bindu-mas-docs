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

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 1883, "MQTT Broker Port")
	user := flag.String("user", "console", "MQTT Username")
	pass := flag.String("pass", "StrongPass123", "MQTT Password")
	flag.Parse()

	fmt.Println("=== Bindu Telemetry Exception Simulator ===")
	fmt.Printf("MQTT Broker: ssl://%s:%d | User: %s\n", *host, *port, *user)
	fmt.Println("--------------------------------------------------------")

	// Connect to MQTT
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	if *user != "" {
		opts.SetUsername(*user)
		opts.SetPassword(*pass)
	}
	opts.SetClientID(fmt.Sprintf("bindu-sysmon-sim-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to MQTT broker: %v\n", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)
	fmt.Println("Connected to MQTT successfully!")

	// 1. Publish Unhealthy MessageState (CPU threshold exceeded)
	msgStateID := fmt.Sprintf("sim-state-%s", uuid.NewString()[:8])
	stateMsg := map[string]any{
		"$type":       "MessageState",
		"$id":         msgStateID,
		"$group":      "mesh",
		"$producer":   "edge-node-sim",
		"$node":       "edge-node-sim", // EnvelopeField required by MessageState
		"$uptime_sec": 120,             // EnvelopeField required by MessageState
		"$user":       "system",
		"$when":       time.Now().UTC().Format(time.RFC3339),
		"fields": map[string]any{
			"cpu": map[string]any{
				"usage_pct": 98.5, // Alert limit is > 90%
			},
			"memory": map[string]any{
				"used_pct": 45.0,
			},
			"disk": map[string]any{
				"used_pct": 30.0,
			},
		},
	}
	publishMessage(client, stateMsg, "MessageState (High CPU)", msgStateID)

	time.Sleep(1 * time.Second)

	// 2. Publish Unhealthy MessageState (Stopped BPR plugin hello-world-plugin)
	msgVersionID := fmt.Sprintf("sim-version-%s", uuid.NewString()[:8])
	versionMsg := map[string]any{
		"$type":       "MessageState",
		"$id":         msgVersionID,
		"$group":      "mesh",
		"$producer":   "edge-node-sim",
		"$node":       "edge-node-sim", // EnvelopeField required by MessageState
		"$uptime_sec": 120,             // EnvelopeField required by MessageState
		"$user":       "system",
		"$when":       time.Now().UTC().Format(time.RFC3339),
		"fields": map[string]any{
			"bpr_plugins": []any{
				map[string]any{
					"id":      "hello-world-plugin",
					"version": "1.0.0",
					"status":  "stopped", // Alert triggers if != "running"
					"pid":     0,
				},
			},
		},
	}
	publishMessage(client, versionMsg, "MessageState (BPR Stopped)", msgVersionID)

	fmt.Println("\nExecution completed! Published 2 exception events to bindu/in.")
}

func publishMessage(client mqtt.Client, msg map[string]any, label string, msgID string) {
	payload, err := json.Marshal(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal message: %v\n", err)
		return
	}

	token := client.Publish("bindu/in", 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Failed to publish %s: %v\n", label, token.Error())
	} else {
		fmt.Printf("Published unhealthy %s (Message ID: %s)\n", label, msgID)
	}
}
