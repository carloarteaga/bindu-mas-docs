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
	scenario := flag.String("scenario", "mildew_risk", "Simulation Scenario: mildew_risk | frost_risk | nominal")
	vineyardID := flag.String("vineyard", "BODEGA-ALBARINO-01", "Vineyard Identifier")
	parcel := flag.String("parcel", "Val-Do-Salnes-Finca-Abalos", "Parcel Name")
	where := flag.String("where", "/rias-baixas/salnes", "Locagram Path")
	producer := flag.String("producer", "station-salnes-01", "Producer ID")
	rawMsg := flag.String("msg", "", "Optional raw JSON message (overrides scenario generation)")
	flag.Parse()

	var payloadBytes []byte
	var err error

	if *rawMsg != "" {
		payloadBytes = []byte(*rawMsg)
	} else {
		var temp, humidity, soil, solar, wetness float64
		switch *scenario {
		case "frost_risk":
			temp = 3.2
			humidity = 88.0
			soil = 42.0
			solar = 120.0
			wetness = 4.5
		case "nominal":
			temp = 22.5
			humidity = 60.0
			soil = 55.0
			solar = 650.0
			wetness = 0.5
		case "mildew_risk":
			fallthrough
		default:
			temp = 18.5
			humidity = 92.5
			soil = 38.0
			solar = 280.0
			wetness = 5.2
		}

		eventID := fmt.Sprintf("vnyd-%s", uuid.NewString()[:8])
		event := map[string]any{
			"$id":       eventID,
			"$type":     "SensorVineyardReading",
			"$when":     time.Now().UTC().Format(time.RFC3339),
			"$producer": *producer,
			"$user":     *user,
			"$group":    "mesh",
			"$where": map[string]any{
				"locagram": *where,
			},
			"fields": map[string]any{
				"vineyard_id":          *vineyardID,
				"parcel_name":          *parcel,
				"ambient_temp_c":       temp,
				"humidity_pct":         humidity,
				"soil_moisture_pct":    soil,
				"solar_radiation_w_m2": solar,
				"leaf_wetness_hours":   wetness,
			},
		}

		payloadBytes, err = json.MarshalIndent(event, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error serializing vineyard event: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Connecting to MQTT broker ssl://%s:%d...\n", *host, *port)

	// Try TLS first, fallback to TCP if needed
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
	opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	if *user != "" {
		opts.SetUsername(*user)
		opts.SetPassword(*pass)
	}
	opts.SetClientID(fmt.Sprintf("bindu-vineyard-pub-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(5*time.Second) || token.Error() != nil {
		// Attempt plain TCP connection fallback
		optsPlain := mqtt.NewClientOptions()
		optsPlain.AddBroker(fmt.Sprintf("tcp://%s:%d", *host, *port))
		if *user != "" {
			optsPlain.SetUsername(*user)
			optsPlain.SetPassword(*pass)
		}
		optsPlain.SetClientID(fmt.Sprintf("bindu-vineyard-pub-tcp-%d", time.Now().UnixNano()))
		optsPlain.SetConnectTimeout(5 * time.Second)
		client = mqtt.NewClient(optsPlain)
		tokenPlain := client.Connect()
		if !tokenPlain.WaitTimeout(5*time.Second) || tokenPlain.Error() != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to MQTT broker (%s:%d): TLS err: %v, Plain err: %v\n",
				*host, *port, token.Error(), tokenPlain.Error())
			os.Exit(1)
		}
	}
	defer client.Disconnect(250)

	fmt.Println("✅ Connected to MQTT broker successfully!")
	pubToken := client.Publish(*topic, 1, false, payloadBytes)
	if !pubToken.WaitTimeout(5 * time.Second) {
		fmt.Fprintf(os.Stderr, "Timeout publishing to topic %s\n", *topic)
		os.Exit(1)
	}
	if pubToken.Error() != nil {
		fmt.Fprintf(os.Stderr, "Failed to publish event to topic %s: %v\n", *topic, pubToken.Error())
		os.Exit(1)
	}

	fmt.Printf("🍇 Published SensorVineyardReading to topic %q:\n%s\n", *topic, string(payloadBytes))
}
