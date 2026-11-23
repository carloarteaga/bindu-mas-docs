//go:build ignore

package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
)

type captureStatusMsg struct {
	Type     string         `json:"$type"`
	ID       string         `json:"$id"`
	Group    string         `json:"$group"`
	Producer string         `json:"$producer"`
	NodeDest string         `json:"$node-dest,omitempty"`
	User     string         `json:"$user"`
	When     string         `json:"$when"`
	Fields   map[string]any `json:"fields"`
}

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 1883, "MQTT Broker Port")
	user := flag.String("user", "console", "MQTT Username")
	pass := flag.String("pass", "StrongPass123", "MQTT Password")
	topic := flag.String("topic", "bindu/in", "MQTT Ingress Topic")
	nodeDest := flag.String("node", "node-dell-1", "Target gateway node (e.g. node-dell-1, root)")
	producer := flag.String("producer", "", "Producer ID (defaults to target node)")
	capture := flag.Bool("capture", true, "Capture status: true (active) or false (paused)")
	reason := flag.String("reason", "", "Optional reason for status change")
	flag.Parse()

	prodID := *producer
	if prodID == "" {
		prodID = *nodeDest
	}
	if prodID == "" {
		prodID = "root"
	}

	eventID := fmt.Sprintf("status-cv-%s", uuid.NewString()[:8])
	nowStr := time.Now().UTC().Format(time.RFC3339)

	fields := map[string]any{
		"capture": *capture,
	}
	if *reason != "" {
		fields["reason"] = *reason
	}

	msg := captureStatusMsg{
		Type:     "EdgeCVSetStatus",
		ID:       eventID,
		Group:    "config",
		Producer: prodID,
		NodeDest: *nodeDest,
		User:     *user,
		When:     nowStr,
		Fields:   fields,
	}

	payload, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to serialize JSON message: %v\n", err)
		os.Exit(1)
	}

	stateLabel := "🔴 PAUSED (capture=false)"
	if *capture {
		stateLabel = "🟢 ACTIVE CAPTURE & INFERENCE (capture=true)"
	}

	fmt.Println("================================================================================")
	fmt.Println("📹 BINDU EDGE CV: Dynamic Capture Status Control (EdgeCVSetStatus)")
	fmt.Println("================================================================================")
	fmt.Printf("Broker Host:     %s:%d\n", *host, *port)
	fmt.Printf("Target Node:     %s\n", *nodeDest)
	fmt.Printf("Producer:        %s\n", prodID)
	fmt.Printf("Capture State:   %s\n", stateLabel)
	fmt.Printf("Topic:           %s\n", *topic)
	fmt.Println("--------------------------------------------------------------------------------")

	var client mqtt.Client
	connected := false
	var lastErr error

	// Try SSL/TLS first, then plaintext TCP fallback
	for attempt := 1; attempt <= 3; attempt++ {
		optsTLS := mqtt.NewClientOptions()
		optsTLS.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
		optsTLS.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
		if *user != "" {
			optsTLS.SetUsername(*user)
			optsTLS.SetPassword(*pass)
		}
		optsTLS.SetClientID(fmt.Sprintf("bindu-edge-cv-status-%d", time.Now().UnixNano()))
		optsTLS.SetConnectTimeout(3 * time.Second)

		cTLS := mqtt.NewClient(optsTLS)
		tokenTLS := cTLS.Connect()
		if tokenTLS.WaitTimeout(3*time.Second) && tokenTLS.Error() == nil {
			client = cTLS
			connected = true
			break
		}
		lastErr = tokenTLS.Error()

		optsTCP := mqtt.NewClientOptions()
		optsTCP.AddBroker(fmt.Sprintf("tcp://%s:%d", *host, *port))
		if *user != "" {
			optsTCP.SetUsername(*user)
			optsTCP.SetPassword(*pass)
		}
		optsTCP.SetClientID(fmt.Sprintf("bindu-edge-cv-status-tcp-%d", time.Now().UnixNano()))
		optsTCP.SetConnectTimeout(3 * time.Second)

		cTCP := mqtt.NewClient(optsTCP)
		tokenTCP := cTCP.Connect()
		if tokenTCP.WaitTimeout(3*time.Second) && tokenTCP.Error() == nil {
			client = cTCP
			connected = true
			break
		}
		lastErr = tokenTCP.Error()
		time.Sleep(300 * time.Millisecond)
	}

	if !connected {
		fmt.Fprintf(os.Stderr, "❌ Error connecting to MQTT broker %s:%d: %v\n", *host, *port, lastErr)
		os.Exit(1)
	}
	defer client.Disconnect(250)

	token := client.Publish(*topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		fmt.Fprintf(os.Stderr, "❌ Error publishing EdgeCVSetStatus: %v\n", token.Error())
		os.Exit(1)
	}

	fmt.Printf("✅ Published EdgeCVSetStatus successfully to MQTT (%s):\n", *topic)
	fmt.Println(strings.TrimSpace(string(payload)))
	fmt.Println("================================================================================")
}
