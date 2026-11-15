package coordinator

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recMQTT records every published message.
type recMQTT struct {
	fakeMQTT
	mu   sync.Mutex
	msgs map[string]string
}

func (m *recMQTT) SendMessage(topic string, v interface{}, _ bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch b := v.(type) {
	case []byte:
		m.msgs[topic] = string(b)
	case string:
		m.msgs[topic] = b
	}
	return nil
}

// response builds a read reply frame: BuildPacket's request layout with a
// byte count in front of the values is the reply layout.
func response(function byte, start uint16, vals []uint16) []byte {
	data := []byte{byte(len(vals) * 2)}
	for _, v := range vals {
		data = append(data, modbus.GetBytesForUInt16(v)...)
	}
	return modbus.BuildPacket(datalog, serial, 0, function, start, data...)
}

func TestEveryRegisterIsAnnouncedAndPublished(t *testing.T) {
	m := &recMQTT{msgs: map[string]string{}}
	sink := &fakeSink{}
	c, err := build(models.Config{
		MQTTConfig: models.MQTTConfig{NameSpace: "lxp", HomeAssistant: models.HomeAssistantConfig{Enabled: true, Prefix: "homeassistant"}},
		Inverters:  []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial}},
	}, deps{mqtt: m, metrics: sink, newModbus: func(models.LXPConfig, modbus.ParserCallback, modbus.Observer) (modbus.Client, error) {
		return &fakeModbus{}, nil
	}})
	require.NoError(t, err)
	c.onMQTTConnect()

	// Every input and hold block the bridge polls, filled with 1s.
	ones := func(n int) []uint16 {
		v := make([]uint16, n)
		for i := range v {
			v[i] = 1
		}
		return v
	}
	for _, start := range []uint16{0, 40, 80, 120} {
		_, err := modbus.Decode(response(0x4, start, ones(40)), c)
		require.NoError(t, err)
	}
	_, err = modbus.Decode(response(0x4, 160, ones(45)), c)
	require.NoError(t, err)
	for _, start := range []uint16{0, 40, 80, 120, 160, 200} {
		_, err := modbus.Decode(response(0x3, start, ones(40)), c)
		require.NoError(t, err)
	}

	type cfg struct {
		UniqueID     string `json:"unique_id"`
		StateTopic   string `json:"state_topic"`
		CommandTopic string `json:"command_topic"`
	}
	ids := map[string]bool{}
	check := func(r registers.Register) {
		component := r.DiscoveryType().String()
		topic := "homeassistant/" + strings.ToLower(component) + "/lxp_" + datalog + "/" + strings.ToLower(r.ShortName) + "/config"
		raw, ok := m.msgs[topic]
		if !assert.True(t, ok, "%s not announced at %s", r.ShortName, topic) {
			return
		}
		var c cfg
		require.NoError(t, json.Unmarshal([]byte(raw), &c))
		assert.False(t, ids[c.UniqueID], "duplicate unique_id %s", c.UniqueID)
		ids[c.UniqueID] = true
		_, published := m.msgs[c.StateTopic]
		assert.True(t, published, "%s: nothing published to state topic %s", r.ShortName, c.StateTopic)
		switch component {
		case "Number", "Switch":
			assert.NotEmpty(t, c.CommandTopic, "%s is a %s without a command topic", r.ShortName, component)
		default:
			assert.Empty(t, c.CommandTopic, "%s", r.ShortName)
		}
	}
	for _, r := range registers.InputRegisters {
		check(r)
	}
	for _, r := range registers.HoldRegisters {
		check(r)
	}
	assert.Len(t, ids, len(registers.InputRegisters)+len(registers.HoldRegisters))

	// Every register is also exported to statsd as <namespace>_<name>, and no
	// two registers share a gauge name.
	gauges := map[string]string{}
	all := []registers.Register{}
	for _, r := range registers.InputRegisters {
		all = append(all, r)
	}
	for _, r := range registers.HoldRegisters {
		all = append(all, r)
	}
	for _, r := range all {
		name := "lxp_" + strings.ToLower(r.ShortName)
		other, dup := gauges[name]
		assert.False(t, dup, "%s and %s share the statsd gauge %s", r.ShortName, other, name)
		gauges[name] = r.ShortName
		_, ok := sink.gauges[name]
		assert.True(t, ok, "%s: no statsd gauge %s", r.ShortName, name)
	}
	t.Logf("%d entities announced and published", len(ids))
}
