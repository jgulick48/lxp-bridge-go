package registers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDevice = Device{Name: "lxp_DL", Identifiers: []string{"lxp_DL"}}

func discovery(t *testing.T, hr HoldRegister) map[string]interface{} {
	t.Helper()
	r := HoldRegisters[hr]
	b, err := json.Marshal(r.ToJson(int(hr), testDevice, "lxp", "DL"))
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func TestSwitchDiscovery(t *testing.T) {
	m := discovery(t, AC_Charge_Enable)
	assert.Equal(t, "lxp/cmd/DL/set/hold/21/ac_charge_enable", m["command_topic"])
	assert.Equal(t, "lxp/DL/hold/ac_charge_enable", m["state_topic"])
	assert.Equal(t, "1", m["payload_on"])
	assert.Equal(t, "0", m["state_off"])
	assert.NotContains(t, m, "min")
	assert.NotContains(t, m, "unit_of_measurement")
}

func TestNumberDiscoverySendsZeroMin(t *testing.T) {
	m := discovery(t, EPS_Discharge_Cut_Off_SOC)
	assert.Equal(t, "lxp/cmd/DL/set/hold/125", m["command_topic"])
	assert.Equal(t, 0.0, m["min"])
	assert.Equal(t, 100.0, m["max"])

	m = discovery(t, AC_Charge_Power_CMD)
	assert.InDelta(t, 12.0, m["max"], 1e-4)
	assert.InDelta(t, 0.1, m["step"], 1e-6)
}

func TestFieldValueAndSetField(t *testing.T) {
	r := HoldRegisters[AC_Charge_Type]
	assert.Equal(t, uint16(1), r.FieldValue(0x82))
	assert.Equal(t, uint16(0x84), r.SetField(0x82, 2))
	assert.Equal(t, uint16(0x80), r.SetField(0x82, 0))
	sw := HoldRegisters[Feed_In_Grid_Enable]
	assert.Equal(t, uint16(0x8001), sw.SetField(1, 1))
	assert.Equal(t, uint16(1), sw.FieldValue(0x8001))
	whole := HoldRegisters[Gen_Chg_Start_SOC]
	assert.Equal(t, uint16(20), whole.SetField(99, 20))
}

func TestBinarySensorDiscovery(t *testing.T) {
	r := InputRegisters[GEN_DRY_CONTACT]
	assert.Equal(t, "binary_sensor", r.HomeAssistantType.String())
	b, err := json.Marshal(r.ToJson(int(GEN_DRY_CONTACT), testDevice, "lxp", "DL"))
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, "lxp/DL/input/gen_dry_contact", m["state_topic"])
	assert.Equal(t, "1", m["payload_on"])
	assert.Equal(t, "0", m["payload_off"])
	assert.Equal(t, "running", m["device_class"])
	assert.NotContains(t, m, "command_topic")
	assert.NotContains(t, m, "unit_of_measurement")
}
