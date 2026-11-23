// Profile: ai-safe
// Bindu Industrial RFID LLRP Mock Reader & 4-Antenna Dock Portal Simulator
// Implements standard LLRP TCP server (port 5084) streaming realistic 11-standard EPC tag reads across 4 antennas.

package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// LLRP Protocol Constants
const (
	MsgGetReaderCapabilities         uint16 = 1
	MsgGetReaderCapabilitiesResponse uint16 = 11
	MsgAddROSpec                     uint16 = 20
	MsgAddROSpecResponse             uint16 = 30
	MsgDeleteROSpec                  uint16 = 21
	MsgDeleteROSpecResponse          uint16 = 31
	MsgStartROSpec                   uint16 = 22
	MsgStartROSpecResponse           uint16 = 32
	MsgStopROSpec                    uint16 = 23
	MsgStopROSpecResponse            uint16 = 33
	MsgEnableROSpec                  uint16 = 24
	MsgEnableROSpecResponse          uint16 = 34
	MsgROAccessReport                uint16 = 61
	MsgKeepalive                     uint16 = 62
	MsgKeepaliveAck                  uint16 = 72
)

// LLRP Parameter Constants
const (
	ParamTagReportData         uint16 = 240
	ParamEPC96                 uint16 = 13 // TV
	ParamAntennaID             uint16 = 1  // TV
	ParamFirstSeenTimestampUTC uint16 = 2  // TV
	ParamLastSeenTimestampUTC  uint16 = 3  // TV
	ParamPeakRSSI              uint16 = 6  // TV
	ParamChannelIndex          uint16 = 7  // TV
	ParamTagSeenCount          uint16 = 8  // TV
)

// Precomputed 96-bit HEX codes representing the 11 EPC Standards
type SampleEPCTag struct {
	Standard    string
	Description string
	Hex         string
}

var sampleTags = []SampleEPCTag{
	// 1. SGTIN-96: Filter 3, Partition 5, Company 0614141, Item 107346, Serial 2026
	{Standard: "SGTIN-96", Description: "Case Pack - Cold-Pressed Juice 1L", Hex: "3074257bf419cb80000007ea"},
	// 2. SSCC-96: Filter 1, Partition 4, Company 0614141, Extension 0, Serial 12345678
	{Standard: "SSCC-96", Description: "Master Logistics Pallet (SSCC)", Hex: "3144257bf400bc614e000000"},
	// 3. SGLN-96: Filter 0, Partition 5, Company 0614141, Location 4001, Ext 0
	{Standard: "SGLN-96", Description: "Dock Door #4 Berth Location Marker", Hex: "3214257bf400fa1000000000"},
	// 4. GRAI-96: Filter 2, Partition 5, Company 0614141, Asset 5544, Serial 8899
	{Standard: "GRAI-96", Description: "Reusable Insulated Euro-Tote Container", Hex: "3354257bf4015a80000022c3"},
	// 5. GIAI-96: Filter 0, Partition 5, Company 0614141, Individual Asset 998877
	{Standard: "GIAI-96", Description: "Electric Reach Forklift #09 (Fleet Asset)", Hex: "3414257bf400f3dd00000000"},
	// 6. GSRN-96: Filter 0, Partition 5, Company 0614141, Service Ref 112233
	{Standard: "GSRN-96", Description: "Certified Forklift Operator RFID Badge", Hex: "3514257bf40001b669000000"},
	// 7. GDTI-96: Filter 0, Partition 5, Company 0614141, Document 7766, Serial 5544
	{Standard: "GDTI-96", Description: "Digital Customs Phytosanitary Certificate", Hex: "3c14257bf401e560000015a8"},
	// 8. CPI-96: Filter 0, Partition 5, Company 0614141, Component 4321, Serial 98765
	{Standard: "CPI-96", Description: "Automotive Chilled Refrigeration Compressor", Hex: "3d14257bf4010e10000181cd"},
	// 9. SGCN-96: Filter 0, Partition 5, Company 0614141, Coupon 8877, Serial 1234
	{Standard: "SGCN-96", Description: "High-Value Supplier Rebate Voucher", Hex: "3e14257bf4022ad0000004d2"},
	// 10. ITIP-96: Filter 0, Partition 5, Company 0614141, Piece Item 6543, Serial 2211
	{Standard: "ITIP-96", Description: "Corrugated Pallet Wrap Film Reel (Piece Good)", Hex: "3814257bf40198f0000008a3"},
	// 11. GID-96: GenMgr 123456, ObjClass 654321, Serial 99887766
	{Standard: "GID-96", Description: "Industrial Raw Steel Casing (General ID)", Hex: "350001e24009fbf105f42696"},
}

// EncodeLLRPMessage builds a standard binary framed LLRP message
func EncodeLLRPMessage(msgType uint16, msgID uint32, payload []byte) []byte {
	totalLen := 10 + len(payload)
	buf := make([]byte, totalLen)
	firstWord := (uint16(1) << 10) | (msgType & 0x03FF)
	binary.BigEndian.PutUint16(buf[0:2], firstWord)
	binary.BigEndian.PutUint32(buf[2:6], uint32(totalLen))
	binary.BigEndian.PutUint32(buf[6:10], msgID)
	if len(payload) > 0 {
		copy(buf[10:], payload)
	}
	return buf
}

// ReadLLRPMessage reads exactly one frame from a stream
func ReadLLRPMessage(r io.Reader) (uint16, uint32, []byte, error) {
	headerBuf := make([]byte, 10)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return 0, 0, nil, err
	}
	firstWord := binary.BigEndian.Uint16(headerBuf[0:2])
	msgType := firstWord & 0x03FF
	length := binary.BigEndian.Uint32(headerBuf[2:6])
	msgID := binary.BigEndian.Uint32(headerBuf[6:10])

	if length < 10 {
		return 0, 0, nil, fmt.Errorf("invalid frame length: %d", length)
	}
	payloadLen := int(length) - 10
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return 0, 0, nil, err
		}
	}
	return msgType, msgID, payload, nil
}

// BuildTagReportDataTLV builds a TagReportData (Type 240) TLV containing standard TV parameters
func BuildTagReportDataTLV(epcHex string, antennaID uint16, rssi float64, channelIdx uint16) []byte {
	epcBytes, _ := hex.DecodeString(epcHex)
	if len(epcBytes) != 12 {
		epcBytes = make([]byte, 12)
	}

	var pBuf []byte

	// 1. EPC-96 TV parameter (Type 13, 1 byte header + 12 bytes = 13 bytes)
	pBuf = append(pBuf, 0x80|byte(ParamEPC96))
	pBuf = append(pBuf, epcBytes...)

	// 2. AntennaID TV parameter (Type 1, 1 byte header + 2 bytes = 3 bytes)
	pBuf = append(pBuf, 0x80|byte(ParamAntennaID))
	antBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(antBytes, antennaID)
	pBuf = append(pBuf, antBytes...)

	// 3. PeakRSSI TV parameter (Type 6, 1 byte header + 1 byte int8 = 2 bytes)
	pBuf = append(pBuf, 0x80|byte(ParamPeakRSSI))
	pBuf = append(pBuf, byte(int8(rssi)))

	// 4. ChannelIndex TV parameter (Type 7, 1 byte header + 2 bytes = 3 bytes)
	pBuf = append(pBuf, 0x80|byte(ParamChannelIndex))
	chanBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(chanBytes, channelIdx)
	pBuf = append(pBuf, chanBytes...)

	// 5. TagSeenCount TV parameter (Type 8, 1 byte header + 2 bytes = 3 bytes)
	pBuf = append(pBuf, 0x80|byte(ParamTagSeenCount))
	countBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(countBytes, 1)
	pBuf = append(pBuf, countBytes...)

	// 6. FirstSeenTimestampUTC TV parameter (Type 2, 1 byte header + 8 bytes = 9 bytes)
	nowMicros := uint64(time.Now().UTC().UnixMicro())
	pBuf = append(pBuf, 0x80|byte(ParamFirstSeenTimestampUTC))
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, nowMicros)
	pBuf = append(pBuf, timeBytes...)

	// 7. LastSeenTimestampUTC TV parameter (Type 3, 1 byte header + 8 bytes = 9 bytes)
	pBuf = append(pBuf, 0x80|byte(ParamLastSeenTimestampUTC))
	pBuf = append(pBuf, timeBytes...)

	// Wrap in TagReportData TLV (Type 240)
	tlvLen := 4 + len(pBuf)
	tlvBuf := make([]byte, tlvLen)
	binary.BigEndian.PutUint16(tlvBuf[0:2], ParamTagReportData)
	binary.BigEndian.PutUint16(tlvBuf[2:4], uint16(tlvLen))
	copy(tlvBuf[4:], pBuf)

	return tlvBuf
}

type DockPortalSimulator struct {
	listenAddr string
	listener   net.Listener
	mu         sync.Mutex
	clients    map[net.Conn]bool
	streaming  bool
	stopChan   chan struct{}
}

func NewDockPortalSimulator(addr string) *DockPortalSimulator {
	return &DockPortalSimulator{
		listenAddr: addr,
		clients:    make(map[net.Conn]bool),
		stopChan:   make(chan struct{}),
	}
}

func (s *DockPortalSimulator) Start() error {
	l, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	s.listener = l
	log.Printf("=================================================================")
	log.Printf("📡 BINDU LLRP RFID READER & 4-ANTENNA DOCK PORTAL SIMULATOR")
	log.Printf("   Listening on TCP %s (LLRP Standard Port)", s.listenAddr)
	log.Printf("   Simulating 4 Antennas (1-2: Inbound, 3-4: Outbound)")
	log.Printf("   Simulating all 11 GS1 / EPC Global 96-bit Standards")
	log.Printf("=================================================================")

	go s.acceptLoop()
	go s.transitSimulationLoop()
	return nil
}

func (s *DockPortalSimulator) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return
			default:
				log.Printf("[Simulator] Accept error: %v", err)
				return
			}
		}

		s.mu.Lock()
		s.clients[conn] = true
		s.mu.Unlock()

		log.Printf("[Simulator] Client connected: %s", conn.RemoteAddr())
		go s.handleClient(conn)
	}
}

func (s *DockPortalSimulator) handleClient(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
		log.Printf("[Simulator] Client disconnected: %s", conn.RemoteAddr())
	}()

	for {
		msgType, msgID, _, err := ReadLLRPMessage(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("[Simulator] Read error from %s: %v", conn.RemoteAddr(), err)
			}
			return
		}

		switch msgType {
		case MsgGetReaderCapabilities:
			log.Printf("[Simulator] <-- GET_READER_CAPABILITIES (ID: %d)", msgID)
			resp := EncodeLLRPMessage(MsgGetReaderCapabilitiesResponse, msgID, nil)
			_, _ = conn.Write(resp)

		case MsgAddROSpec:
			log.Printf("[Simulator] <-- ADD_ROSPEC (ID: %d)", msgID)
			statusBuf := []byte{0x00, 0x00} // Success
			resp := EncodeLLRPMessage(MsgAddROSpecResponse, msgID, statusBuf)
			_, _ = conn.Write(resp)

		case MsgEnableROSpec:
			log.Printf("[Simulator] <-- ENABLE_ROSPEC (ID: %d)", msgID)
			statusBuf := []byte{0x00, 0x00}
			resp := EncodeLLRPMessage(MsgEnableROSpecResponse, msgID, statusBuf)
			_, _ = conn.Write(resp)

		case MsgStartROSpec:
			log.Printf("[Simulator] <-- START_ROSPEC (ID: %d) -> Starting RF Tag Streaming!", msgID)
			s.mu.Lock()
			s.streaming = true
			s.mu.Unlock()
			statusBuf := []byte{0x00, 0x00}
			resp := EncodeLLRPMessage(MsgStartROSpecResponse, msgID, statusBuf)
			_, _ = conn.Write(resp)

		case MsgStopROSpec:
			log.Printf("[Simulator] <-- STOP_ROSPEC (ID: %d)", msgID)
			s.mu.Lock()
			s.streaming = false
			s.mu.Unlock()
			resp := EncodeLLRPMessage(MsgStopROSpecResponse, msgID, nil)
			_, _ = conn.Write(resp)

		case MsgKeepaliveAck:
			// Client acknowledged keepalive
		default:
			log.Printf("[Simulator] Unhandled LLRP command type %d (ID: %d)", msgType, msgID)
		}
	}
}

// transitSimulationLoop executes periodic realistic forklift passes through the 4-antenna portal
func (s *DockPortalSimulator) transitSimulationLoop() {
	transitTicker := time.NewTicker(4 * time.Second)
	defer transitTicker.Stop()

	passCounter := 0

	for {
		select {
		case <-s.stopChan:
			return
		case <-transitTicker.C:
			s.mu.Lock()
			numClients := len(s.clients)
			s.mu.Unlock()

			if numClients == 0 {
				continue
			}

			passCounter++
			direction := "INBOUND"
			if passCounter%2 == 0 {
				direction = "OUTBOUND"
			}

			log.Printf("\n🚚 [Simulator] --- SIMULATING FORKLIFT TRANSIT #%d (%s) ---", passCounter, direction)
			s.simulateForkliftTransit(direction)
		}
	}
}

func (s *DockPortalSimulator) simulateForkliftTransit(direction string) {
	// Antennas sequence based on direction:
	// Inbound: Ant 1 & 2 first (Entry), then Ant 3 & 4 (Exit)
	// Outbound: Ant 3 & 4 first (Entry), then Ant 1 & 2 (Exit)
	var firstAnts, secondAnts []uint16
	if direction == "INBOUND" {
		firstAnts = []uint16{1, 2}
		secondAnts = []uint16{3, 4}
	} else {
		firstAnts = []uint16{3, 4}
		secondAnts = []uint16{1, 2}
	}

	// 1. First Gate Antennas Read
	for _, antID := range firstAnts {
		var reportPayload []byte
		// Pick 3-5 sample tags across all standards to simulate pallet + cartons + assets
		numTags := 3 + rand.Intn(4)
		for i := 0; i < numTags; i++ {
			tagIdx := (i + rand.Intn(len(sampleTags))) % len(sampleTags)
			tag := sampleTags[tagIdx]
			rssi := -48.0 - rand.Float64()*15.0 // Strong entry RSSI
			chanIdx := uint16(1 + rand.Intn(16))

			tlv := BuildTagReportDataTLV(tag.Hex, antID, rssi, chanIdx)
			reportPayload = append(reportPayload, tlv...)
		}
		s.broadcastReport(reportPayload)
	}

	// Physical Transit Delay (Forklift moving at ~1.5 m/s through the 2m portal)
	time.Sleep(350 * time.Millisecond)

	// 2. Second Gate Antennas Read
	for _, antID := range secondAnts {
		var reportPayload []byte
		numTags := 3 + rand.Intn(4)
		for i := 0; i < numTags; i++ {
			tagIdx := (i + rand.Intn(len(sampleTags))) % len(sampleTags)
			tag := sampleTags[tagIdx]
			rssi := -52.0 - rand.Float64()*12.0 // Strong exit RSSI
			chanIdx := uint16(1 + rand.Intn(16))

			tlv := BuildTagReportDataTLV(tag.Hex, antID, rssi, chanIdx)
			reportPayload = append(reportPayload, tlv...)
		}
		s.broadcastReport(reportPayload)
	}

	// 3. Sporadic Stray Tag in Staging Area (Static, low RSSI on Ant 1 only)
	if rand.Float64() < 0.4 {
		strayTag := sampleTags[0] // SGTIN carton sitting nearby
		tlv := BuildTagReportDataTLV(strayTag.Hex, 1, -79.0, 5)
		s.broadcastReport(tlv)
	}
}

func (s *DockPortalSimulator) broadcastReport(payload []byte) {
	if len(payload) == 0 {
		return
	}
	frame := EncodeLLRPMessage(MsgROAccessReport, 0, payload)

	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.clients {
		_, _ = conn.Write(frame)
	}
}

func (s *DockPortalSimulator) Stop() {
	close(s.stopChan)
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.mu.Unlock()
}

func main() {
	port := flag.Int("port", 5084, "TCP port to listen on for LLRP connections")
	host := flag.String("host", "0.0.0.0", "Host IP to bind LLRP server")
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	sim := NewDockPortalSimulator(addr)

	if err := sim.Start(); err != nil {
		log.Fatalf("Failed to start dock portal simulator: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("[Simulator] Shutting down...")
	sim.Stop()
	log.Printf("[Simulator] Stopped.")
}
