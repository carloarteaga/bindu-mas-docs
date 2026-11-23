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
	level := flag.String("level", "DEEP", "Audit Level: STANDARD, DEEP, or SECURITY")
	cluster := flag.String("cluster", "/galicia", "Target cluster locagram path")
	flag.Parse()

	fmt.Println("=== Bindu CEL & celext Fleet Audit Trigger ===")
	fmt.Printf("MQTT Broker: ssl://%s:%d | User: %s\n", *host, *port, *user)
	fmt.Printf("Audit Level: %s | Target Cluster: %s\n", *level, *cluster)
	fmt.Println("--------------------------------------------------------")

	// Connect to MQTT broker
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	if *user != "" {
		opts.SetUsername(*user)
		opts.SetPassword(*pass)
	}
	opts.SetClientID(fmt.Sprintf("bindu-audit-trigger-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to MQTT broker: %v\n", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)
	fmt.Println("✅ Connected to MQTT broker successfully!")

	// Construct MeshAuditTrigger event
	triggerID := fmt.Sprintf("audit-trig-%s", uuid.NewString()[:8])
	event := map[string]any{
		"$type":     "MeshAuditTrigger",
		"$id":       triggerID,
		"$group":    "mesh",
		"$producer": "audit-operator-cli",
		"$user":     "admin",
		"$when":     time.Now().UTC().Format(time.RFC3339),
		"$where": map[string]any{
			"locagram": *cluster,
		},
		"fields": map[string]any{
			"audit_level":      *level,
			"target_cluster":   *cluster,
			"check_compliance": true,
		},
	}

	payloadBytes, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error serializing event: %v\n", err)
		os.Exit(1)
	}

	topic := "bindu/in"
	token := client.Publish(topic, 0, false, payloadBytes)
	token.Wait()
	if token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Failed to publish event to topic %s: %v\n", topic, token.Error())
		os.Exit(1)
	}

	fmt.Printf("🚀 Published MeshAuditTrigger to %s:\n%s\n", topic, string(payloadBytes))
	fmt.Println("--------------------------------------------------------")
	fmt.Println("🎯 Check Bindu console or logs to observe the CEL reactor evaluate the audit and emit MeshTopologyInspection!")
}
