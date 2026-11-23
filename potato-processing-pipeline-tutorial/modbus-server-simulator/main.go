// Standalone Modbus TCP Server Simulator for Bindu Potato Processing Pipeline
// Pure Go implementation of Modbus TCP MBAP protocol and register handling.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ANSI Color Codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// Modbus Function Codes
const (
	FuncReadCoils              = 0x01
	FuncReadDiscreteInputs     = 0x02
	FuncReadHoldingRegisters   = 0x03
	FuncReadInputRegisters     = 0x04
	FuncWriteSingleCoil        = 0x05
	FuncWriteSingleRegister    = 0x06
	FuncWriteMultipleRegisters = 0x10
)

// Well-known Register Names & Descriptions
var registerNames = map[uint16]string{
	100: "Silo_Temp_SP (°C * 10)",
	101: "Silo_Duration_SP (days)",
	102: "Silo_Fan_Speed_SP (RPM)",
	200: "Fryer_Oil_Temp_SP (°C * 10)",
	201: "Fryer_Belt_Speed_SP (mm/s)",
	202: "Fryer_Exhaust_Blower_SP (%)",
	300: "Truck_Avg_Transit_Temp (°C * 10)",
	301: "Truck_Transit_Duration (hours)",
}

type ModbusServer struct {
	port      int
	unitID    uint8
	listener  net.Listener
	mu        sync.RWMutex
	holding   map[uint16]uint16
	inputs    map[uint16]uint16
	coils     map[uint16]bool
	discretes map[uint16]bool
	stopChan  chan struct{}
}

func NewModbusServer(port int, unitID uint8) *ModbusServer {
	s := &ModbusServer{
		port:      port,
		unitID:    unitID,
		holding:   make(map[uint16]uint16),
		inputs:    make(map[uint16]uint16),
		coils:     make(map[uint16]bool),
		discretes: make(map[uint16]bool),
		stopChan:  make(chan struct{}),
	}

	// Initialize default values for simulation
	s.holding[100] = 180 // 18.0 °C default silo temperature
	s.holding[101] = 0   // 0 days default
	s.holding[200] = 1750 // 175.0 °C default oil temp
	s.holding[201] = 30   // 30 mm/s default belt speed
	s.inputs[300] = 65   // 6.5 °C default truck temperature
	s.inputs[301] = 18   // 18 hours default

	return s
}

func (s *ModbusServer) Start() error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = l

	fmt.Printf("%s%s[MODBUS PLC SIMULATOR] Listening on %s (Unit ID: %d)%s\n", ColorBold, ColorGreen, addr, s.unitID, ColorReset)
	fmt.Printf("%s[MODBUS PLC SIMULATOR] Pre-loaded Registers:%s\n", ColorCyan, ColorReset)
	for reg, val := range s.holding {
		name := registerNames[reg]
		fmt.Printf("  • Holding Reg %3d: %5d  [%s]\n", reg, val, name)
	}

	go s.acceptLoop()
	return nil
}

func (s *ModbusServer) Stop() {
	close(s.stopChan)
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *ModbusServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopChan:
				return
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *ModbusServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	fmt.Printf("%s[%s] Client connected: %s%s\n", ColorBlue, time.Now().Format("15:04:05.000"), remote, ColorReset)

	buf := make([]byte, 512)
	for {
		// Read MBAP Header (7 bytes: TxID(2), ProtoID(2), Length(2), UnitID(1))
		_, err := io.ReadFull(conn, buf[:7])
		if err != nil {
			if err != io.EOF {
				fmt.Printf("%s[%s] Connection closed from %s: %v%s\n", ColorYellow, time.Now().Format("15:04:05.000"), remote, err, ColorReset)
			}
			return
		}

		txID := binary.BigEndian.Uint16(buf[0:2])
		protoID := binary.BigEndian.Uint16(buf[2:4])
		length := binary.BigEndian.Uint16(buf[4:6])
		unitID := buf[6]

		if protoID != 0 {
			fmt.Printf("%s[%s] Invalid protocol ID: %d%s\n", ColorRed, time.Now().Format("15:04:05.000"), protoID, ColorReset)
			return
		}

		if length < 2 || length > 256 {
			fmt.Printf("%s[%s] Invalid length: %d%s\n", ColorRed, time.Now().Format("15:04:05.000"), length, ColorReset)
			return
		}

		pduLen := int(length - 1)
		_, err = io.ReadFull(conn, buf[7:7+pduLen])
		if err != nil {
			log.Printf("Failed reading PDU: %v", err)
			return
		}

		pdu := buf[7 : 7+pduLen]
		funcCode := pdu[0]

		respPDU := s.processPDU(remote, funcCode, pdu[1:], unitID)

		// Build response MBAP Header
		respLen := uint16(len(respPDU) + 1)
		respMBAP := make([]byte, 7)
		binary.BigEndian.PutUint16(respMBAP[0:2], txID)
		binary.BigEndian.PutUint16(respMBAP[2:4], 0)
		binary.BigEndian.PutUint16(respMBAP[4:6], respLen)
		respMBAP[6] = unitID

		_, err = conn.Write(append(respMBAP, respPDU...))
		if err != nil {
			log.Printf("Failed writing response: %v", err)
			return
		}
	}
}

func (s *ModbusServer) processPDU(remote string, funcCode byte, data []byte, unitID byte) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Format("15:04:05.000")

	switch funcCode {
	case FuncReadHoldingRegisters:
		if len(data) < 4 {
			return []byte{funcCode | 0x80, 0x03} // Illegal Data Value
		}
		startAddr := binary.BigEndian.Uint16(data[0:2])
		quantity := binary.BigEndian.Uint16(data[2:4])

		fmt.Printf("%s[%s] 📥 READ_HOLDING_REGISTERS (0x03) | From: %s | Start: %d | Count: %d%s\n",
			ColorCyan, now, remote, startAddr, quantity, ColorReset)

		byteCount := byte(quantity * 2)
		resp := make([]byte, 2+byteCount)
		resp[0] = funcCode
		resp[1] = byteCount

		for i := uint16(0); i < quantity; i++ {
			addr := startAddr + i
			val := s.holding[addr]
			name := registerNames[addr]
			if name != "" {
				fmt.Printf("   ├─ Reg %d (%s) -> %d\n", addr, name, val)
			}
			binary.BigEndian.PutUint16(resp[2+i*2:4+i*2], val)
		}
		return resp

	case FuncReadInputRegisters:
		if len(data) < 4 {
			return []byte{funcCode | 0x80, 0x03}
		}
		startAddr := binary.BigEndian.Uint16(data[0:2])
		quantity := binary.BigEndian.Uint16(data[2:4])

		fmt.Printf("%s[%s] 📥 READ_INPUT_REGISTERS (0x04) | From: %s | Start: %d | Count: %d%s\n",
			ColorCyan, now, remote, startAddr, quantity, ColorReset)

		byteCount := byte(quantity * 2)
		resp := make([]byte, 2+byteCount)
		resp[0] = funcCode
		resp[1] = byteCount

		for i := uint16(0); i < quantity; i++ {
			addr := startAddr + i
			val := s.inputs[addr]
			binary.BigEndian.PutUint16(resp[2+i*2:4+i*2], val)
		}
		return resp

	case FuncWriteSingleRegister:
		if len(data) < 4 {
			return []byte{funcCode | 0x80, 0x03}
		}
		addr := binary.BigEndian.Uint16(data[0:2])
		val := binary.BigEndian.Uint16(data[2:4])

		s.holding[addr] = val
		name := registerNames[addr]

		var humanVal string
		if addr == 100 || addr == 200 || addr == 300 {
			humanVal = fmt.Sprintf("%.1f °C", float64(val)/10.0)
		} else if addr == 201 {
			humanVal = fmt.Sprintf("%d mm/s", val)
		} else if addr == 101 {
			humanVal = fmt.Sprintf("%d days", val)
		} else {
			humanVal = fmt.Sprintf("%d", val)
		}

		fmt.Printf("%s%s[%s] ⚡ WRITE_SINGLE_REGISTER (0x06) | From: %s | Reg: %d [%s] = %d (%s)%s\n",
			ColorBold, ColorYellow, now, remote, addr, name, val, humanVal, ColorReset)

		resp := make([]byte, 5)
		resp[0] = funcCode
		binary.BigEndian.PutUint16(resp[1:3], addr)
		binary.BigEndian.PutUint16(resp[3:5], val)
		return resp

	case FuncWriteMultipleRegisters:
		if len(data) < 5 {
			return []byte{funcCode | 0x80, 0x03}
		}
		startAddr := binary.BigEndian.Uint16(data[0:2])
		quantity := binary.BigEndian.Uint16(data[2:4])
		byteCount := data[4]

		if int(byteCount) != int(quantity*2) || len(data) < 5+int(byteCount) {
			return []byte{funcCode | 0x80, 0x03}
		}

		fmt.Printf("%s%s[%s] ⚡ WRITE_MULTIPLE_REGISTERS (0x10) | From: %s | Start: %d | Count: %d%s\n",
			ColorBold, ColorYellow, now, remote, startAddr, quantity, ColorReset)

		for i := uint16(0); i < quantity; i++ {
			addr := startAddr + i
			val := binary.BigEndian.Uint16(data[5+i*2 : 7+i*2])
			s.holding[addr] = val
			name := registerNames[addr]
			fmt.Printf("   ├─ Reg %d [%s] = %d\n", addr, name, val)
		}

		resp := make([]byte, 5)
		resp[0] = funcCode
		binary.BigEndian.PutUint16(resp[1:3], startAddr)
		binary.BigEndian.PutUint16(resp[3:5], quantity)
		return resp

	case FuncReadCoils:
		if len(data) < 4 {
			return []byte{funcCode | 0x80, 0x03}
		}
		startAddr := binary.BigEndian.Uint16(data[0:2])
		quantity := binary.BigEndian.Uint16(data[2:4])
		byteCount := byte((quantity + 7) / 8)
		resp := make([]byte, 2+byteCount)
		resp[0] = funcCode
		resp[1] = byteCount
		for i := uint16(0); i < quantity; i++ {
			if s.coils[startAddr+i] {
				resp[2+i/8] |= (1 << (i % 8))
			}
		}
		return resp

	case FuncWriteSingleCoil:
		if len(data) < 4 {
			return []byte{funcCode | 0x80, 0x03}
		}
		addr := binary.BigEndian.Uint16(data[0:2])
		val := binary.BigEndian.Uint16(data[2:4])
		coilState := (val == 0xFF00)
		s.coils[addr] = coilState
		fmt.Printf("%s[%s] ⚡ WRITE_SINGLE_COIL (0x05) | From: %s | Coil: %d = %v%s\n",
			ColorYellow, now, remote, addr, coilState, ColorReset)

		resp := make([]byte, 5)
		resp[0] = funcCode
		binary.BigEndian.PutUint16(resp[1:3], addr)
		binary.BigEndian.PutUint16(resp[3:5], val)
		return resp

	default:
		fmt.Printf("%s[%s] ❌ Unsupported Modbus Function Code: 0x%02X%s\n", ColorRed, now, funcCode, ColorReset)
		return []byte{funcCode | 0x80, 0x01} // Illegal Function
	}
}

func main() {
	port := flag.Int("port", 5020, "Modbus TCP server listening port")
	unitID := flag.Int("unit-id", 1, "Modbus Unit ID / Slave ID")
	flag.Parse()

	server := NewModbusServer(*port, uint8(*unitID))
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start Modbus server: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Printf("\n%s[MODBUS PLC SIMULATOR] Shutting down gracefully...%s\n", ColorYellow, ColorReset)
	server.Stop()
}
