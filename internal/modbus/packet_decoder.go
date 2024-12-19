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

func Decode(src []byte, callback ParserCallback) (*Packet, error) {
	srcLen := len(src)

	if srcLen < 6 {
		// not enough data to read packet length
		return nil, nil
	}
	if srcLen <= 8 {
		return nil, errors.New("Not enough data to read packets")
	}
	log.Infof("%v", src)

	if !bytes.Equal(src[:2], []byte{161, 26}) {
		return nil, errors.New("161, 26 header not found")
	}
	// protocol is in src[2..4], not used here yet
	//log.Infof("Protocol version: %v", binary.LittleEndian.Uint16(src[2:4]))
	packetLen := binary.LittleEndian.Uint16(src[4:6])
	log.Infof("Packet length: %v", packetLen)

	// packetLen excludes the first 6 bytes, re-add those to make maths easier
	//frameLen := 6 + int(packetLen)
	//log.Infof("TCP Function: %s", src[7:8])

	data := src[18:]
	log.Infof("Data Lenght: %v", len(data))

	//log.Infof("total length %v %d bytes in: %v\n", frameLen, len(data), data)

	packet, err := ParsePacket(data, src[7], callback, string(src[8:18])) // Implement ParsePacket according to your needs
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

func ParsePacket(data []byte, function byte, callback ParserCallback, dataLogger string) (*Packet, error) {
	switch function {
	case 0xc1:
		log.Info("Got Heartbeat function")
	case 0xc2:
		//log.Info("Got Translated Data")
		//log.Infof("Address: %v", data[2])
		//log.Infof("Function: %v", data[3])
		dataLength := int(binary.LittleEndian.Uint16(data[14:16]))
		log.Infof("Data Length: %v", dataLength)
		//log.Infof("%v", data[17:])
		if data[3] == 4 {
			switch dataLength {
			case 0:
				ReadInput1(data[17:], callback, dataLogger)
				if len(data) > 130 {
					ReadInput2(data[97:], callback, dataLogger)
					ReadInput3(data[177:], callback, dataLogger)
					ReadInput4Short(data[257:], callback, dataLogger)
				}
			case 40:
				ReadInput2(data[17:], callback, dataLogger)
			case 80:
				ReadInput3(data[17:], callback, dataLogger)
			case 120:
				ReadInput4(data[17:], callback, dataLogger)
			case 127:
				ReadInput4Long(data[17:], callback, dataLogger)
			}
		}
		if data[3] == 3 {
			switch dataLength {
			case 0:
				ReadHold1(data[17:])
			case 40:
				ReadHold2(data[17:])
			case 80:
				ReadHold3(data[17:])
			case 120:
				ReadHold4(data[17:])
			case 160:
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
func ReadHold5(data []byte) {
	log.Infof("State: %v", binary.LittleEndian.Uint16(data[0:2]))
}
func ReadInput1(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 80 {
		log.Infof("Not enough data got %v bytes but was expecting 80", len(data))
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
		log.Infof("Not enough data got %v bytes but was expecting 80", len(data))
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
	callback.ReportValue(registers.InputRegisters[registers.AC_INPUT_TYPE], int32(binary.LittleEndian.Uint16(data[74:76])), dataLogger)
}
func ReadInput3(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 80 {
		log.Infof("Not enough data got %v bytes but was expecting 80", len(data))
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
	callback.ReportValue(registers.InputRegisters[registers.BATT_CURRENT_BMS], int32(binary.LittleEndian.Uint16(data[36:38])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.Fault_Code_BMS], int32(binary.LittleEndian.Uint16(data[38:40])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.Warning_Code_BMS], int32(binary.LittleEndian.Uint16(data[40:42])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MAX_CELL_VOLT], int32(binary.LittleEndian.Uint16(data[42:44])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MIN_CELL_VOLT], int32(binary.LittleEndian.Uint16(data[44:46])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MAX_CELL_TEMP], int32(binary.LittleEndian.Uint16(data[46:48])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.MIN_CELL_TEMP], int32(binary.LittleEndian.Uint16(data[48:50])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BMS_FW_UPDATE_STATE], int32(binary.LittleEndian.Uint16(data[50:52])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.CYCLE_CNT_BMS], int32(binary.LittleEndian.Uint16(data[52:54])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.BAT_VOLT_SMPL_INV], int32(binary.LittleEndian.Uint16(data[54:56])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T1], int32(binary.LittleEndian.Uint16(data[56:58])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T2], int32(binary.LittleEndian.Uint16(data[58:60])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T3], int32(binary.LittleEndian.Uint16(data[60:62])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T4], int32(binary.LittleEndian.Uint16(data[62:64])), dataLogger)
	callback.ReportValue(registers.InputRegisters[registers.T5], int32(binary.LittleEndian.Uint16(data[64:66])), dataLogger)
}
func ReadInput4Short(data []byte, callback ParserCallback, dataLogger string) {
	if len(data) < 14 {
		log.Infof("Not enough data got %v bytes but was expecting 14", len(data))
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
	if len(data) < 42 {
		log.Infof("Not enough data got %v bytes but was expecting 42", len(data))
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
		log.Infof("Not enough data got %v bytes but was expecting 80", len(data))
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
