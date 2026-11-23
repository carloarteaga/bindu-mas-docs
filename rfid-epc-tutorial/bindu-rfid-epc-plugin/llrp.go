package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"time"
)

// LLRP Message Type Constants
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
	MsgReaderEventNotification       uint16 = 63
)

// LLRP Parameter Type Constants
const (
	ParamTagReportData         uint16 = 240
	ParamEPCData               uint16 = 241
	ParamEPC96                 uint16 = 13 // TV Parameter
	ParamAntennaID             uint16 = 1  // TV Parameter
	ParamFirstSeenTimestampUTC uint16 = 2  // TV Parameter
	ParamLastSeenTimestampUTC  uint16 = 3  // TV Parameter
	ParamPeakRSSI              uint16 = 6  // TV Parameter
	ParamChannelIndex          uint16 = 7  // TV Parameter
	ParamTagSeenCount          uint16 = 8  // TV Parameter
	ParamROSpecID              uint16 = 9  // TV Parameter
)

// LLRPHeader represents the 10-byte standard LLRP frame header
type LLRPHeader struct {
	Version uint8
	Type    uint16
	Length  uint32
	ID      uint32
}

// TagReportItem holds raw decoded attributes from an LLRP TagReportData parameter
type TagReportItem struct {
	EPCHex        string
	AntennaID     int
	PeakRSSI      float64
	ChannelIndex  int
	ReadCount     int
	FirstSeenTime time.Time
	LastSeenTime  time.Time
}

// EncodeLLRPMessage builds a binary framed LLRP message
func EncodeLLRPMessage(msgType uint16, msgID uint32, payload []byte) []byte {
	totalLen := 10 + len(payload)
	buf := make([]byte, totalLen)

	// Version (bits 0-2: 0x01) and Type (bits 6-15)
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
func ReadLLRPMessage(r io.Reader) (*LLRPHeader, []byte, error) {
	headerBuf := make([]byte, 10)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return nil, nil, err
	}

	firstWord := binary.BigEndian.Uint16(headerBuf[0:2])
	version := uint8(firstWord >> 10)
	msgType := firstWord & 0x03FF
	length := binary.BigEndian.Uint32(headerBuf[2:6])
	msgID := binary.BigEndian.Uint32(headerBuf[6:10])

	if length < 10 {
		return nil, nil, fmt.Errorf("invalid LLRP frame length %d < 10", length)
	}

	payloadLen := int(length) - 10
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, nil, err
		}
	}

	hdr := &LLRPHeader{
		Version: version,
		Type:    msgType,
		Length:  length,
		ID:      msgID,
	}
	return hdr, payload, nil
}

// ParseROAccessReport extracts all TagReportData parameters from an LLRP RO_ACCESS_REPORT payload
func ParseROAccessReport(payload []byte) ([]TagReportItem, error) {
	var items []TagReportItem
	offset := 0

	for offset < len(payload) {
		if offset+4 > len(payload) {
			break
		}
		paramTypeAndReserved := binary.BigEndian.Uint16(payload[offset : offset+2])
		isTV := (paramTypeAndReserved & 0x8000) != 0

		if isTV {
			// TV Parameter: skip 1 byte type + fixed length
			offset += 2
			continue
		}

		paramType := paramTypeAndReserved & 0x03FF
		paramLen := int(binary.BigEndian.Uint16(payload[offset+2 : offset+4]))
		if paramLen < 4 || offset+paramLen > len(payload) {
			break
		}

		if paramType == ParamTagReportData {
			item := parseTagReportData(payload[offset+4 : offset+paramLen])
			if item.EPCHex != "" {
				items = append(items, item)
			}
		}

		offset += paramLen
	}

	return items, nil
}

func parseTagReportData(data []byte) TagReportItem {
	item := TagReportItem{
		AntennaID:     1,
		PeakRSSI:      -60.0,
		ReadCount:     1,
		FirstSeenTime: time.Now().UTC(),
		LastSeenTime:  time.Now().UTC(),
	}

	offset := 0
	for offset < len(data) {
		firstByte := data[offset]
		isTV := (firstByte & 0x80) != 0

		if isTV {
			tvType := uint16(firstByte & 0x7F)
			switch tvType {
			case ParamEPC96:
				if offset+13 <= len(data) {
					item.EPCHex = hex.EncodeToString(data[offset+1 : offset+13])
					offset += 13
					continue
				}
			case ParamAntennaID:
				if offset+3 <= len(data) {
					item.AntennaID = int(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
					offset += 3
					continue
				}
			case ParamPeakRSSI:
				if offset+2 <= len(data) {
					rawRSSI := int8(data[offset+1])
					item.PeakRSSI = float64(rawRSSI)
					offset += 2
					continue
				}
			case ParamChannelIndex:
				if offset+3 <= len(data) {
					item.ChannelIndex = int(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
					offset += 3
					continue
				}
			case ParamTagSeenCount:
				if offset+3 <= len(data) {
					item.ReadCount = int(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
					offset += 3
					continue
				}
			case ParamFirstSeenTimestampUTC:
				if offset+9 <= len(data) {
					micros := binary.BigEndian.Uint64(data[offset+1 : offset+9])
					item.FirstSeenTime = time.UnixMicro(int64(micros)).UTC()
					offset += 9
					continue
				}
			case ParamLastSeenTimestampUTC:
				if offset+9 <= len(data) {
					micros := binary.BigEndian.Uint64(data[offset+1 : offset+9])
					item.LastSeenTime = time.UnixMicro(int64(micros)).UTC()
					offset += 9
					continue
				}
			}
			offset += 2 // generic TV fallback
		} else {
			if offset+4 > len(data) {
				break
			}
			paramType := binary.BigEndian.Uint16(data[offset:offset+2]) & 0x03FF
			paramLen := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
			if paramLen < 4 || offset+paramLen > len(data) {
				break
			}

			if paramType == ParamEPCData && paramLen > 4 {
				// TLV EPC Data (contains EPC length and EPC bits)
				epcBytes := data[offset+4 : offset+paramLen]
				if len(epcBytes) >= 14 {
					item.EPCHex = hex.EncodeToString(epcBytes[2:14])
				}
			}

			offset += paramLen
		}
	}

	return item
}

// SendLLRPAck writes a KEEPALIVE_ACK response back to the reader
func SendLLRPAck(conn net.Conn, msgID uint32) error {
	frame := EncodeLLRPMessage(MsgKeepaliveAck, msgID, nil)
	_, err := conn.Write(frame)
	return err
}

// BuildStandardROSpecPayload constructs an LLRP ADD_ROSPEC payload for a 4-antenna portal
func BuildStandardROSpecPayload(roSpecID uint32) []byte {
	// Standard compact ROSpec parameter binary buffer
	buf := make([]byte, 16)
	binary.BigEndian.PutUint16(buf[0:2], 177) // Parameter Type 177 (ROSpec)
	binary.BigEndian.PutUint16(buf[2:4], 16)  // Parameter Length
	binary.BigEndian.PutUint32(buf[4:8], roSpecID)
	buf[8] = 0 // Priority
	buf[9] = 1 // CurrentState (Disabled)
	return buf
}
