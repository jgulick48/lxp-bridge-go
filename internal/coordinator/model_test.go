package coordinator

import (
	"strings"
	"testing"

	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func modelConfig(model string) models.Config {
	return models.Config{
		MQTTConfig: models.MQTTConfig{NameSpace: "lxp", HomeAssistant: models.HomeAssistantConfig{Enabled: true, Prefix: "homeassistant"}},
		Inverters:  []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial, Model: model}},
	}
}

func newModel(t *testing.T, model string) (*client, *fakeModbus, *fakeSink, *recMQTT) {
	t.Helper()
	m := &recMQTT{msgs: map[string]string{}}
	c, f, sink := newRO(t, modelConfig(model), m, nil)
	return c, f, sink, m
}

func TestBuildRejectsUnknownModel(t *testing.T) {
	_, err := build(modelConfig("foo"), deps{mqtt: fakeMQTT{}, newModbus: func(models.LXPConfig, modbus.ParserCallback, modbus.Observer) (modbus.Client, error) {
		return &fakeModbus{}, nil
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), datalog)
	assert.Contains(t, err.Error(), "model")

	c, _, _, _ := newModel(t, " 6000XP ")
	assert.Equal(t, registers.Model6000XP, c.inverters[datalog].model)
}

func TestReportValue6000XPDropsRegisters(t *testing.T) {
	c, _, sink, m := newModel(t, "6000xp")
	for _, id := range []registers.InputRegister{registers.V_PV_3, registers.P_GEN, registers.GEN_DRY_CONTACT} {
		c.ReportValue(registers.InputRegisters[id], 5, datalog)
		assert.NotContains(t, m.msgs, "lxp/"+datalog+"/input/"+strings.ToLower(id.String()))
		assert.NotContains(t, sink.gauges, "lxp_"+strings.ToLower(id.String()))
	}
	c.ReportValue(registers.InputRegisters[registers.SOC], 55, datalog)
	c.ReportValue(registers.InputRegisters[registers.V_GEN], 2400, datalog)
	assert.Equal(t, "55", m.msgs["lxp/"+datalog+"/input/soc"])
	assert.Equal(t, "240", m.msgs["lxp/"+datalog+"/input/v_gen"])
	assert.Contains(t, sink.gauges, "lxp_soc")

	d, _, dsink, dm := newModel(t, "")
	d.ReportValue(registers.InputRegisters[registers.P_GEN], 5, datalog)
	assert.Equal(t, "5", dm.msgs["lxp/"+datalog+"/input/p_gen"])
	assert.Contains(t, dsink.gauges, "lxp_p_gen")
}

func TestGenRatePowerScale(t *testing.T) {
	c, _, sink, m := newModel(t, "6000xp")
	c.ReportValue(registers.HoldRegisters[registers.Gen_Rate_power], 7370, datalog)
	assert.Equal(t, "7370", m.msgs["lxp/"+datalog+"/hold/gen_rate_power"])
	assert.Equal(t, float64(7370), sink.gauges["lxp_gen_rate_power"])

	d, _, dsink, dm := newModel(t, "")
	d.ReportValue(registers.HoldRegisters[registers.Gen_Rate_power], 7370, datalog)
	assert.Equal(t, "737", dm.msgs["lxp/"+datalog+"/hold/gen_rate_power"])
	assert.InDelta(t, 737, dsink.gauges["lxp_gen_rate_power"], 0.01)
}

func announced(m *recMQTT, component string, name string) (string, bool) {
	v, ok := m.msgs["homeassistant/"+component+"/lxp_"+datalog+"/"+name+"/config"]
	return v, ok
}

func TestDiscovery6000XP(t *testing.T) {
	c, _, _, m := newModel(t, "6000xp")
	c.onMQTTConnect()
	cfgs := discoveryConfigs(t, m)

	// Omitted registers: nothing announced, every possible topic cleared.
	for _, tc := range []struct{ name, component string }{
		{"p_gen", "sensor"}, {"v_pv_3", "sensor"}, {"gen_dry_contact", "binary_sensor"},
		{"gen_quick_start", "switch"}, {"gen_quick_start", "binary_sensor"},
		{"ac_charge_start_1", "text"}, {"ac_charge_start_1", "sensor"},
		{"gen_cool_down_time", "number"}, {"gen_cool_down_time", "sensor"},
	} {
		v, ok := announced(m, tc.component, tc.name)
		assert.True(t, ok && v == "", "%s/%s should be cleared", tc.component, tc.name)
	}
	for topic := range cfgs {
		for _, gone := range []string{"/p_gen/", "/v_pv_3/", "/gen_quick_start/", "/ac_charge_start_1/", "/gen_dry_contact/"} {
			assert.NotContains(t, topic, gone)
		}
	}

	// gen_rate_power is a read-only sensor in W; the number is cleared.
	js, ok := cfgs["homeassistant/sensor/lxp_"+datalog+"/gen_rate_power/config"]
	require.True(t, ok)
	assert.Equal(t, "W", js["unit_of_measurement"])
	assert.NotContains(t, js, "command_topic")
	v, ok := announced(m, "number", "gen_rate_power")
	assert.True(t, ok && v == "")

	// Kept registers and the new ones.
	for _, name := range []string{"v_gen", "f_gen", "parallel_role", "parallel_phase", "parallel_num", "ac_couple_power"} {
		_, ok := cfgs["homeassistant/sensor/lxp_"+datalog+"/"+name+"/config"]
		assert.True(t, ok, name)
	}
	_, ok = cfgs["homeassistant/switch/lxp_"+datalog+"/eps_enable/config"]
	assert.True(t, ok)
}

func TestDiscoveryDefaultModelUnchanged(t *testing.T) {
	c, _, _, m := newModel(t, "")
	c.onMQTTConnect()
	cfgs := discoveryConfigs(t, m)
	for _, tc := range []struct{ component, name string }{
		{"switch", "gen_quick_start"}, {"text", "ac_charge_start_1"}, {"number", "gen_rate_power"},
		{"sensor", "p_gen"}, {"sensor", "parallel_role"}, {"sensor", "ac_couple_power"},
	} {
		_, ok := cfgs["homeassistant/"+tc.component+"/lxp_"+datalog+"/"+tc.name+"/config"]
		assert.True(t, ok, tc.name)
	}
	// Only the read-only variants of writeable registers are cleared.
	v, ok := announced(m, "sensor", "gen_rate_power")
	assert.True(t, ok && v == "")
}

func TestHandleCommand6000XP(t *testing.T) {
	topic := func(id string) string { return "lxp/cmd/" + datalog + "/set/hold/" + id }
	c, f, _, _ := newModel(t, "6000xp")
	c.HandleCommand(topic("110")+"/gen_quick_start", []byte("ON"))
	c.ReportHolding(datalog, 110, []uint16{1056})
	c.HandleCommand(topic("177"), []byte("5000"))
	c.HandleCommand(topic("68"), []byte("01:30"))
	c.HandleCommand(topic("237"), []byte("5"))
	assert.Empty(t, f.sent())
	assert.Empty(t, c.inverters[datalog].pending)

	// Other settings still work.
	c.HandleCommand(topic("21")+"/ac_charge_enable", []byte("1"))
	assert.Len(t, f.sent(), 1)

	d, df, _, _ := newModel(t, "")
	d.HandleCommand(topic("110")+"/gen_quick_start", []byte("ON"))
	assert.Len(t, df.sent(), 1) // reads the register first
	d.ReportHolding(datalog, 110, []uint16{1056})
	assert.Len(t, df.sent(), 2)
}

func TestModelHint(t *testing.T) {
	hook := logtest.NewGlobal()
	defer hook.Reset()
	old := logrus.GetLevel()
	logrus.SetLevel(logrus.InfoLevel)
	defer logrus.SetLevel(old)

	vals := make([]uint16, 20)
	vals[19] = 54
	c, _, _, _ := newModel(t, "")
	c.ReportHolding(datalog, 0, vals)
	c.ReportHolding(datalog, 0, vals) // only once
	require.Len(t, hook.AllEntries(), 1)
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	assert.Contains(t, hook.LastEntry().Message, "set model: 6000xp")

	hook.Reset()
	c, _, _, _ = newModel(t, "6000xp")
	c.ReportHolding(datalog, 0, vals)
	assert.Equal(t, logrus.InfoLevel, hook.LastEntry().Level)
	hook.Reset()
	vals[19] = 18
	c, _, _, _ = newModel(t, "6000xp")
	c.ReportHolding(datalog, 0, vals)
	assert.Equal(t, logrus.WarnLevel, hook.LastEntry().Level)
	hook.Reset()
	c.ReportHolding(datalog, 20, vals) // does not cover 19
	assert.Empty(t, hook.AllEntries())
}
