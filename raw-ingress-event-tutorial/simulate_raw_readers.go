//go:build ignore

package main

import (
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var sampleEPCs = []string{
	"E280110C2000000213456789", // SGTIN Case Pack
	"300833B2DDD9014000000001", // SSCC Master Pallet
	"E2000019060B013317708573", // SGLN Location Anchor
	"E280116060000204B768820A", // GRAI Returnable Tote
	"3074257BF419CB80000007EA", // GIAI Fleet Forklift
}

type HopelandTagItem struct {
	EPC       string  `json:"epc"`
	TID       string  `json:"tid,omitempty"`
	RSSI      float64 `json:"rssi"`
	ReadCount int     `json:"read_count"`
	FirstSeen int64   `json:"first_seen"`
	LastSeen  int64   `json:"last_seen"`
}

type HopelandPayload struct {
	DeviceID  string            `json:"device_id"`
	ReaderIP  string            `json:"reader_ip"`
	Timestamp int64             `json:"timestamp"`
	MsgType   string            `json:"msg_type"`
	Antenna   int               `json:"antenna"`
	Tags      []HopelandTagItem `json:"tags"`
}

type ZebraDataBlock struct {
	IDHex    string  `json:"idHex"`
	Antenna  int     `json:"antenna"`
	PeakRSSI float64 `json:"peakRssi"`
	Reads    int     `json:"reads"`
	Format   string  `json:"format"`
}

type ZebraPayload struct {
	Timestamp string         `json:"timestamp"`
	Type      string         `json:"type"`
	Data      ZebraDataBlock `json:"data"`
}

type ChafonTagItem struct {
	EPC  string  `json:"epc"`
	RSSI float64 `json:"rssi"`
	PC   string  `json:"pc"`
}

type ChafonPayload struct {
	DeviceID  string        `json:"device_id"`
	Antenna   int           `json:"antenna"`
	Timestamp int64         `json:"timestamp"`
	Tag       ChafonTagItem `json:"tag"`
}

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 1885, "MQTT Broker Port")
	user := flag.String("user", "console", "MQTT Username")
	pass := flag.String("pass", "SuperSecret1", "MQTT Password")
	broker := flag.String("broker", "", "Explicit MQTT broker URL (e.g. ssl://127.0.0.1:1885)")
	reader := flag.String("reader", "all", "Reader model to simulate (all, hopeland, zebra, chafon, marktrace)")
	count := flag.Int("count", 20, "Number of readings to publish (0 for continuous)")
	interval := flag.Duration("interval", 500*time.Millisecond, "Publishing interval between readings")
	insecure := flag.Bool("insecure", true, "Skip TLS verification for localhost certificates")
	flag.Parse()

	readerChoice := strings.ToLower(strings.TrimSpace(*reader))
	if readerChoice != "all" && readerChoice != "hopeland" && readerChoice != "zebra" && readerChoice != "chafon" && readerChoice != "marktrace" {
		fmt.Fprintf(os.Stderr, "❌ Invalid reader choice %q. Supported options: all, hopeland, zebra, chafon, marktrace\n", *reader)
		os.Exit(1)
	}

	brokerURL := *broker
	if brokerURL == "" {
		brokerURL = fmt.Sprintf("tcp://%s:%d", *host, *port)
	}

	fmt.Println("=================================================================")
	fmt.Println("📡 Bindu Heterogeneous Multi-Reader Raw Ingress Simulator")
	fmt.Println("=================================================================")
	fmt.Printf("🎯 Target Broker:   %s\n", brokerURL)
	fmt.Printf("👤 Auth User:       %s\n", *user)
	fmt.Printf("🏷️ Reader Profile:  %s\n", strings.ToUpper(readerChoice))
	fmt.Printf("⏱️  Interval:        %s\n", *interval)
	if *count > 0 {
		fmt.Printf("🔢 Total Count:     %d readings\n", *count)
	} else {
		fmt.Println("🔢 Total Count:     Continuous (Ctrl+C to stop)")
	}
	fmt.Println("=================================================================")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(fmt.Sprintf("raw-ingress-sim-%d", time.Now().UnixNano()%100000))
	opts.SetUsername(*user)
	opts.SetPassword(*pass)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)

	if strings.HasPrefix(brokerURL, "ssl://") || strings.HasPrefix(brokerURL, "tls://") {
		u, _ := url.Parse(brokerURL)
		serverName := u.Hostname()
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: *insecure,
			ServerName:         serverName,
		})
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		fmt.Fprintf(os.Stderr, "❌ Connection timeout to broker %s\n", brokerURL)
		os.Exit(1)
	}
	if err := token.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to connect to broker %s: %v\n", brokerURL, err)
		os.Exit(1)
	}
	defer client.Disconnect(500)
	fmt.Println("✅ Connected successfully to MQTT Broker.")
	fmt.Println("🚀 Streaming raw hardware frames to non-semantic topic bindu/in/raw/...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	readers := []string{"hopeland", "zebra", "chafon", "marktrace"}
	if readerChoice != "all" {
		readers = []string{readerChoice}
	}

	iteration := 0
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			fmt.Println("\n🛑 Gracefully shutting down simulator...")
			return
		case <-ticker.C:
			currentReader := readers[iteration%len(readers)]
			iteration++

			// Introduce occasional marginal RSSI (< -75 dBm) to test weak_signal_health_guard
			isWeakSignal := (iteration % 5) == 0
			antennaPort := 1 + (iteration % 4)
			epc := sampleEPCs[iteration%len(sampleEPCs)]

			var topic string
			var payloadBytes []byte
			var summary string

			switch currentReader {
			case "hopeland":
				topic = "bindu/in/raw/rfid/hopeland/dock_door_1"
				rssi := -42.0 - float64(iteration%15)
				if isWeakSignal {
					rssi = -78.5 - float64(iteration%10)
				}
				nowMs := time.Now().UnixMilli()
				hPayload := HopelandPayload{
					DeviceID:  "HH340-A01",
					ReaderIP:  "192.168.10.51",
					Timestamp: nowMs,
					MsgType:   "tag_report",
					Antenna:   antennaPort,
					Tags: []HopelandTagItem{
						{
							EPC:       epc,
							TID:       "E280110C2000",
							RSSI:      rssi,
							ReadCount: 8 + (iteration % 10),
							FirstSeen: nowMs - 120,
							LastSeen:  nowMs,
						},
					},
				}
				payloadBytes, _ = json.Marshal(hPayload)
				summary = fmt.Sprintf("Vendor: Hopeland | Ant: %d | EPC: %s | RSSI: %.1f dBm", antennaPort, epc, rssi)

			case "zebra":
				topic = "bindu/in/raw/rfid/zebra/dock_north"
				rssi := -48.0 - float64(iteration%18)
				if isWeakSignal {
					rssi = -82.0 - float64(iteration%8)
				}
				zPayload := ZebraPayload{
					Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
					Type:      "SIMPLE",
					Data: ZebraDataBlock{
						IDHex:    epc,
						Antenna:  antennaPort,
						PeakRSSI: rssi,
						Reads:    4 + (iteration % 6),
						Format:   "epc",
					},
				}
				payloadBytes, _ = json.Marshal(zPayload)
				summary = fmt.Sprintf("Vendor: Zebra    | Ant: %d | EPC: %s | RSSI: %.1f dBm", antennaPort, epc, rssi)

			case "chafon":
				topic = "bindu/in/raw/rfid/chafon/gate_01"
				rssi := -50.0 - float64(iteration%12)
				if isWeakSignal {
					rssi = -79.0 - float64(iteration%11)
				}
				cPayload := ChafonPayload{
					DeviceID:  "CF6D2-GATE-01",
					Antenna:   antennaPort,
					Timestamp: time.Now().UnixMilli(),
					Tag: ChafonTagItem{
						EPC:  epc,
						RSSI: rssi,
						PC:   "3000",
					},
				}
				payloadBytes, _ = json.Marshal(cPayload)
				summary = fmt.Sprintf("Vendor: Chafon   | Ant: %d | EPC: %s | RSSI: %.1f dBm", antennaPort, epc, rssi)

			case "marktrace":
				topic = "bindu/in/raw/rfid/marktrace/portal_west"
				rssi := -45.0 - float64(iteration%16)
				if isWeakSignal {
					rssi = -80.0 - float64(iteration%9)
				}
				// 18-byte Binary Frame Protocol:
				// [0]: 0xBB (Header)
				// [1]: 0x02 (Tag Report)
				// [2]: 0x22 (Subcommand)
				// [3]: 0x00 (Reserved)
				// [4]: Antenna (1..4)
				// [5..16]: 12-byte EPC
				// [17]: RSSI byte (256 + rssi)
				frame := make([]byte, 18)
				frame[0] = 0xBB
				frame[1] = 0x02
				frame[2] = 0x22
				frame[3] = 0x00
				frame[4] = byte(antennaPort)
				epcBytes, _ := hex.DecodeString(epc)
				if len(epcBytes) == 12 {
					copy(frame[5:17], epcBytes)
				} else {
					copy(frame[5:17], []byte{0xE2, 0x80, 0x11, 0x60, 0x60, 0x00, 0x02, 0x04, 0xB7, 0x68, 0x82, 0x0A})
				}
				frame[17] = byte(int(rssi) + 256)
				payloadBytes = frame
				summary = fmt.Sprintf("Vendor: Marktrace [BINARY 18-BYTE] | Ant: %d | EPC: %s | RSSI: %.1f dBm", antennaPort, epc, rssi)
			}

			pubToken := client.Publish(topic, 0, false, payloadBytes)
			if !pubToken.WaitTimeout(3 * time.Second) {
				fmt.Printf("[%03d] ⚠️ Publish timeout on %s\n", iteration, topic)
			} else if pubToken.Error() != nil {
				fmt.Printf("[%03d] ❌ Publish failed on %s: %v\n", iteration, topic, pubToken.Error())
			} else {
				if isWeakSignal {
					fmt.Printf("[%03d] ⚠️  PUB -> %s (WEAK SIGNAL ALERT TRIGGER)\n       %s\n", iteration, topic, summary)
				} else {
					fmt.Printf("[%03d] 📤 PUB -> %s\n       %s\n", iteration, topic, summary)
				}
			}

			if *count > 0 && iteration >= *count {
				fmt.Println("\n🎉 Completed requested reading sequence.")
				return
			}
		}
	}
}
