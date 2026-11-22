package coordinator

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/jgulick48/lxp-bridge-go/internal/mqtt"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// subMQTT records topics passed to SubMultiple.
type subMQTT struct {
	recMQTT
	subs [][]string
}

func (m *subMQTT) SubMultiple(t []string) { m.subs = append(m.subs, t) }

func roConfig(ro bool, cs models.ClockSyncConfig) models.Config {
	return models.Config{
		ReadOnly:   ro,
		MQTTConfig: models.MQTTConfig{NameSpace: "lxp", HomeAssistant: models.HomeAssistantConfig{Enabled: true, Prefix: "homeassistant"}},
		Inverters:  []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial, ClockSync: cs}},
	}
}

// newRO builds a client with one fake inverter. An empty model means
// auto-detection in production; here it defaults to the 18kPV layout so tests
// not about detection keep their meaning. Use newROExact to keep it empty.
func newRO(t *testing.T, cfg models.Config, m mqtt.Client, now func() time.Time) (*client, *fakeModbus, *fakeSink) {
	t.Helper()
	cfg.Inverters = append([]models.LXPConfig(nil), cfg.Inverters...)
	for i := range cfg.Inverters {
		if cfg.Inverters[i].Model == "" {
			cfg.Inverters[i].Model = "18kpv"
		}
	}
	return newROExact(t, cfg, m, now)
}

func newROExact(t *testing.T, cfg models.Config, m mqtt.Client, now func() time.Time) (*client, *fakeModbus, *fakeSink) {
	t.Helper()
	f := &fakeModbus{}
	sink := &fakeSink{}
	c, err := build(cfg, deps{mqtt: m, metrics: sink, now: now, newModbus: func(models.LXPConfig, modbus.ParserCallback, modbus.Observer) (modbus.Client, error) {
		return f, nil
	}})
	require.NoError(t, err)
	return c, f, sink
}

// functionByte is the modbus function of a frame built by BuildPacket:
// 4 byte start, 2 length, 2 header, datalog, 2 length, mode, then function.
func functionByte(frame []byte) byte { return frame[4+2+2+len(datalog)+2+1] }

func assertOnlyReads(t *testing.T, f *fakeModbus) {
	t.Helper()
	for _, fr := range f.sent() {
		assert.Contains(t, []byte{0x3, 0x4}, functionByte(fr), "non-read frame sent in read-only mode: % x", fr)
	}
}

func roClock(t *testing.T, ro bool) *clockHarness {
	h := &clockHarness{t: t, loc: denver(t), now: time.Date(2026, 11, 15, 6, 53, 44, 0, time.UTC)}
	h.c, h.f, h.sink = newRO(t, roConfig(ro, syncOn), &recMQTT{msgs: map[string]string{}}, func() time.Time { return h.now })
	return h
}

func TestFunctionByteOffset(t *testing.T) {
	assert.Equal(t, byte(0x6), functionByte(modbus.BuildPacket(datalog, serial, 1, 0x6, 1, 0, 0)))
	assert.Equal(t, byte(0x10), functionByte(modbus.BuildWriteMulti(datalog, serial, 12, []uint16{1, 2, 3})))
	assert.Equal(t, byte(0x3), functionByte(modbus.BuildPacket(datalog, serial, 0, 0x3, 0, 0, 40)))
}

func TestReadOnlyDoesNotSubscribeToCommands(t *testing.T) {
	for _, ro := range []bool{true, false} {
		m := &subMQTT{recMQTT: recMQTT{msgs: map[string]string{}}}
		c, _, _ := newRO(t, roConfig(ro, models.ClockSyncConfig{}), m, nil)
		c.Start(t.Context())
		c.Close()
		if ro {
			assert.Empty(t, m.subs)
		} else {
			assert.Equal(t, [][]string{{"lxp/cmd/" + datalog + "/set/#"}}, m.subs)
		}
	}
}

func TestReadOnlyCommandsSendNothing(t *testing.T) {
	c, f, _ := newRO(t, roConfig(true, models.ClockSyncConfig{}), &recMQTT{msgs: map[string]string{}}, nil)
	topic := func(id string) string { return "lxp/cmd/" + datalog + "/set/hold/" + id }
	c.HandleCommand(topic("196"), []byte("20"))                   // number
	c.HandleCommand(topic("21")+"/ac_charge_enable", []byte("1")) // bit field
	c.HandleCommand(topic("72"), []byte("01:30"))                 // time window
	c.ReportHolding(datalog, 21, []uint16{0})                     // would apply a queued field write
	assert.Empty(t, f.sent())
	assert.Empty(t, c.inverters[datalog].pending)
}

func TestReadOnlyClockSyncNeverWrites(t *testing.T) {
	h := roClock(t, true)
	h.report(-10 * time.Minute)
	h.now = h.now.Add(2 * time.Hour)
	h.report(-10 * time.Minute)
	assert.Empty(t, h.f.sent())
	v, ok := h.driftGauge()
	require.True(t, ok)
	assert.InDelta(t, -600, v, 0.001)
}

// TestReadOnlyOnlyReadFrames drives every write path plus the polls and
// requires the modbus client to see read frames only. A new write path that
// bypasses sendWrite fails here.
func TestReadOnlyOnlyReadFrames(t *testing.T) {
	h := roClock(t, true)
	h.f.connected = true
	topic := func(id string) string { return "lxp/cmd/" + datalog + "/set/hold/" + id }
	h.c.HandleCommand(topic("196"), []byte("20"))
	h.c.HandleCommand(topic("21")+"/ac_charge_enable", []byte("1"))
	h.c.HandleCommand(topic("72"), []byte("01:30"))
	h.c.sendHoldingPolls(h.c.inverters[datalog])
	h.report(-10 * time.Minute)
	h.c.ReportHolding(datalog, 21, []uint16{0})
	assert.NotEmpty(t, h.f.sent(), "polls should still be sent")
	assertOnlyReads(t, h.f)

	// The choke point itself refuses writes.
	h.c.sendWrite(h.c.inverters[datalog], modbus.BuildPacket(datalog, serial, 1, 0x6, 1, 0, 0), "test")
	h.c.sendWrite(h.c.inverters[datalog], modbus.BuildWriteMulti(datalog, serial, 12, []uint16{1, 2, 3}), "test")
	assertOnlyReads(t, h.f)

	// Sanity: the same paths do write when not read-only.
	h2 := roClock(t, false)
	h2.c.HandleCommand(topic("196"), []byte("20"))
	h2.c.HandleCommand(topic("21")+"/ac_charge_enable", []byte("1"))
	h2.c.ReportHolding(datalog, 21, []uint16{0})
	h2.report(-10 * time.Minute)
	var fns []byte
	for _, fr := range h2.f.sent() {
		fns = append(fns, functionByte(fr))
	}
	assert.Contains(t, fns, byte(0x6))
	assert.Contains(t, fns, byte(0x10))
}

func TestReadOnlyDiscovery(t *testing.T) {
	m := &recMQTT{msgs: map[string]string{}}
	c, _, _ := newRO(t, roConfig(true, models.ClockSyncConfig{}), m, nil)
	c.onMQTTConnect()

	check := func(r registers.Register, oldComponent, newComponent string) {
		base := "/lxp_" + datalog + "/" + strings.ToLower(r.ShortName) + "/config"
		old, ok := m.msgs["homeassistant/"+oldComponent+base]
		assert.True(t, ok, "%s: old topic not cleared", r.ShortName)
		assert.Equal(t, "", old, r.ShortName)
		raw, ok := m.msgs["homeassistant/"+newComponent+base]
		require.True(t, ok, "%s: not announced as %s", r.ShortName, newComponent)
		var js map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(raw), &js))
		assert.NotContains(t, js, "command_topic", r.ShortName)
	}
	var number, sw, text registers.Register
	for _, r := range registers.HoldRegisters {
		switch {
		case r.Writeable && r.HomeAssistantType == registers.Number && number.ShortName == "":
			number = r
		case r.Writeable && r.HomeAssistantType == registers.Switch && sw.ShortName == "":
			sw = r
		case r.Writeable && r.HomeAssistantType == registers.Text && text.ShortName == "":
			text = r
		}
	}
	require.NotEmpty(t, number.ShortName)
	require.NotEmpty(t, sw.ShortName)
	require.NotEmpty(t, text.ShortName)
	check(number, "number", "sensor")
	check(sw, "switch", "binary_sensor")
	check(text, "text", "sensor")

	// No announced entity has a command topic.
	for topic, raw := range m.msgs {
		if raw == "" || !strings.HasSuffix(topic, "/config") {
			continue
		}
		var js map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(raw), &js), topic)
		assert.NotContains(t, js, "command_topic", topic)
		assert.False(t, strings.Contains(topic, "/number/") || strings.Contains(topic, "/switch/") || strings.Contains(topic, "/text/"), topic)
	}
}

// Turning read_only off again must clear the read-only sensors it announced,
// or Home Assistant shows each control twice.
func TestLeavingReadOnlyClearsReadOnlyEntities(t *testing.T) {
	m := &recMQTT{msgs: map[string]string{}}
	c, _, _ := newRO(t, roConfig(false, models.ClockSyncConfig{}), m, nil)
	c.onMQTTConnect()

	for _, r := range registers.HoldRegisters {
		if !r.Writeable {
			continue
		}
		base := "/lxp_" + datalog + "/" + strings.ToLower(r.ShortName) + "/config"
		control := "homeassistant/" + strings.ToLower(r.DiscoveryType().String()) + base
		ro := r.AsReadOnly()
		readOnly := "homeassistant/" + strings.ToLower(ro.DiscoveryType().String()) + base
		require.NotEqual(t, control, readOnly, r.ShortName)
		assert.NotEmpty(t, m.msgs[control], "%s: control not announced", r.ShortName)
		v, ok := m.msgs[readOnly]
		assert.True(t, ok && v == "", "%s: read-only variant %s not cleared", r.ShortName, readOnly)
	}
}

// discoveryConfigs returns every announced (non-empty) discovery config by topic.
func discoveryConfigs(t *testing.T, m *recMQTT) map[string]map[string]interface{} {
	t.Helper()
	out := map[string]map[string]interface{}{}
	for topic, raw := range m.msgs {
		if raw == "" || !strings.HasPrefix(topic, "homeassistant/") || !strings.HasSuffix(topic, "/config") {
			continue
		}
		var js map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(raw), &js), topic)
		out[topic] = js
	}
	return out
}

// Home Assistant rejects entity_category "config" on sensors and binary
// sensors (#36), so read-only mode must never announce one.
func TestReadOnlyNoConfigCategoryOnSensors(t *testing.T) {
	for _, r := range registers.HoldRegisters {
		ro := r.AsReadOnly()
		if ro.DiscoveryType() == registers.Sensor || ro.DiscoveryType() == registers.BinarySensor {
			assert.NotEqual(t, "config", ro.EntityCategory, r.ShortName)
		}
		if r.Writeable && r.EntityCategory == "config" {
			assert.Equal(t, "diagnostic", ro.EntityCategory, r.ShortName)
		}
	}
	for _, r := range registers.InputRegisters {
		assert.NotEqual(t, "config", r.AsReadOnly().EntityCategory, r.ShortName)
	}

	m := &recMQTT{msgs: map[string]string{}}
	c, _, _ := newRO(t, roConfig(true, models.ClockSyncConfig{}), m, nil)
	c.onMQTTConnect()
	diagnostic := 0
	for topic, js := range discoveryConfigs(t, m) {
		if strings.Contains(topic, "/sensor/") || strings.Contains(topic, "/binary_sensor/") {
			assert.NotEqual(t, "config", js["entity_category"], topic)
		}
		if js["entity_category"] == "diagnostic" {
			diagnostic++
		}
	}
	assert.Positive(t, diagnostic, "downgraded config entities should be announced as diagnostic")
}

// Writeable mode keeps entity_category config on the controls.
func TestWriteableModeKeepsConfigCategory(t *testing.T) {
	m := &recMQTT{msgs: map[string]string{}}
	c, _, _ := newRO(t, roConfig(false, models.ClockSyncConfig{}), m, nil)
	c.onMQTTConnect()
	cfgs := discoveryConfigs(t, m)
	n := 0
	for _, r := range registers.HoldRegisters {
		if r.EntityCategory != "config" {
			continue
		}
		topic := "homeassistant/" + strings.ToLower(r.DiscoveryType().String()) + "/lxp_" + datalog + "/" + strings.ToLower(r.ShortName) + "/config"
		js, ok := cfgs[topic]
		require.True(t, ok, topic)
		assert.Equal(t, "config", js["entity_category"], topic)
		assert.Contains(t, js, "command_topic", topic)
		n++
	}
	assert.Positive(t, n)
}
