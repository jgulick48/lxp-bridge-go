package modbus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sigurn/crc16"
	log "github.com/sirupsen/logrus"
	"strings"
)

// Packet describes the outcome of decoding a single frame.
type Packet struct {
	// CRCMismatch is set when a translated data frame failed CRC verification
	// but was still decoded (CRCModeLog).
	CRCMismatch bool
}

// CRCMode controls what happens when a translated data frame fails its CRC check.
type CRCMode int

const (
	// CRCModeLog counts and debug logs mismatches but still decodes the frame.
	CRCModeLog CRCMode = iota
	// CRCModeDrop discards frames that fail the CRC check.
	CRCModeDrop
)

// ParseCRCMode converts the crc_mode configuration value into a CRCMode.
func ParseCRCMode(value string) (CRCMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "log":
		return CRCModeLog, nil
	case "drop":
		return CRCModeDrop, nil
	}
	return CRCModeLog, fmt.Errorf("invalid crc_mode %q (expected log or drop)", value)
}

var (
	// ErrCRCMismatch is returned in CRCModeDrop when a frame fails CRC verification.
	ErrCRCMismatch = errors.New("crc mismatch")
	// ErrShortPacket is returned when a frame is too short to be decoded.
	ErrShortPacket = errors.New("packet too short")

	crcTable = crc16.MakeTable(crc16.CRC16_MODBUS)
)

const (
	frameHeaderLen = 6
	// translatedHeaderLen is the offset of the translated data section in a frame.
	translatedHeaderLen = 18
)

// Decode decodes a full frame (including the 6 byte header) using CRCModeLog.
// It never panics on malformed input.
func Decode(src []byte, callback ParserCallback) (*Packet, error) {
	return DecodeWithMode(src, callback, CRCModeLog)
}

// DecodeWithMode decodes a full frame using the given CRC mode. It never
// panics on malformed input; problems are returned as errors.
func DecodeWithMode(src []byte, callback ParserCallback, mode CRCMode) (packet *Packet, err error) {
	defer func() {
		if r := recover(); r != nil {
			packet, err = nil, fmt.Errorf("panic decoding packet: %v", r)
		}
	}()
	srcLen := len(src)
	if srcLen < frameHeaderLen {
		return nil, fmt.Errorf("%w: %d bytes, need header", ErrShortPacket, srcLen)
	}
	if !bytes.Equal(src[:2], []byte{161, 26}) {
		return nil, errors.New("161, 26 header not found")
	}
	packetLen := int(binary.LittleEndian.Uint16(src[4:6]))
	if srcLen < frameHeaderLen+packetLen {
		return nil, fmt.Errorf("%w: have %d bytes, header declares %d", ErrShortPacket, srcLen, frameHeaderLen+packetLen)
	}
	src = src[:frameHeaderLen+packetLen]
	if len(src) < 8 {
		return nil, fmt.Errorf("%w: no tcp function", ErrShortPacket)
	}
	function := src[7]
	if function != 0xc2 {
		// Heartbeats and other frames carry nothing we decode.
		var dataLogger string
		if len(src) >= translatedHeaderLen {
			dataLogger = string(src[8:translatedHeaderLen])
		}
		return ParsePacket(nil, function, callback, dataLogger)
	}
	if len(src) < translatedHeaderLen {
		return nil, fmt.Errorf("%w: translated frame of %d bytes", ErrShortPacket, len(src))
	}
	data := src[translatedHeaderLen:]
	log.Debugf("Got new Packet from Inverter with Translated Data and Length %v", packetLen)
	crcOK := verifyCRC(data)
	if !crcOK {
		if mode == CRCModeDrop {
			return nil, ErrCRCMismatch
		}
		log.Debugf("CRC mismatch on translated frame from %s (%d data bytes)", string(src[8:translatedHeaderLen]), len(data))
	}
	packet, err = ParsePacket(data, function, callback, string(src[8:translatedHeaderLen]))
	if err != nil {
		return nil, err
	}
	packet.CRCMismatch = !crcOK
	return packet, nil
}

// verifyCRC checks the CRC16/MODBUS of the translated data section:
// [len u16 LE][addr][func][serial 10B][register u16][...values][crc u16 LE].
// The CRC covers everything between the 2 byte length prefix and the CRC.
func verifyCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	declared := int(binary.LittleEndian.Uint16(data[0:2]))
	if declared < 3 || 2+declared > len(data) {
		return false
	}
	body := data[2 : 2+declared-2]
	want := binary.LittleEndian.Uint16(data[2+declared-2 : 2+declared])
	return crc16.Checksum(body, crcTable) == want
}

func BuildPacket(dataLogger, serial string, mode, function byte, register uint16, values ...byte) []byte {
	packetStart := []byte{0xa1, 0x1a, 1, 0}
	packet := []byte{1, 0xc2}
	packet = append(packet, []byte(dataLogger)...)
	data := []byte{mode, function}
	data = append(data, []byte(serial)...)
	data = append(data, GetBytesForUInt16(register)...)
	data = append(data, values...)
	data = append(data, GetBytesForUInt16(crc16.Checksum(data, crcTable))...)
	data = append(GetBytesForUInt16(uint16(len(data))), data...)
	data = append(packet, data...)
	packetStart = append(packetStart, GetBytesForUInt16(uint16(len(data)))...)
	data = append(packetStart, data...)
	return data
}

func GetBytesForUInt16(value uint16) []byte {
	result := make([]byte, 2)
	binary.LittleEndian.PutUint16(result, value)
	return result
}
func GetBytesForUInt32(value uint32) []byte {
	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, value)
	return result
}

// ParsePacket decodes the translated data section of a frame. It never panics
// on short input; too-short data is reported as an error.
func ParsePacket(data []byte, function byte, callback ParserCallback, dataLogger string) (packet *Packet, err error) {
	defer func() {
		if r := recover(); r != nil {
			packet, err = nil, fmt.Errorf("panic parsing packet: %v", r)
		}
	}()
	switch function {
	case 0xc1:
		log.Debug("Got Heartbeat function")
	case 0xc2:
		// data: [len u16][addr][func][serial 10B][register u16][byte count][values...]
		if len(data) < 16 {
			return nil, fmt.Errorf("%w: translated data of %d bytes", ErrShortPacket, len(data))
		}
		dataLength := int(binary.LittleEndian.Uint16(data[14:16]))
		log.Debugf("Attempting to parse packet with Function: %v and Data Length %v", data[3], dataLength)
		values := func(offset, need int) ([]byte, bool) {
			if len(data) < offset+need {
				log.Debugf("Short data: have %d bytes, need %d at offset %d", len(data), need, offset)
				return nil, false
			}
			return data[offset:], true
		}
		short := func(offset int) error {
			return fmt.Errorf("%w: %d bytes, need offset %d", ErrShortPacket, len(data), offset)
		}
		switch data[3] {
		case 4:
			switch dataLength {
			case 0:
				d, ok := values(17, 1)
				if !ok {
					return nil, short(17)
				}
				ReadInput1(d, callback, dataLogger)
				if len(data) > 130 {
					log.Debug("Parsing Inputs 1, 2, 3, and 4 Short")
					if d, ok = values(97, 1); ok {
						ReadInput2(d, callback, dataLogger)
					}
					if d, ok = values(177, 1); ok {
						ReadInput3(d, callback, dataLogger)
					}
					if d, ok = values(257, 1); ok {
						ReadInput4Short(d, callback, dataLogger)
					}
				}
			case 40:
				if d, ok := values(17, 1); ok {
					ReadInput2(d, callback, dataLogger)
				}
			case 80:
				if d, ok := values(17, 1); ok {
					ReadInput3(d, callback, dataLogger)
				}
			case 120:
				if d, ok := values(17, 1); ok {
					ReadInput4(d, callback, dataLogger)
				}
			case 127:
				if d, ok := values(17, 1); ok {
					ReadInput4Long(d, callback, dataLogger)
				}
				// A 127 register read from 127 also covers block 160.
				if d, ok := values(17+(160-127)*2, 1); ok {
					ReadInput5(d, callback, dataLogger)
				}
			case 160:
				if d, ok := values(17, 1); ok {
					ReadInput5(d, callback, dataLogger)
				}
			}
		case 3:
			count := int(data[16])
			if len(data) < 17+count {
				count = len(data) - 17
			}
			if count < 2 {
				return nil, short(19)
			}
			vals := make([]uint16, count/2)
			for i := range vals {
				vals[i] = binary.LittleEndian.Uint16(data[17+2*i:])
			}
			reportHolding(uint16(dataLength), vals, callback, dataLogger)
		case 6:
			// Echo of a single register write: [register][value].
			if _, ok := values(14, 4); !ok {
				return nil, short(18)
			}
			reportHolding(uint16(dataLength), []uint16{binary.LittleEndian.Uint16(data[16:18])}, callback, dataLogger)
		}
	}
	return &Packet{}, nil
}

// HoldingCallback is implemented by callbacks that want the raw holding
// register values, for example to change single bits of a register.
type HoldingCallback interface {
	ReportHolding(dataLogger string, start uint16, values []uint16)
}

// reportHolding publishes every known hold register in values, which start
// at register start.
func reportHolding(start uint16, values []uint16, callback ParserCallback, dataLogger string) {
	if hc, ok := callback.(HoldingCallback); ok {
		hc.ReportHolding(dataLogger, start, values)
	}
	for id, register := range registers.HoldRegisters {
		addr, ok := registers.HoldRegisterToIDMap[id]
		if !ok || addr < int(start) || addr >= int(start)+len(values) {
			continue
		}
		callback.ReportValue(register, int32(register.FieldValue(values[addr-int(start)])), dataLogger)
	}
}

func ReadInput1(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 80 {
		log.Debugf("Not enough data got %v bytes but was expecting 80", len(data))
		return
	}
	callback.ReportValue(registers.InputRegisters[registers.State], int32(binary.LittleEndian.Uint16(data[0:2])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_PV_1], int32(binary.LittleEndian.Uint16(data[2:4])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_PV_2], int32(binary.LittleEndian.Uint16(data[4:6])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_PV_3], int32(binary.LittleEndian.Uint16(data[6:8])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_BAT], int32(binary.LittleEndian.Uint16(data[8:10])), dataLogger)
	soc := int32(data[10:11][0])
	callback.ReportValue(registers.InputRegisters[registers.SOC], soc, dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.SOH], int32(data[11:12][0]), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.Internal_Fault], int32(binary.LittleEndian.Uint16(data[12:14])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_PV], int32(binary.LittleEndian.Uint16(data[14:16])+binary.LittleEndian.Uint16(data[16:18])+binary.LittleEndian.Uint16(data[18:20])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_PV_1], int32(binary.LittleEndian.Uint16(data[14:16])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_PV_2], int32(binary.LittleEndian.Uint16(data[16:18])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_PV_3], int32(binary.LittleEndian.Uint16(data[18:20])), dataLogger)
	batteryPower := int32(binary.LittleEndian.Uint16(data[20:22])) - int32(binary.LittleEndian.Uint16(data[22:24]))
	callback.ReportValue(registers.InputRegisters[registers.P_Battery], batteryPower, dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_Charge], int32(binary.LittleEndian.Uint16(data[20:22])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_Discharge], int32(binary.LittleEndian.Uint16(data[22:24])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_AC_R], int32(binary.LittleEndian.Uint16(data[24:26])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_AC_S], int32(binary.LittleEndian.Uint16(data[26:28])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_AC_T], int32(binary.LittleEndian.Uint16(data[28:30])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.F_AC], int32(binary.LittleEndian.Uint16(data[30:32])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_INV], int32(binary.LittleEndian.Uint16(data[32:34])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_REC], int32(binary.LittleEndian.Uint16(data[34:36])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.INV_RMS], int32(binary.LittleEndian.Uint16(data[36:38])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.PF], int32(binary.LittleEndian.Uint16(data[38:40])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_EPS_R], int32(binary.LittleEndian.Uint16(data[40:42])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_EPS_S], int32(binary.LittleEndian.Uint16(data[42:44])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_EPS_T], int32(binary.LittleEndian.Uint16(data[44:46])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.F_EPS], int32(binary.LittleEndian.Uint16(data[46:48])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_EPS], int32(binary.LittleEndian.Uint16(data[48:50])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.S_EPS], int32(binary.LittleEndian.Uint16(data[50:52])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_TO_GRID], int32(binary.LittleEndian.Uint16(data[52:54])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_TO_USER], int32(binary.LittleEndian.Uint16(data[54:56])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_PV_DAY], int32(binary.LittleEndian.Uint16(data[56:58])+binary.LittleEndian.Uint16(data[58:60])+binary.LittleEndian.Uint16(data[60:62])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_PV_DAY_1], int32(binary.LittleEndian.Uint16(data[56:58])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_PV_DAY_2], int32(binary.LittleEndian.Uint16(data[58:60])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_PV_DAY_3], int32(binary.LittleEndian.Uint16(data[60:62])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_INV_DAY], int32(binary.LittleEndian.Uint16(data[62:64])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_REC_DAY], int32(binary.LittleEndian.Uint16(data[64:66])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_CHG_DAY], int32(binary.LittleEndian.Uint16(data[66:68])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_DISCHG_DAY], int32(binary.LittleEndian.Uint16(data[68:70])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_DAY], int32(binary.LittleEndian.Uint16(data[70:72])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_TO_GRID_DAY], int32(binary.LittleEndian.Uint16(data[72:74])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_TO_USER_DAY], int32(binary.LittleEndian.Uint16(data[74:76])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_BUS_1], int32(binary.LittleEndian.Uint16(data[76:78])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_BUS_2], int32(binary.LittleEndian.Uint16(data[78:80])), dataLogger)
	callback.ReportPowerOpenEVSE(soc, batteryPower)

}
func ReadInput2(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 80 {
		log.Debugf("Not enough data got %v bytes but was expecting 80", len(data))
		return
	}
	epvall := int32(binary.LittleEndian.Uint32(data[0:4]) + binary.LittleEndian.Uint32(data[4:8]) + binary.LittleEndian.Uint32(data[8:12]))
	callback.ReportValue(registers.InputRegisters[registers.E_PV_ALL], epvall, dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_PV_1_ALL], int32(binary.LittleEndian.Uint32(data[0:4])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_PV_2_ALL], int32(binary.LittleEndian.Uint32(data[4:8])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_PV_3_ALL], int32(binary.LittleEndian.Uint32(data[8:12])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_INV_ALL], int32(binary.LittleEndian.Uint32(data[12:16])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_REC_ALL], int32(binary.LittleEndian.Uint32(data[16:20])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_CHG_ALL], int32(binary.LittleEndian.Uint32(data[20:24])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_DISCHG_ALL], int32(binary.LittleEndian.Uint32(data[24:28])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_ALL], int32(binary.LittleEndian.Uint32(data[28:32])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_TO_GRID_ALL], int32(binary.LittleEndian.Uint32(data[32:36])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_TO_USER_ALL], int32(binary.LittleEndian.Uint32(data[36:40])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.Fault_Code], int32(binary.LittleEndian.Uint32(data[40:44])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.Warning_Code], int32(binary.LittleEndian.Uint32(data[44:48])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T_INNER], int32(binary.LittleEndian.Uint16(data[48:50])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T_RAD_1], int32(binary.LittleEndian.Uint16(data[50:52])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T_RAD_2], int32(binary.LittleEndian.Uint16(data[52:54])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T_BAT], int32(binary.LittleEndian.Uint16(data[54:56])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.Runtime], int32(binary.LittleEndian.Uint32(data[58:62])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.I_PV_1], int32(binary.LittleEndian.Uint16(data[64:66])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.I_PV_2], int32(binary.LittleEndian.Uint16(data[66:68])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.I_PV_3], int32(binary.LittleEndian.Uint16(data[68:70])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AC_INPUT_TYPE], int32(binary.LittleEndian.Uint16(data[74:76])), dataLogger)
}
func ReadInput3(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 80 {
		log.Debugf("Not enough data got %v bytes but was expecting 80", len(data))
		return
	}
	callback.ReportValue(registers.InputRegisters[registers.MAX_CHG_CURR], int32(binary.LittleEndian.Uint16(data[2:4])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MAX_DISCHG_CURR], int32(binary.LittleEndian.Uint16(data[4:6])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.CHRG_VOLT_REF], int32(binary.LittleEndian.Uint16(data[6:8])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.DISCHRG_CUT_VOLT], int32(binary.LittleEndian.Uint16(data[8:10])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_0_BMS], int32(binary.LittleEndian.Uint16(data[10:12])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_1_BMS], int32(binary.LittleEndian.Uint16(data[12:14])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_2_BMS], int32(binary.LittleEndian.Uint16(data[14:16])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_3_BMS], int32(binary.LittleEndian.Uint16(data[16:18])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_4_BMS], int32(binary.LittleEndian.Uint16(data[18:20])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_5_BMS], int32(binary.LittleEndian.Uint16(data[20:22])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_6_BMS], int32(binary.LittleEndian.Uint16(data[22:24])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_7_BMS], int32(binary.LittleEndian.Uint16(data[24:26])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_8_BMS], int32(binary.LittleEndian.Uint16(data[26:28])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_9_BMS], int32(binary.LittleEndian.Uint16(data[28:30])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_STATUS_INV], int32(binary.LittleEndian.Uint16(data[30:32])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_PARRALlEL_NUM], int32(binary.LittleEndian.Uint16(data[32:34])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_CAPACITY], int32(binary.LittleEndian.Uint16(data[34:36])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BATT_CURRENT_BMS], uint16ToInt32(binary.LittleEndian.Uint16(data[36:38])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.Fault_Code_BMS], int32(binary.LittleEndian.Uint16(data[38:40])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.Warning_Code_BMS], int32(binary.LittleEndian.Uint16(data[40:42])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MAX_CELL_VOLT], int32(binary.LittleEndian.Uint16(data[42:44])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MIN_CELL_VOLT], int32(binary.LittleEndian.Uint16(data[44:46])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MAX_CELL_TEMP], int32(binary.LittleEndian.Uint16(data[46:48])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MIN_CELL_TEMP], int32(binary.LittleEndian.Uint16(data[48:50])), dataLogger)
	// Register 105: bits 0-2 BMS firmware update state, bit 3 generator dry contact.
	fwState := binary.LittleEndian.Uint16(data[50:52])
	callback.ReportValue(registers.InputRegisters[registers.BMS_FW_UPDATE_STATE], int32(fwState&0x7), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.GEN_DRY_CONTACT], int32(fwState>>3&1), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.CYCLE_CNT_BMS], int32(binary.LittleEndian.Uint16(data[52:54])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BAT_VOLT_SMPL_INV], int32(binary.LittleEndian.Uint16(data[54:56])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T1], int32(binary.LittleEndian.Uint16(data[56:58])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T2], int32(binary.LittleEndian.Uint16(data[58:60])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T3], int32(binary.LittleEndian.Uint16(data[60:62])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T4], int32(binary.LittleEndian.Uint16(data[62:64])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T5], int32(binary.LittleEndian.Uint16(data[64:66])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.CONSUMPTION_POWER], int32(binary.LittleEndian.Uint16(data[68:70])), dataLogger)
}

// ReadInput5 decodes input registers starting at 160: load power and energy
// (170-173) and, when the read reaches register 204, the per-leg values
// (193-204).
func ReadInput5(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 28 { // through register 173
		log.Debugf("Not enough data got %v bytes but was expecting 28", len(data))
		return
	}
	callback.ReportValue(registers.InputRegisters[registers.P_LOAD], uint16ToInt32(binary.LittleEndian.Uint16(data[20:22])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_LOAD_DAY], int32(binary.LittleEndian.Uint16(data[22:24])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_LOAD_ALL], int32(binary.LittleEndian.Uint32(data[24:28])), dataLogger)
	if len(data) >= (204-160+1)*2 {
		callback.ReportValue(registers.InputRegisters[registers.V_GRID_L1], int32(binary.LittleEndian.Uint16(data[66:68])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.V_GRID_L2], int32(binary.LittleEndian.Uint16(data[68:70])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.V_GEN_L1], int32(binary.LittleEndian.Uint16(data[70:72])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.V_GEN_L2], int32(binary.LittleEndian.Uint16(data[72:74])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.P_INV_L1], int32(binary.LittleEndian.Uint16(data[74:76])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.P_INV_L2], int32(binary.LittleEndian.Uint16(data[76:78])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.P_REC_L1], int32(binary.LittleEndian.Uint16(data[78:80])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.P_REC_L2], int32(binary.LittleEndian.Uint16(data[80:82])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.P_TO_GRID_L1], int32(binary.LittleEndian.Uint16(data[82:84])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.P_TO_GRID_L2], int32(binary.LittleEndian.Uint16(data[84:86])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.P_TO_USER_L1], int32(binary.LittleEndian.Uint16(data[86:88])), dataLogger)
		callback.ReportValue(registers.InputRegisters[registers.P_TO_USER_L2], int32(binary.LittleEndian.Uint16(data[88:90])), dataLogger)
	}
}
func ReadInput4Short(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 14 {
		log.Debugf("Not enough data got %v bytes but was expecting 14", len(data))
		return
	}
	callback.ReportValue(registers.InputRegisters[registers.V_BUS_P], int32(binary.LittleEndian.Uint16(data[0:2])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_GEN], int32(binary.LittleEndian.Uint16(data[2:4])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.F_GEN], int32(binary.LittleEndian.Uint16(data[4:6])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_GEN], int32(binary.LittleEndian.Uint16(data[6:8])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_GEN_DAY], int32(binary.LittleEndian.Uint16(data[8:10])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_GEN_ALL], int32(binary.LittleEndian.Uint32(data[10:14])), dataLogger)
}
func ReadInput4Long(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 52 {
		log.Debugf("Not enough data got %v bytes but was expecting 52", len(data))
		return
	}
	data = append(make([]byte, 14), data...)
	callback.ReportValue(registers.InputRegisters[registers.V_EPS_L1], int32(binary.LittleEndian.Uint16(data[14:16])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_EPS_L2], int32(binary.LittleEndian.Uint16(data[16:18])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_EPS_L1], int32(binary.LittleEndian.Uint16(data[18:20])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_EPS_L2], int32(binary.LittleEndian.Uint16(data[20:22])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.S_EPS_L1], int32(binary.LittleEndian.Uint16(data[22:24])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.S_EPS_L2], int32(binary.LittleEndian.Uint16(data[24:26])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_DAY_L1], int32(binary.LittleEndian.Uint16(data[26:28])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_DAY_L2], int32(binary.LittleEndian.Uint16(data[28:30])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_ALL_L1], int32(binary.LittleEndian.Uint32(data[30:34])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_ALL_L2], int32(binary.LittleEndian.Uint32(data[34:38])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_CURR_CH1], int32(binary.LittleEndian.Uint16(data[40:42])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_CURR_CH2], int32(binary.LittleEndian.Uint16(data[42:44])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_CURR_CH3], int32(binary.LittleEndian.Uint16(data[44:46])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_CURR_CH4], int32(binary.LittleEndian.Uint16(data[46:48])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_ARC_CH1], int32(binary.LittleEndian.Uint16(data[50:52])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_ARC_CH2], int32(binary.LittleEndian.Uint16(data[52:54])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_ARC_CH3], int32(binary.LittleEndian.Uint16(data[54:56])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_ARC_CH4], int32(binary.LittleEndian.Uint16(data[56:58])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_MAX_ARC_CH1], int32(binary.LittleEndian.Uint16(data[58:60])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_MAX_ARC_CH2], int32(binary.LittleEndian.Uint16(data[60:62])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_MAX_ARC_CH3], int32(binary.LittleEndian.Uint16(data[62:64])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_MAX_ARC_CH4], int32(binary.LittleEndian.Uint16(data[64:66])), dataLogger)
}
func ReadInput4(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 80 {
		log.Debugf("Not enough data got %v bytes but was expecting 80", len(data))
		return
	}
	callback.ReportValue(registers.InputRegisters[registers.V_BUS_P], int32(binary.LittleEndian.Uint16(data[0:2])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_GEN], int32(binary.LittleEndian.Uint16(data[2:4])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.F_GEN], int32(binary.LittleEndian.Uint16(data[4:6])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_GEN], int32(binary.LittleEndian.Uint16(data[6:8])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_GEN_DAY], int32(binary.LittleEndian.Uint16(data[8:10])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_GEN_ALL], int32(binary.LittleEndian.Uint32(data[10:14])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_EPS_L1], int32(binary.LittleEndian.Uint16(data[14:16])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.V_EPS_L2], int32(binary.LittleEndian.Uint16(data[16:18])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_EPS_L1], int32(binary.LittleEndian.Uint16(data[18:20])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.P_EPS_L2], int32(binary.LittleEndian.Uint16(data[20:22])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.S_EPS_L1], int32(binary.LittleEndian.Uint16(data[22:24])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.S_EPS_L2], int32(binary.LittleEndian.Uint16(data[24:26])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_DAY_L1], int32(binary.LittleEndian.Uint16(data[26:28])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_DAY_L2], int32(binary.LittleEndian.Uint16(data[28:30])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_ALL_L1], int32(binary.LittleEndian.Uint32(data[30:34])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.E_EPS_ALL_L2], int32(binary.LittleEndian.Uint32(data[34:38])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_CURR_CH1], int32(binary.LittleEndian.Uint16(data[40:42])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_CURR_CH2], int32(binary.LittleEndian.Uint16(data[42:44])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_CURR_CH3], int32(binary.LittleEndian.Uint16(data[44:46])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_CURR_CH4], int32(binary.LittleEndian.Uint16(data[46:48])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_ARC_CH1], int32(binary.LittleEndian.Uint16(data[50:52])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_ARC_CH2], int32(binary.LittleEndian.Uint16(data[52:54])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_ARC_CH3], int32(binary.LittleEndian.Uint16(data[54:56])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_ARC_CH4], int32(binary.LittleEndian.Uint16(data[56:58])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_MAX_ARC_CH1], int32(binary.LittleEndian.Uint16(data[58:60])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_MAX_ARC_CH2], int32(binary.LittleEndian.Uint16(data[60:62])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_MAX_ARC_CH3], int32(binary.LittleEndian.Uint16(data[62:64])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.AFCI_MAX_ARC_CH4], int32(binary.LittleEndian.Uint16(data[64:66])), dataLogger)
}

func uint16ToInt32(value uint16) int32 {
	return int32(int16(value))
}
