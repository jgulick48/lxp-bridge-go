package modbus

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sigurn/crc16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Input frames captured from an EG4 18kPV (logger BA31101550) on 2026-11-03.
const (
	input80Frame  = "a11a05006f0001c24241333131303135353061000004333139323637303036355000500200941128233002c20100000000000000000000c00000000000000000000300090084030308000000001c0d0e0d2c01fa000800300115028801000000000000000005015304333139323637303036358bd9"
	input160Frame = "a11a05006f0001c2424133313130313535306100000433313932363730303635a0005000000000000000000000000000000000000000007104c700b1970400000000000000000000000000000000000000000000006400000000000000000000000000000000000000000000000000000000008ced"
)

func decodeHex(t *testing.T, s string) *recorder {
	t.Helper()
	frame, err := hex.DecodeString(s)
	require.NoError(t, err)
	require.True(t, verifyCRC(frame[translatedHeaderLen:]), "fixture CRC")
	rec := newRecorder()
	_, err = Decode(frame, rec)
	require.NoError(t, err)
	return rec
}

// inputFrame builds a function 4 response frame holding vals from start.
func inputFrame(start uint16, vals []uint16) []byte {
	data := []byte{0, 0x4}
	data = append(data, "3192670065"...)
	data = binary.LittleEndian.AppendUint16(data, start)
	data = append(data, byte(len(vals)*2))
	for _, v := range vals {
		data = binary.LittleEndian.AppendUint16(data, v)
	}
	data = binary.LittleEndian.AppendUint16(data, crc16.Checksum(data, crcTable))
	data = append(binary.LittleEndian.AppendUint16(nil, uint16(len(data))), data...)
	body := append([]byte{1, 0xc2}, "BA31101550"...)
	body = append(body, data...)
	frame := []byte{0xa1, 0x1a, 5, 0}
	frame = binary.LittleEndian.AppendUint16(frame, uint16(len(body)))
	return append(frame, body...)
}

func TestDecodeInputBlock160(t *testing.T) {
	rec := decodeHex(t, input160Frame)
	assert.Equal(t, int32(1137), rec.values["P_LOAD"])
	assert.Equal(t, int32(199), rec.values["E_LOAD_DAY"])
	assert.Equal(t, int32(38833+4*65536), rec.values["E_LOAD_ALL"])
}

func TestDecodeInputBlock80ConsumptionPower(t *testing.T) {
	rec := decodeHex(t, input80Frame)
	assert.Equal(t, int32(1107), rec.values["CONSUMPTION_POWER"])
	assert.Equal(t, int32(3356), rec.values["MAX_CELL_VOLT"])
}

func TestDecodeLoadPowerSigned(t *testing.T) {
	vals := make([]uint16, 40)
	vals[10] = 0xFFF6 // register 170 = -10 W
	rec := newRecorder()
	_, err := Decode(inputFrame(160, vals), rec)
	require.NoError(t, err)
	assert.Equal(t, int32(-10), rec.values["P_LOAD"])
}

// The EG4 cloud dongle reads 127 registers from 127; its replies reach every
// client connected to the logger and include the load registers.
func TestDecodeInputBlock127IncludesLoad(t *testing.T) {
	vals := make([]uint16, 127)
	vals[170-127] = 1147
	vals[171-127] = 200
	vals[172-127] = 38834
	vals[173-127] = 4
	rec := newRecorder()
	_, err := Decode(inputFrame(127, vals), rec)
	require.NoError(t, err)
	assert.Equal(t, int32(1147), rec.values["P_LOAD"])
	assert.Equal(t, int32(200), rec.values["E_LOAD_DAY"])
	assert.Equal(t, int32(38834+4*65536), rec.values["E_LOAD_ALL"])
}

func TestDecodeInputBlock40PVCurrent(t *testing.T) {
	vals := make([]uint16, 40)
	vals[72-40] = 1234
	vals[73-40] = 567
	vals[74-40] = 89
	rec := newRecorder()
	_, err := Decode(inputFrame(40, vals), rec)
	require.NoError(t, err)
	assert.Equal(t, int32(1234), rec.values["I_PV_1"])
	assert.Equal(t, int32(567), rec.values["I_PV_2"])
	assert.Equal(t, int32(89), rec.values["I_PV_3"])
}

func TestDecodeInputBlock160PerLeg(t *testing.T) {
	vals := make([]uint16, 45)
	vals[170-160] = 1270
	for i := 193; i <= 204; i++ {
		vals[i-160] = uint16(i)
	}
	rec := newRecorder()
	_, err := Decode(inputFrame(160, vals), rec)
	require.NoError(t, err)
	assert.Equal(t, int32(1270), rec.values["P_LOAD"])
	want := map[string]int32{
		"V_GRID_L1": 193, "V_GRID_L2": 194, "V_GEN_L1": 195, "V_GEN_L2": 196,
		"P_INV_L1": 197, "P_INV_L2": 198, "P_REC_L1": 199, "P_REC_L2": 200,
		"P_TO_GRID_L1": 201, "P_TO_GRID_L2": 202, "P_TO_USER_L1": 203, "P_TO_USER_L2": 204,
	}
	for k, v := range want {
		assert.Equal(t, v, rec.values[k], k)
	}
}

// A 40 register read of block 160 stops at 199, so no per-leg values.
func TestDecodeInputBlock160ShortHasNoPerLeg(t *testing.T) {
	rec := decodeHex(t, input160Frame)
	assert.NotContains(t, rec.values, "V_GRID_L1")
}

// holdFrame builds a function 3 response frame.
func holdFrame(start uint16, vals []uint16) []byte {
	f := inputFrame(start, vals)
	f[translatedHeaderLen+3] = 0x3
	d := f[translatedHeaderLen:]
	n := len(d)
	binary.LittleEndian.PutUint16(d[n-2:], crc16.Checksum(d[2:n-2], crcTable))
	return f
}

type holdingRecorder struct {
	*recorder
	start uint16
	raw   []uint16
}

func (h *holdingRecorder) ReportHolding(_ string, start uint16, values []uint16) {
	h.start, h.raw = start, values
}

func TestDecodeHoldingFields(t *testing.T) {
	vals := make([]uint16, 40)
	vals[21] = 25429 // live: EPS on, AC charge off, feed-in off
	rec := &holdingRecorder{recorder: newRecorder()}
	_, err := Decode(holdFrame(0, vals), rec)
	require.NoError(t, err)
	assert.Equal(t, int32(1), rec.values["EPS_Enable"])
	assert.Equal(t, int32(0), rec.values["AC_Charge_Enable"])
	assert.Equal(t, int32(0), rec.values["Feed_In_Grid_Enable"])
	assert.Equal(t, uint16(0), rec.start)
	assert.Len(t, rec.raw, 40)
}

func TestDecodeHoldingSingleRegisterRead(t *testing.T) {
	rec := &holdingRecorder{recorder: newRecorder()}
	_, err := Decode(holdFrame(179, []uint16{4416}), rec)
	require.NoError(t, err)
	assert.Equal(t, int32(1), rec.values["Gen_Peak_Shaving_Enable"])
	assert.Equal(t, int32(0), rec.values["Smart_Load_Enable"])
	assert.Equal(t, uint16(179), rec.start)
	assert.Equal(t, []uint16{4416}, rec.raw)
}

func TestDecodeWriteEcho(t *testing.T) {
	// A function 6 reply echoes [register][value].
	f := BuildPacket("BA31101550", "3192670065", 0, 0x6, 21, GetBytesForUInt16(25429|1<<7)...)
	rec := &holdingRecorder{recorder: newRecorder()}
	_, err := Decode(f, rec)
	require.NoError(t, err)
	assert.Equal(t, int32(1), rec.values["AC_Charge_Enable"])
	assert.Equal(t, uint16(21), rec.start)
}

func TestDecodeGenDryContact(t *testing.T) {
	// Captured while the inverter was running the generator: register 105 = 8.
	rec := decodeHex(t, input80Frame)
	assert.Equal(t, int32(1), rec.values["GEN_DRY_CONTACT"])
	assert.Equal(t, int32(0), rec.values["BMS_FW_UPDATE_STATE"])

	vals := make([]uint16, 40)
	vals[105-80] = 2 // firmware update successful, contact open
	rec = newRecorder()
	_, err := Decode(inputFrame(80, vals), rec)
	require.NoError(t, err)
	assert.Equal(t, int32(0), rec.values["GEN_DRY_CONTACT"])
	assert.Equal(t, int32(2), rec.values["BMS_FW_UPDATE_STATE"])
}

func TestBatteryVoltageSampleScale(t *testing.T) {
	// Input 107 is in 0.1 V like V_BAT (input 4): raw 533 is 53.3 V.
	assert.InDelta(t, 0.1, registers.InputRegisters[registers.BAT_VOLT_SMPL_INV].Multiplier, 1e-6)
}
