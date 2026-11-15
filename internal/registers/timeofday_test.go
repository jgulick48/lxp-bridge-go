package registers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeOfDayEncodeDecode(t *testing.T) {
	tests := []struct {
		raw    uint16
		hour   int
		minute int
		text   string
	}{
		{0, 0, 0, "00:00"},
		{0x3b17, 23, 59, "23:59"},
		{0x1e07, 7, 30, "07:30"},
	}
	for _, tt := range tests {
		h, m, ok := DecodeTimeOfDay(tt.raw)
		assert.True(t, ok)
		assert.Equal(t, tt.hour, h)
		assert.Equal(t, tt.minute, m)
		raw, err := EncodeTimeOfDay(tt.hour, tt.minute)
		require.NoError(t, err)
		assert.Equal(t, tt.raw, raw)
		text, ok := FormatTimeOfDay(tt.raw)
		assert.True(t, ok)
		assert.Equal(t, tt.text, text)
		parsed, err := ParseTimeOfDay(tt.text)
		require.NoError(t, err)
		assert.Equal(t, tt.raw, parsed)
	}
	assert.Equal(t, uint16(15127), uint16(0x3b17))
}

func TestTimeOfDayInvalid(t *testing.T) {
	_, _, ok := DecodeTimeOfDay(24 | 0<<8)
	assert.False(t, ok, "hour 24")
	_, _, ok = DecodeTimeOfDay(12 | 60<<8)
	assert.False(t, ok, "minute 60")
	_, ok = FormatTimeOfDay(0x3c18)
	assert.False(t, ok)
	_, err := EncodeTimeOfDay(24, 0)
	assert.Error(t, err)
	_, err = EncodeTimeOfDay(0, 60)
	assert.Error(t, err)
	_, err = EncodeTimeOfDay(-1, 0)
	assert.Error(t, err)
	for _, s := range []string{"24:00", "12:60", "abc", "730", "", "7:3", "007:30", "1:2:3", "-1:30", "+7:30", "7:30pm"} {
		_, err := ParseTimeOfDay(s)
		assert.Error(t, err, "%q", s)
	}
	raw, err := ParseTimeOfDay(" 7:30\n")
	require.NoError(t, err)
	assert.Equal(t, uint16(0x1e07), raw)
}

func TestTimeOfDayDiscovery(t *testing.T) {
	m := discovery(t, AC_Charge_Start_1)
	assert.Equal(t, "AC Charge Window 1 Start", m["name"])
	assert.Equal(t, "lxp/DL/hold/ac_charge_start_1", m["state_topic"])
	assert.Equal(t, "lxp/cmd/DL/set/hold/68", m["command_topic"])
	assert.Equal(t, TimeOfDayPattern, m["pattern"])
	assert.Equal(t, "config", m["entity_category"])
	assert.NotContains(t, m, "min")
	assert.NotContains(t, m, "max")
	assert.NotContains(t, m, "step")
	assert.NotContains(t, m, "unit_of_measurement")
	assert.Equal(t, "Text", HoldRegisters[AC_Charge_Start_1].HomeAssistantType.String())
	assert.Equal(t, 89, HoldRegisterToIDMap[Forced_Dischg_End_3])
	assert.Equal(t, 81, HoldRegisterToIDMap[Forced_Chg_End_3])
	assert.Equal(t, 73, HoldRegisterToIDMap[AC_Charge_End_3])
}
