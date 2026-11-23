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
	topic := flag.String("topic", "bindu/in", "MQTT Topic")
	source := flag.String("source", "operator-manual-trigger", "Source identifier")
	producer := flag.String("producer", "mascl-operator-client", "Producer ID")
	notes := flag.String("notes", "Autonomous verification trigger for all 4 MAS agent paradigms", "Notes")
	flag.Parse()

	eventID := fmt.Sprintf("trig-%s", uuid.NewString()[:8])
	nowStr := time.Now().UTC().Format(time.RFC3339)

	event := map[string]any{
		"$id":       eventID,
		"$type":     "MASclTrigger",
		"$when":     nowStr,
		"$producer": *producer,
		"$user":     *user,
		"$group":    "mesh",
		"fields": map[string]any{
			"trigger_id": eventID,
			"source":     *source,
			"timestamp":  nowStr,
			"notes":      *notes,
		},
	}

	payloadBytes, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error serializing trigger event: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connecting to MQTT broker ssl://%s:%d...\n", *host, *port)

	var client mqtt.Client
	connected := false
	var lastErr error

	for attempt := 1; attempt <= 10; attempt++ {
		opts := mqtt.NewClientOptions()
		opts.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
		opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
		if *user != "" {
			opts.SetUsername(*user)
			opts.SetPassword(*pass)
		}
		opts.SetClientID(fmt.Sprintf("bindu-mascl-pub-%d", time.Now().UnixNano()))
		opts.SetConnectTimeout(3 * time.Second)

		c := mqtt.NewClient(opts)
		token := c.Connect()
		if token.WaitTimeout(3*time.Second) && token.Error() == nil {
			client = c
			connected = true
			break
		}
		lastErr = token.Error()

		// Fallback to plaintext TCP
		optsPlain := mqtt.NewClientOptions()
		optsPlain.AddBroker(fmt.Sprintf("tcp://%s:%d", *host, *port))
		if *user != "" {
			optsPlain.SetUsername(*user)
			optsPlain.SetPassword(*pass)
		}
		optsPlain.SetClientID(fmt.Sprintf("bindu-mascl-pub-tcp-%d", time.Now().UnixNano()))
		optsPlain.SetConnectTimeout(3 * time.Second)
		cPlain := mqtt.NewClient(optsPlain)
		tokenPlain := cPlain.Connect()
		if tokenPlain.WaitTimeout(3*time.Second) && tokenPlain.Error() == nil {
			client = cPlain
			connected = true
			break
		}
		lastErr = tokenPlain.Error()
		time.Sleep(500 * time.Millisecond)
	}

	if !connected {
		fmt.Fprintf(os.Stderr, "Failed to connect to MQTT broker after retries: %v\n", lastErr)
		os.Exit(1)
	}
	defer client.Disconnect(250)

	fmt.Printf("Publishing MASclTrigger to topic %s:\n%s\n", *topic, string(payloadBytes))
	pubToken := client.Publish(*topic, 1, false, payloadBytes)
	if !pubToken.WaitTimeout(5 * time.Second) || pubToken.Error() != nil {
		fmt.Fprintf(os.Stderr, "Publish failed: %v\n", pubToken.Error())
		os.Exit(1)
	}

	fmt.Println("✅ MASclTrigger event published successfully to the mesh!")
}
