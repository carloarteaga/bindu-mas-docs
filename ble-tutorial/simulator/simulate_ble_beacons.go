// Profile: ai-safe
// Bindu BLE Beacon Simulator
// Broadcasts genuine 2.4 GHz physical iBeacon radio frames via tinygo.org/x/bluetooth
// allowing operators to simulate tag arrival, presence, and exit without external hardware dongles.

package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	appleCompanyID = 0x004c
)

type Simulator struct {
	mu          sync.Mutex
	adapter     *bluetooth.Adapter
	adv         *bluetooth.Advertisement
	backend     string
	uuid        string
	major       uint16
	minor       uint16
	txPower     int
	interval    time.Duration
	advertising bool
}

func NewSimulator(backend, uuid string, major, minor uint16, txPower int, interval time.Duration) *Simulator {
	if strings.TrimSpace(uuid) == "" {
		uuid = "fda50693-a4e2-4fb1-afcf-c6eb07647825"
	}
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	return &Simulator{
		backend:  backend,
		uuid:     uuid,
		major:    major,
		minor:    minor,
		txPower:  txPower,
		interval: interval,
	}
}

func (s *Simulator) StartAdvertising() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.advertising {
		fmt.Printf("⚠️ Beacon is already advertising [UUID=%s Major=%d Minor=%d]\n", s.uuid, s.major, s.minor)
		return nil
	}

	payload, err := s.buildIBeaconPayload()
	if err != nil {
		return err
	}

	if s.backend == "dry-run" {
		s.advertising = true
		fmt.Printf("📢 [DRY-RUN] Started simulated advertising iBeacon [UUID=%s Major=%d Minor=%d TxPower=%d Interval=%v]\n",
			s.uuid, s.major, s.minor, s.txPower, s.interval)
		return nil
	}

	if s.adapter == nil {
		adapter := bluetooth.DefaultAdapter
		if err := adapter.Enable(); err != nil {
			return fmt.Errorf("failed to enable bluetooth adapter: %w", err)
		}
		s.adapter = adapter
		s.adv = adapter.DefaultAdvertisement()
	}

	opts := bluetooth.AdvertisementOptions{
		LocalName: fmt.Sprintf("Bindu-%d-%d", s.major, s.minor),
		Interval:  bluetooth.NewDuration(s.interval),
		ManufacturerData: []bluetooth.ManufacturerDataElement{
			{
				CompanyID: appleCompanyID,
				Data:      payload,
			},
		},
	}

	if err := s.adv.Configure(opts); err != nil {
		return fmt.Errorf("failed to configure bluetooth advertisement: %w", err)
	}

	if err := s.adv.Start(); err != nil {
		return fmt.Errorf("failed to start advertising: %w", err)
	}

	s.advertising = true
	fmt.Printf("🟢 [RADIO ON] Broadcasting physical 2.4 GHz iBeacon [UUID=%s Major=%d Minor=%d TxPower=%d dBm]\n",
		s.uuid, s.major, s.minor, s.txPower)
	return nil
}

func (s *Simulator) StopAdvertising() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.advertising {
		fmt.Printf("⚠️ Beacon is already stopped\n")
		return nil
	}

	if s.backend == "dry-run" {
		s.advertising = false
		fmt.Printf("🔴 [DRY-RUN] Stopped simulated advertising iBeacon\n")
		return nil
	}

	if s.adv != nil {
		if err := s.adv.Stop(); err != nil {
			return fmt.Errorf("failed to stop advertising: %w", err)
		}
	}

	s.advertising = false
	fmt.Printf("🔴 [RADIO OFF] Stopped broadcasting iBeacon. Radio is now SILENT.\n")
	return nil
}

func (s *Simulator) buildIBeaconPayload() ([]byte, error) {
	cleanUUID := strings.ReplaceAll(s.uuid, "-", "")
	uuidBytes, err := hex.DecodeString(cleanUUID)
	if err != nil || len(uuidBytes) != 16 {
		return nil, fmt.Errorf("invalid 16-byte hex UUID: %s", s.uuid)
	}

	data := make([]byte, 23)
	data[0] = 0x02 // iBeacon indicator type
	data[1] = 0x15 // length (21 bytes)
	copy(data[2:18], uuidBytes)
	binary.BigEndian.PutUint16(data[18:20], s.major)
	binary.BigEndian.PutUint16(data[20:22], s.minor)
	data[22] = byte(int8(s.txPower))

	return data, nil
}

func main() {
	actionFlag := flag.String("action", "interactive", "Action: start-tag, stop-tag, walk-in, walk-out, interactive")
	backendFlag := flag.String("backend", "ble", "Backend: ble (real radio) or dry-run")
	uuidFlag := flag.String("uuid", "fda50693-a4e2-4fb1-afcf-c6eb07647825", "iBeacon Proximity UUID")
	majorFlag := flag.Int("major", 100, "iBeacon Major value")
	minorFlag := flag.Int("minor", 1, "iBeacon Minor value")
	txPowerFlag := flag.Int("txpower", -59, "iBeacon measured TX power at 1 meter")
	intervalFlag := flag.Duration("interval", 500*time.Millisecond, "Advertising cadence")

	flag.Parse()

	sim := NewSimulator(*backendFlag, *uuidFlag, uint16(*majorFlag), uint16(*minorFlag), *txPowerFlag, *intervalFlag)

	switch *actionFlag {
	case "start-tag":
		if err := sim.StartAdvertising(); err != nil {
			log.Fatalf("Error starting tag: %v", err)
		}
		// Hold until interrupt
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		fmt.Println("Press Ctrl+C to stop advertising...")
		<-ch
		_ = sim.StopAdvertising()

	case "stop-tag":
		_ = sim.StopAdvertising()

	case "walk-in":
		fmt.Println("🚶 Simulating asset arrival into Operation Room...")
		if err := sim.StartAdvertising(); err != nil {
			log.Fatalf("Error: %v", err)
		}
		time.Sleep(10 * time.Second)
		_ = sim.StopAdvertising()

	case "walk-out":
		fmt.Println("🚪 Simulating asset departure from Operation Room...")
		_ = sim.StopAdvertising()
		fmt.Println("Waiting 6 seconds for absence threshold to expire on edge scanner...")
		time.Sleep(6 * time.Second)
		fmt.Println("Absence window complete.")

	case "interactive":
		runInteractiveCLI(sim)

	default:
		log.Fatalf("Unknown action: %s", *actionFlag)
	}
}

func runInteractiveCLI(sim *Simulator) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("==================================================")
	fmt.Println("📡 Bindu BLE Beacon Interactive Simulator")
	fmt.Println("Commands:")
	fmt.Println("  1 | on    - Start radio broadcast (Asset in room)")
	fmt.Println("  2 | off   - Stop radio broadcast (Asset left room)")
	fmt.Println("  3 | walk  - Walk-in for 15s then walk-out")
	fmt.Println("  q | quit  - Exit simulator")
	fmt.Println("==================================================")

	for {
		fmt.Print("ble-sim> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		cmd := strings.TrimSpace(strings.ToLower(line))
		switch cmd {
		case "1", "on", "start":
			if err := sim.StartAdvertising(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "2", "off", "stop":
			if err := sim.StopAdvertising(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "3", "walk":
			fmt.Println("🚶 Asset entering room...")
			_ = sim.StartAdvertising()
			fmt.Println("Holding presence for 15 seconds...")
			time.Sleep(15 * time.Second)
			fmt.Println("🚪 Asset leaving room...")
			_ = sim.StopAdvertising()
		case "q", "quit", "exit":
			_ = sim.StopAdvertising()
			fmt.Println("Exiting simulator.")
			return
		default:
			fmt.Println("Unknown command. Type: on, off, walk, or quit")
		}
	}
}
