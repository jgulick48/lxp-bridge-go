package coordinator

import (
	"testing"

	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeWindowStates(t *testing.T) {
	m := &recMQTT{msgs: map[string]string{}}
	sink := &fakeSink{}
	c, err := build(models.Config{
		MQTTConfig: models.MQTTConfig{NameSpace: "lxp"},
		Inverters:  []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial}},
	}, deps{mqtt: m, metrics: sink, newModbus: func(models.LXPConfig, modbus.ParserCallback, modbus.Observer) (modbus.Client, error) {
		return &fakeModbus{}, nil
	}})
	require.NoError(t, err)

	// Holding block 40-79 contains registers 68-73 (AC charge windows).
	vals := make([]uint16, 40)
	vals[68-40] = 0x1e07 // 07:30
	vals[69-40] = 0x3b17 // 23:59
	vals[70-40] = 0      // 00:00
	vals[71-40] = 24     // hour 24: invalid
	vals[72-40] = 12 | 60<<8
	vals[73-40] = 0x0a09 // 09:10
	_, err = modbus.Decode(response(0x3, 40, vals), c)
	require.NoError(t, err)

	topic := func(name string) string { return "lxp/" + datalog + "/hold/" + name }
	assert.Equal(t, "07:30", m.msgs[topic("ac_charge_start_1")])
	assert.Equal(t, "23:59", m.msgs[topic("ac_charge_end_1")])
	assert.Equal(t, "00:00", m.msgs[topic("ac_charge_start_2")])
	assert.Equal(t, "09:10", m.msgs[topic("ac_charge_end_3")])
	assert.NotContains(t, m.msgs, topic("ac_charge_end_2"))
	assert.NotContains(t, m.msgs, topic("ac_charge_start_3"))
	for name := range sink.gauges {
		assert.NotContains(t, name, "ac_charge_start")
		assert.NotContains(t, name, "ac_charge_end")
	}
	// Other registers in the same block still publish as numbers.
	assert.Contains(t, sink.gauges, "lxp_ac_charge_power_cmd")
}

func TestTimeWindowCommands(t *testing.T) {
	cfg := []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial}}
	topic := func(id string) string { return "lxp/cmd/" + datalog + "/set/hold/" + id }
	expect := func(reg uint16, val uint16) []byte {
		return modbus.BuildPacket(datalog, serial, 1, 0x6, reg, modbus.GetBytesForUInt16(val)...)
	}
	tests := []struct {
		name    string
		topic   string
		payload string
		want    [][]byte
	}{
		{"HH:MM", topic("68"), "07:30", [][]byte{expect(68, 0x1e07)}},
		{"H:MM", topic("69"), "7:30", [][]byte{expect(69, 0x1e07)}},
		{"spaces", topic("76"), " 7:30\n", [][]byte{expect(76, 0x1e07)}},
		{"max", topic("89"), "23:59", [][]byte{expect(89, 0x3b17)}},
		{"midnight", topic("84"), "00:00", [][]byte{expect(84, 0)}},
		{"hour 24", topic("68"), "24:00", nil},
		{"minute 60", topic("68"), "12:60", nil},
		{"text", topic("68"), "abc", nil},
		{"no colon", topic("68"), "730", nil},
		{"empty", topic("68"), "", nil},
		{"numeric raw is not accepted", topic("68"), "7687", nil},
		{"field topic", topic("68") + "/ac_charge_start_1", "07:30", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, fakes, _ := newTest(t, cfg, nil)
			c.HandleCommand(tt.topic, []byte(tt.payload))
			assert.Equal(t, tt.want, fakes[datalog].sent())
		})
	}
}

func TestTimeRegistersAreSingleHoldRegisters(t *testing.T) {
	for hr, r := range registers.HoldRegisters {
		if !r.TimeOfDay {
			continue
		}
		assert.True(t, controllable(r), "%s", r.ShortName)
		assert.Equal(t, registers.Text, r.HomeAssistantType, "%s", r.ShortName)
		assert.Equal(t, hr, registers.HoldIDToRegisterMap[registers.HoldRegisterToIDMap[hr]], "%s", r.ShortName)
	}
}
