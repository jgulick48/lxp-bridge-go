package modbus

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sigurn/crc16"
	log "github.com/sirupsen/logrus"
)

type Packet struct {
	// Define the structure of Packet according to your needs
}

func Decode(src []byte, messageSendFunc func(register registers.Register, value float64)) (*Packet, error) {
	srcLen := len(src)

	if srcLen < 6 {
		// not enough data to read packet length
		return nil, nil
	}
	if srcLen <= 8 {
		return nil, errors.New("Not enough data to read packets")
	}

	if !bytes.Equal(src[:2], []byte{161, 26}) {
		return nil, errors.New("161, 26 header not found")
	}

	// protocol is in src[2..4], not used here yet
	log.Infof("Protocol version: %v", binary.LittleEndian.Uint16(src[2:4]))
	packetLen := binary.LittleEndian.Uint16(src[4:6])
	log.Infof("Packet length: %v", packetLen)

	// packetLen excludes the first 6 bytes, re-add those to make maths easier
	frameLen := 6 + int(packetLen)
	log.Infof("TCP Function: %s", src[7:8])
	log.Infof("Dataloger Serial Number: %s", src[8:18])

	data := src[18:]

	log.Infof("total length %v %d bytes in: %v\n", frameLen, len(data), data)

	packet, err := ParsePacket(data, src[7], messageSendFunc) // Implement ParsePacket according to your needs
	if err != nil {
		return nil, err
	}

	return packet, nil
}

func BuildPacket(dataLogger, serial string, mode, function byte, register, value uint16) []byte {
	packetStart := []byte{0xa1, 0x1a, 1, 0}
	packet := []byte{1, 0xc2}
	packet = append(packet, []byte(dataLogger)...)
	data := []byte{mode, function}
	data = append(data, []byte(serial)...)
	data = append(data, getBytesForUInt16(register)...)
	data = append(data, getBytesForUInt16(value)...)
	table := crc16.MakeTable(crc16.CRC16_MODBUS)
	data = append(data, getBytesForUInt16(crc16.Checksum(data, table))...)
	data = append(getBytesForUInt16(uint16(len(data))), data...)
	data = append(packet, data...)
	packetStart = append(packetStart, getBytesForUInt16(uint16(len(data)))...)
	data = append(packetStart, data...)
	return data
}

func getBytesForUInt16(value uint16) []byte {
	result := make([]byte, 2)
	binary.LittleEndian.PutUint16(result, value)
	return result
}

func ParsePacket(data []byte, function byte, messageSendFunc func(register registers.Register, value float64)) (*Packet, error) {
	switch function {
	case 0xc1:
		log.Info("Got Heartbeat function")
	case 0xc2:
		log.Info("Got Translated Data")
		log.Infof("Address: %v", data[2])
		log.Infof("Function: %v", data[3])
		log.Infof("Inverter Serial Number: %s", data[4:14])
		dataLength := int(binary.LittleEndian.Uint16(data[14:16]))
		log.Infof("Data Length: %v", dataLength)
		if data[3] == 4 {
			switch dataLength {
			case 0:
				ReadInput1(data[17:], messageSendFunc)
			case 40:
				ReadInput2(data[17:], messageSendFunc)
			case 80:
				ReadInput3(data[17:], messageSendFunc)
			case 120:
				ReadInput4(data[17:], messageSendFunc)
			}
		}
		if data[3] == 3 {
			switch dataLength {
			case 40:
				ReadHold1(data[17:])
			case 80:
				ReadHold2(data[17:])
			case 120:
				ReadHold3(data[17:])
			case 200:
				ReadHold4(data[17:])
			}
		}
	}
	return &Packet{}, nil
}

func ReadHold1(data []byte) {

}
func ReadHold2(data []byte) {

}
func ReadHold3(data []byte) {

}
func ReadHold4(data []byte) {
	log.Infof("State: %v", binary.LittleEndian.Uint16(data[0:2]))
}
func ReadInput1(data []byte, messageSendFunc func(register registers.Register, value float64)) {
	if len(data) < 80 {
		log.Infof("Not enough data got %v bytes but was expecting 80", len(data))
		return
	}
	messageSendFunc(registers.InputRegisters[registers.State], float64(binary.LittleEndian.Uint16(data[0:2])))
	messageSendFunc(registers.InputRegisters[registers.V_PV_1], float64(binary.LittleEndian.Uint16(data[2:4])))
	messageSendFunc(registers.InputRegisters[registers.V_PV_2], float64(binary.LittleEndian.Uint16(data[4:6])))
	messageSendFunc(registers.InputRegisters[registers.V_PV_3], float64(binary.LittleEndian.Uint16(data[6:8])))
	messageSendFunc(registers.InputRegisters[registers.V_BAT], float64(binary.LittleEndian.Uint16(data[8:10])))
	messageSendFunc(registers.InputRegisters[registers.SOC], float64(data[10:11][0]))
	messageSendFunc(registers.InputRegisters[registers.SOH], float64(data[11:12][0]))
	messageSendFunc(registers.InputRegisters[registers.Internal_Fault], float64(binary.LittleEndian.Uint16(data[12:14])))
	messageSendFunc(registers.InputRegisters[registers.P_PV], float64(binary.LittleEndian.Uint16(data[14:16])+binary.LittleEndian.Uint16(data[16:18])+binary.LittleEndian.Uint16(data[18:20])))
	messageSendFunc(registers.InputRegisters[registers.P_PV_1], float64(binary.LittleEndian.Uint16(data[14:16])))
	messageSendFunc(registers.InputRegisters[registers.P_PV_2], float64(binary.LittleEndian.Uint16(data[16:18])))
	messageSendFunc(registers.InputRegisters[registers.P_PV_3], float64(binary.LittleEndian.Uint16(data[18:20])))
	messageSendFunc(registers.InputRegisters[registers.P_Battery], float64(binary.LittleEndian.Uint16(data[20:22]))-float64(binary.LittleEndian.Uint16(data[22:24])))
	messageSendFunc(registers.InputRegisters[registers.P_Charge], float64(binary.LittleEndian.Uint16(data[20:22])))
	messageSendFunc(registers.InputRegisters[registers.P_Discharge], float64(binary.LittleEndian.Uint16(data[22:24])))
	messageSendFunc(registers.InputRegisters[registers.V_AC_R], float64(binary.LittleEndian.Uint16(data[24:26])))
	messageSendFunc(registers.InputRegisters[registers.V_AC_S], float64(binary.LittleEndian.Uint16(data[26:28])))
	messageSendFunc(registers.InputRegisters[registers.V_AC_T], float64(binary.LittleEndian.Uint16(data[28:30])))
	messageSendFunc(registers.InputRegisters[registers.F_AC], float64(binary.LittleEndian.Uint16(data[30:32])))
	messageSendFunc(registers.InputRegisters[registers.P_INV], float64(binary.LittleEndian.Uint16(data[32:34])))
	messageSendFunc(registers.InputRegisters[registers.P_REC], float64(binary.LittleEndian.Uint16(data[34:36])))
	messageSendFunc(registers.InputRegisters[registers.INV_RMS], float64(binary.LittleEndian.Uint16(data[36:38])))
	messageSendFunc(registers.InputRegisters[registers.PF], float64(binary.LittleEndian.Uint16(data[38:40])))
	messageSendFunc(registers.InputRegisters[registers.V_EPS_R], float64(binary.LittleEndian.Uint16(data[40:42])))
	messageSendFunc(registers.InputRegisters[registers.V_EPS_S], float64(binary.LittleEndian.Uint16(data[42:44])))
	messageSendFunc(registers.InputRegisters[registers.V_EPS_T], float64(binary.LittleEndian.Uint16(data[44:46])))
	messageSendFunc(registers.InputRegisters[registers.F_EPS], float64(binary.LittleEndian.Uint16(data[46:48])))
	messageSendFunc(registers.InputRegisters[registers.P_EPS], float64(binary.LittleEndian.Uint16(data[48:50])))
	messageSendFunc(registers.InputRegisters[registers.S_EPS], float64(binary.LittleEndian.Uint16(data[50:52])))
	messageSendFunc(registers.InputRegisters[registers.P_TO_GRID], float64(binary.LittleEndian.Uint16(data[52:54])))
	messageSendFunc(registers.InputRegisters[registers.P_TO_USER], float64(binary.LittleEndian.Uint16(data[54:56])))
	messageSendFunc(registers.InputRegisters[registers.E_PV_DAY], float64(binary.LittleEndian.Uint16(data[56:58])+binary.LittleEndian.Uint16(data[58:60])+binary.LittleEndian.Uint16(data[60:62])))
	messageSendFunc(registers.InputRegisters[registers.E_PV_DAY_1], float64(binary.LittleEndian.Uint16(data[56:58])))
	messageSendFunc(registers.InputRegisters[registers.E_PV_DAY_2], float64(binary.LittleEndian.Uint16(data[58:60])))
	messageSendFunc(registers.InputRegisters[registers.E_PV_DAY_3], float64(binary.LittleEndian.Uint16(data[60:62])))
	messageSendFunc(registers.InputRegisters[registers.E_INV_DAY], float64(binary.LittleEndian.Uint16(data[62:64])))
	messageSendFunc(registers.InputRegisters[registers.E_REC_DAY], float64(binary.LittleEndian.Uint16(data[64:66])))
	messageSendFunc(registers.InputRegisters[registers.E_CHG_DAY], float64(binary.LittleEndian.Uint16(data[66:68])))
	messageSendFunc(registers.InputRegisters[registers.E_DISCHG_DAY], float64(binary.LittleEndian.Uint16(data[68:70])))
	messageSendFunc(registers.InputRegisters[registers.E_EPS_DAY], float64(binary.LittleEndian.Uint16(data[70:72])))
	messageSendFunc(registers.InputRegisters[registers.E_TO_GRID_DAY], float64(binary.LittleEndian.Uint16(data[72:74])))
	messageSendFunc(registers.InputRegisters[registers.E_TO_USER_DAY], float64(binary.LittleEndian.Uint16(data[74:76])))
	messageSendFunc(registers.InputRegisters[registers.V_BUS_1], float64(binary.LittleEndian.Uint16(data[76:78])))
	messageSendFunc(registers.InputRegisters[registers.V_BUS_2], float64(binary.LittleEndian.Uint16(data[78:80])))

}
func ReadInput2(data []byte, messageSendFunc func(register registers.Register, value float64)) {
	if len(data) < 40 {
		log.Infof("Not enough data got %v bytes but was expecting 40", len(data))
		return
	}
	messageSendFunc(registers.InputRegisters[registers.E_PV_ALL], float64(binary.LittleEndian.Uint32(data[0:4])+binary.LittleEndian.Uint32(data[4:8])+binary.LittleEndian.Uint32(data[8:12])))
	messageSendFunc(registers.InputRegisters[registers.E_PV_1_ALL], float64(binary.LittleEndian.Uint32(data[0:4])))
	messageSendFunc(registers.InputRegisters[registers.E_PV_2_ALL], float64(binary.LittleEndian.Uint32(data[4:8])))
	messageSendFunc(registers.InputRegisters[registers.E_PV_3_ALL], float64(binary.LittleEndian.Uint32(data[8:12])))
	messageSendFunc(registers.InputRegisters[registers.E_INV_ALL], float64(binary.LittleEndian.Uint32(data[12:16])))
	messageSendFunc(registers.InputRegisters[registers.E_REC_ALL], float64(binary.LittleEndian.Uint32(data[16:20])))
	messageSendFunc(registers.InputRegisters[registers.E_CHG_ALL], float64(binary.LittleEndian.Uint32(data[20:24])))
	messageSendFunc(registers.InputRegisters[registers.E_DISCHG_ALL], float64(binary.LittleEndian.Uint32(data[24:28])))
	messageSendFunc(registers.InputRegisters[registers.E_EPS_ALL], float64(binary.LittleEndian.Uint32(data[28:32])))
	messageSendFunc(registers.InputRegisters[registers.E_TO_GRID_ALL], float64(binary.LittleEndian.Uint32(data[32:36])))
	messageSendFunc(registers.InputRegisters[registers.E_TO_USER_ALL], float64(binary.LittleEndian.Uint32(data[36:40])))
	messageSendFunc(registers.InputRegisters[registers.Fault_Code], float64(binary.LittleEndian.Uint32(data[40:44])))
	messageSendFunc(registers.InputRegisters[registers.Warning_Code], float64(binary.LittleEndian.Uint32(data[44:48])))
	messageSendFunc(registers.InputRegisters[registers.T_INNER], float64(binary.LittleEndian.Uint16(data[44:46])))
	messageSendFunc(registers.InputRegisters[registers.T_RAD_1], float64(binary.LittleEndian.Uint16(data[46:48])))
	messageSendFunc(registers.InputRegisters[registers.T_RAD_2], float64(binary.LittleEndian.Uint16(data[48:50])))
	messageSendFunc(registers.InputRegisters[registers.T_BAT], float64(binary.LittleEndian.Uint16(data[50:52])))
	messageSendFunc(registers.InputRegisters[registers.Runtime], float64(binary.LittleEndian.Uint32(data[58:62])))
	messageSendFunc(registers.InputRegisters[registers.AC_INPUT_TYPE], float64(binary.LittleEndian.Uint16(data[74:76])))
}
func ReadInput3(data []byte, messageSendFunc func(register registers.Register, value float64)) {
	log.Infof("MaxChgCurr: %v", binary.LittleEndian.Uint16(data[2:4]))
	log.Infof("MaxDischgCurr: %v", binary.LittleEndian.Uint16(data[4:6]))
	log.Infof("ChargeVoltRef: %v", binary.LittleEndian.Uint16(data[6:8]))
	log.Infof("DischgCutVolt: %v", binary.LittleEndian.Uint16(data[8:10]))
}
func ReadInput4(data []byte, messageSendFunc func(register registers.Register, value float64)) {
	log.Infof("VBusP: %v", binary.LittleEndian.Uint16(data[0:2]))
	log.Infof("GenVolt: %v", binary.LittleEndian.Uint16(data[2:4]))
	log.Infof("GenFreq: %v", binary.LittleEndian.Uint16(data[4:6]))
	log.Infof("GenPower: %v", binary.LittleEndian.Uint16(data[6:8]))
}
