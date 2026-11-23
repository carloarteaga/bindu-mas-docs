// Profile: ai-safe
// Bindu Modbus BPR Plugin
// Universal Modbus TCP Client and Semantic Event Translator for Industrial PLCs.

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"bindunet/sdk"
)

type ModbusPlugin struct {
	mu           sync.Mutex
	serverHost   string
	serverPort   int
	unitID       uint8
	timeout      time.Duration
	conn         net.Conn
	txCounter    uint32
	connected    bool
	nodeID       string
}

func NewModbusPlugin() *ModbusPlugin {
	return &ModbusPlugin{
		serverHost: "127.0.0.1",
		serverPort: 5020,
		unitID:     1,
		timeout:    2 * time.Second,
	}
}

func (p *ModbusPlugin) Handshake() sdk.HandshakeInfo {
	return sdk.HandshakeInfo{
		Inputs: []string{
			"ModbusConfig",
			"ModbusSet",
			"ModbusGet",
		},
		Outputs: []string{
			"ModbusStatus",
			"ModbusData",
		},
		Capabilities: []string{
			"modbus-tcp-client",
			"plc-semantic-translation",
		},
		Language:        "go",
		SDKVersion:      "bpr-sdk-go/0.1.0",
		ContractVersion: "bpr/v1",
	}
}

func (p *ModbusPlugin) OnStart(ctx *sdk.Context) error {
	p.nodeID = os.Getenv("BINDU_NODE_ID")
	if p.nodeID == "" {
		p.nodeID = "root"
	}
	ctx.Logger().Printf("[Modbus Plugin] Starting universal Modbus client on node %s...", p.nodeID)
	p.ensureConnection(ctx)
	return nil
}

func (p *ModbusPlugin) OnTrigger(ctx *sdk.Context, trigger sdk.Trigger) ([]sdk.Message, error) {
	ctx.Logger().Printf("[Modbus Plugin] Trigger received: %s (%s)", trigger.ID, trigger.Reason)
	return nil, nil
}

func (p *ModbusPlugin) OnEvent(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	switch event.MessageType {
	case "ModbusConfig":
		return p.handleConfig(ctx, event)
	case "ModbusSet":
		return p.handleSet(ctx, event)
	case "ModbusGet":
		return p.handleGet(ctx, event)
	default:
		ctx.Logger().Printf("[Modbus Plugin] Unhandled message type: %s", event.MessageType)
		return nil, nil
	}
}

func (p *ModbusPlugin) handleConfig(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fields := event.Payload
	if host, ok := fields["server_host"].(string); ok && host != "" {
		p.serverHost = host
	}
	if port, ok := fields["server_port"].(float64); ok && port > 0 {
		p.serverPort = int(port)
	}
	if unit, ok := fields["unit_id"].(float64); ok {
		p.unitID = uint8(unit)
	}
	if timeoutMs, ok := fields["timeout_ms"].(float64); ok && timeoutMs > 0 {
		p.timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	ctx.Logger().Printf("[Modbus Plugin] Updated config: %s:%d (Unit %d, Timeout %v)", p.serverHost, p.serverPort, p.unitID, p.timeout)
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
		p.connected = false
	}

	p.connectLocked(ctx)
	return nil, nil
}

func (p *ModbusPlugin) handleSet(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fields := event.Payload
	targetDevice, _ := fields["target_device"].(string)
	registerType, _ := fields["register_type"].(string)
	if registerType == "" {
		registerType = "holding"
	}
	regAddrFloat, _ := fields["register_address"].(float64)
	regAddr := uint16(regAddrFloat)

	valFloat, _ := fields["value"].(float64)
	scaleFactor, hasScale := fields["scale_factor"].(float64)
	if hasScale && scaleFactor > 0 {
		valFloat = valFloat * scaleFactor
	}

	var writeErr error
	if registerType == "coil" {
		coilVal := valFloat != 0
		writeErr = p.writeSingleCoil(ctx, regAddr, coilVal)
	} else {
		regVal := uint16(valFloat)
		writeErr = p.writeSingleRegister(ctx, regAddr, regVal)
	}

	statusStr := "SUCCESS"
	var errStr string
	if writeErr != nil {
		statusStr = "ERROR"
		errStr = writeErr.Error()
		ctx.Logger().Printf("[Modbus Plugin] ❌ Write failed to %s reg %d: %v", targetDevice, regAddr, writeErr)
	} else {
		ctx.Logger().Printf("[Modbus Plugin] ✅ Write success to %s reg %d = %.1f", targetDevice, regAddr, valFloat)
	}

	statusMsg := sdk.Message{
		Type: "ModbusStatus",
		Payload: map[string]any{
			"command_id":       fmt.Sprintf("cmd-%d", time.Now().UnixNano()),
			"target_device":    targetDevice,
			"register_address": int(regAddr),
			"written_value":    valFloat,
			"status":           statusStr,
			"error_message":    errStr,
			"timestamp":        time.Now().UTC().Format(time.RFC3339),
		},
	}

	return []sdk.Message{statusMsg}, nil
}

func (p *ModbusPlugin) handleGet(ctx *sdk.Context, event sdk.Event) ([]sdk.Message, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fields := event.Payload
	targetDevice, _ := fields["target_device"].(string)
	registerType, _ := fields["register_type"].(string)
	if registerType == "" {
		registerType = "holding"
	}
	regAddrFloat, _ := fields["register_address"].(float64)
	regAddr := uint16(regAddrFloat)
	qtyFloat, _ := fields["quantity"].(float64)
	qty := uint16(qtyFloat)
	if qty == 0 {
		qty = 1
	}
	reqID, _ := fields["request_id"].(string)
	if reqID == "" {
		reqID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	scaleFactor, hasScale := fields["scale_factor"].(float64)

	vals, readErr := p.readRegisters(ctx, regAddr, qty, registerType == "input")
	if readErr != nil {
		ctx.Logger().Printf("[Modbus Plugin] ❌ Read failed from %s reg %d: %v", targetDevice, regAddr, readErr)
		return nil, readErr
	}

	rawVals := make([]int, len(vals))
	for i, v := range vals {
		rawVals[i] = int(v)
	}

	payload := map[string]any{
		"request_id":       reqID,
		"target_device":    targetDevice,
		"register_address": int(regAddr),
		"raw_values":       rawVals,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}

	if len(rawVals) > 0 {
		scaled := float64(rawVals[0])
		if hasScale && scaleFactor > 0 {
			scaled = scaled / scaleFactor
		}
		payload["scaled_value"] = scaled
	}

	dataMsg := sdk.Message{
		Type:    "ModbusData",
		Payload: payload,
	}

	return []sdk.Message{dataMsg}, nil
}

func (p *ModbusPlugin) ensureConnection(ctx *sdk.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.connectLocked(ctx)
}

func (p *ModbusPlugin) connectLocked(ctx *sdk.Context) error {
	if p.conn != nil {
		return nil
	}
	addr := fmt.Sprintf("%s:%d", p.serverHost, p.serverPort)
	conn, err := net.DialTimeout("tcp", addr, p.timeout)
	if err != nil {
		ctx.Logger().Printf("[Modbus Plugin] Warning: could not connect to Modbus server at %s: %v", addr, err)
		return err
	}
	p.conn = conn
	p.connected = true
	ctx.Logger().Printf("[Modbus Plugin] Connected to Modbus server at %s", addr)
	return nil
}

func (p *ModbusPlugin) sendMBAP(ctx *sdk.Context, pdu []byte) ([]byte, error) {
	if err := p.connectLocked(ctx); err != nil {
		return nil, err
	}

	txID := uint16(atomic.AddUint32(&p.txCounter, 1) & 0xFFFF)
	length := uint16(len(pdu) + 1)

	req := make([]byte, 7+len(pdu))
	binary.BigEndian.PutUint16(req[0:2], txID)
	binary.BigEndian.PutUint16(req[2:4], 0)
	binary.BigEndian.PutUint16(req[4:6], length)
	req[6] = p.unitID
	copy(req[7:], pdu)

	_ = p.conn.SetDeadline(time.Now().Add(p.timeout))
	if _, err := p.conn.Write(req); err != nil {
		p.conn.Close()
		p.conn = nil
		p.connected = false
		return nil, fmt.Errorf("write error: %w", err)
	}

	header := make([]byte, 7)
	if _, err := io.ReadFull(p.conn, header); err != nil {
		p.conn.Close()
		p.conn = nil
		p.connected = false
		return nil, fmt.Errorf("read header error: %w", err)
	}

	respTxID := binary.BigEndian.Uint16(header[0:2])
	if respTxID != txID {
		return nil, fmt.Errorf("txID mismatch: expected %d got %d", txID, respTxID)
	}

	respLen := binary.BigEndian.Uint16(header[4:6])
	if respLen < 2 {
		return nil, fmt.Errorf("invalid response length: %d", respLen)
	}

	respPDU := make([]byte, respLen-1)
	if _, err := io.ReadFull(p.conn, respPDU); err != nil {
		p.conn.Close()
		p.conn = nil
		p.connected = false
		return nil, fmt.Errorf("read PDU error: %w", err)
	}

	if respPDU[0]&0x80 != 0 {
		return nil, fmt.Errorf("modbus exception code: 0x%02X", respPDU[1])
	}

	return respPDU, nil
}

func (p *ModbusPlugin) writeSingleRegister(ctx *sdk.Context, addr uint16, val uint16) error {
	pdu := make([]byte, 5)
	pdu[0] = 0x06 // Write Single Register
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], val)

	_, err := p.sendMBAP(ctx, pdu)
	return err
}

func (p *ModbusPlugin) writeSingleCoil(ctx *sdk.Context, addr uint16, val bool) error {
	pdu := make([]byte, 5)
	pdu[0] = 0x05 // Write Single Coil
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	if val {
		binary.BigEndian.PutUint16(pdu[3:5], 0xFF00)
	} else {
		binary.BigEndian.PutUint16(pdu[3:5], 0x0000)
	}

	_, err := p.sendMBAP(ctx, pdu)
	return err
}

func (p *ModbusPlugin) readRegisters(ctx *sdk.Context, addr uint16, qty uint16, isInput bool) ([]uint16, error) {
	pdu := make([]byte, 5)
	if isInput {
		pdu[0] = 0x04 // Read Input Registers
	} else {
		pdu[0] = 0x03 // Read Holding Registers
	}
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], qty)

	resp, err := p.sendMBAP(ctx, pdu)
	if err != nil {
		return nil, err
	}

	if len(resp) < 2 {
		return nil, fmt.Errorf("short response")
	}

	byteCount := int(resp[1])
	if len(resp) < 2+byteCount || byteCount != int(qty*2) {
		return nil, fmt.Errorf("unexpected byte count %d for quantity %d", byteCount, qty)
	}

	results := make([]uint16, qty)
	for i := uint16(0); i < qty; i++ {
		results[i] = binary.BigEndian.Uint16(resp[2+i*2 : 4+i*2])
	}
	return results, nil
}

func (p *ModbusPlugin) Shutdown(ctx *sdk.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.conn != nil {
		p.conn.Close()
		p.conn = nil
	}
	ctx.Logger().Printf("[Modbus Plugin] Shutdown complete")
	return nil
}

func main() {
	plugin := NewModbusPlugin()
	if err := sdk.Run(plugin); err != nil {
		log.Fatalf("Modbus plugin terminated: %v", err)
	}
}
