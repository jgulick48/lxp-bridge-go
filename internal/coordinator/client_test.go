package coordinator

import (
	"math/bits"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/metrics"
	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.PanicLevel)
	os.Exit(m.Run())
}

// fakeModbus records commands; it never touches the network.
type fakeModbus struct {
	mu        sync.Mutex
	commands  [][]byte
	last      time.Time
	connected bool
}

func (f *fakeModbus) Connect() {}
func (f *fakeModbus) Close()   {}
func (f *fakeModbus) SendCommand(b []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, append([]byte(nil), b...))
}
func (f *fakeModbus) LastFrame() time.Time { return f.last }
func (f *fakeModbus) Connected() bool      { return f.connected }
func (f *fakeModbus) sent() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.commands...)
}

type fakeMQTT struct{}

func (fakeMQTT) Close()                                      {}
func (fakeMQTT) IsEnabled() bool                             { return false }
func (fakeMQTT) Connect()                                    {}
func (fakeMQTT) SendMessage(string, interface{}, bool) error { return nil }
func (fakeMQTT) SubMultiple([]string)                        {}

type fakeSink struct {
	mu     sync.Mutex
	gauges map[string]float64
	counts map[string]int64
}

func (s *fakeSink) Gauge(n string, v float64, tags []string, _ float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gauges == nil {
		s.gauges = map[string]float64{}
	}
	s.gauges[n] = v
	return nil
}
func (s *fakeSink) Count(n string, v int64, tags []string, _ float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts == nil {
		s.counts = map[string]int64{}
	}
	s.counts[n] += v
	return nil
}
func (s *fakeSink) Close() error { return nil }

var _ metrics.Sink = (*fakeSink)(nil)

const (
	datalog = "BA31101550"
	serial  = "3192670065"
)

func newTest(t *testing.T, inverters []models.LXPConfig, now func() time.Time) (*client, map[string]*fakeModbus, *fakeSink) {
	t.Helper()
	fakes := map[string]*fakeModbus{}
	sink := &fakeSink{}
	c, err := build(models.Config{
		MQTTConfig: models.MQTTConfig{NameSpace: "lxp"},
		Inverters:  inverters,
	}, deps{
		mqtt:    fakeMQTT{},
		metrics: sink,
		now:     now,
		newModbus: func(cfg models.LXPConfig, cb modbus.ParserCallback, obs modbus.Observer) (modbus.Client, error) {
			f := &fakeModbus{}
			fakes[cfg.DataLog] = f
			return f, nil
		},
	})
	require.NoError(t, err)
	return c, fakes, sink
}

func TestHealth(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := start
	clock := func() time.Time { return now }
	inv := models.LXPConfig{Enabled: true, DataLog: datalog, Serial: serial}

	tests := []struct {
		name       string
		cfg        models.LXPConfig
		advance    time.Duration
		lastFrame  func() time.Time
		wantStatus int
		wantBody   string
	}{
		{"fresh start, no frame yet, within default 5m", inv, 4 * time.Minute, nil, http.StatusNoContent, ""},
		{"never got a frame, beyond default 5m", inv, 6 * time.Minute, nil, http.StatusServiceUnavailable, datalog},
		{"recent frame", inv, 30 * time.Minute, func() time.Time { return start.Add(29*time.Minute + 30*time.Second) }, http.StatusNoContent, ""},
		{"stale frame beyond default 5m", inv, 30 * time.Minute, func() time.Time { return start.Add(10 * time.Minute) }, http.StatusServiceUnavailable, "20m0s"},
		{"custom 3m duration is honoured", func() models.LXPConfig { i := inv; i.HealthCheckDuration = 3 * time.Minute; return i }(),
			30 * time.Minute, func() time.Time { return start.Add(26 * time.Minute) }, http.StatusServiceUnavailable, "limit 3m0s"},
		{"disabled inverter ignored", models.LXPConfig{DataLog: "OFF"}, time.Hour, nil, http.StatusNoContent, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now = start
			c, fakes, _ := newTest(t, []models.LXPConfig{tt.cfg}, clock)
			now = start.Add(tt.advance)
			if f, ok := fakes[tt.cfg.DataLog]; ok && tt.lastFrame != nil {
				f.last = tt.lastFrame()
			}
			status, body := c.Health()
			assert.Equal(t, tt.wantStatus, status)
			assert.Contains(t, body, tt.wantBody)

			rec := httptest.NewRecorder()
			c.HealthHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus != http.StatusNoContent {
				assert.Contains(t, rec.Body.String(), "no valid frame")
			}
		})
	}
}

func TestHealthEveryInverterMustBeFresh(t *testing.T) {
	start := time.Now()
	now := start
	cfgs := []models.LXPConfig{
		{Enabled: true, DataLog: "A", Serial: "1"},
		{Enabled: true, DataLog: "B", Serial: "2"},
	}
	c, fakes, _ := newTest(t, cfgs, func() time.Time { return now })
	now = start.Add(time.Hour)
	fakes["A"].last = now
	fakes["B"].last = now.Add(-10 * time.Minute)
	status, body := c.Health()
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.Contains(t, body, "inverter B")
	fakes["B"].last = now
	status, _ = c.Health()
	assert.Equal(t, http.StatusNoContent, status)
}

func TestHandleCommand(t *testing.T) {
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
		{"SOC start in range", topic("196"), "20", [][]byte{expect(196, 20)}},
		{"SOC start at max", topic("196"), "90", [][]byte{expect(196, 90)}},
		{"SOC end at min with whitespace", topic("197"), " 20\n", [][]byte{expect(197, 20)}},
		{"gen charge volt in display units", topic("194"), "45.0", [][]byte{expect(194, 450)}},
		{"gen charge volt with one decimal", topic("194"), "48.5", [][]byte{expect(194, 485)}},
		{"gen charge volt as whole volts", topic("195"), "54", [][]byte{expect(195, 540)}},
		{"gen charge volt finer than step", topic("194"), "48.55", nil},
		{"gen charge volt raw value is out of range", topic("194"), "450", nil},
		{"gen charge volt below min", topic("194"), "38.3", nil},
		{"max gen chg current", topic("198"), "60", [][]byte{expect(198, 60)}},
		{"max gen chg current above max", topic("198"), "251", nil},
		{"AC charge power in kW", topic("66"), "8.5", [][]byte{expect(66, 85)}},
		{"AC charge power above 12 kW", topic("66"), "12.1", nil},
		{"AC charge SOC limit 101", topic("67"), "101", [][]byte{expect(67, 101)}},
		{"stop discharge voltage", topic("202"), "44.0", [][]byte{expect(202, 440)}},
		{"bit field register without field", topic("21"), "1", nil},
		{"out of range high", topic("196"), "91", nil},
		{"out of range low", topic("197"), "19", nil},
		{"negative", topic("196"), "-1", nil},
		{"non-integer SOC", topic("196"), "20.5", nil},
		{"SOC as float with zero fraction", topic("196"), "20.0", [][]byte{expect(196, 20)}},
		{"NaN", topic("196"), "NaN", nil},
		{"garbage", topic("196"), "abc", nil},
		{"empty", topic("196"), "", nil},
		{"huge", topic("196"), "99999999999", nil},
		{"read-only/non-writeable hold register (language)", topic("16"), "1", nil},
		{"charge power percent", topic("64"), "50", [][]byte{expect(64, 50)}},
		{"unknown register", topic("9999"), "1", nil},
		{"non-numeric register", topic("abc"), "1", nil},
		{"input register path", "lxp/cmd/" + datalog + "/set/input/1", "1", nil},
		{"unknown datalog", "lxp/cmd/NOPE/set/hold/196", "20", nil},
		{"wrong namespace", "other/cmd/" + datalog + "/set/hold/196", "20", nil},
		{"short topic", "lxp/cmd/" + datalog, "20", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, fakes, _ := newTest(t, cfg, nil)
			c.HandleCommand(tt.topic, []byte(tt.payload))
			assert.Equal(t, tt.want, fakes[datalog].sent())
		})
	}
}

func TestEveryWritableRegisterIsControllable(t *testing.T) {
	// Guard: anything we allow must be exposed to HA as a settable entity type.
	for id, r := range registers.HoldRegisters {
		if r.Writeable {
			assert.True(t, controllable(r), "%s", id)
		}
	}
	assert.False(t, controllable(registers.Register{Writeable: true, HomeAssistantType: registers.Sensor}))
	assert.False(t, controllable(registers.Register{Writeable: false, HomeAssistantType: registers.Number}))
}

func TestHealthGaugesAndObserverMetrics(t *testing.T) {
	start := time.Now()
	now := start
	c, fakes, sink := newTest(t, []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial}}, func() time.Time { return now })
	fakes[datalog].connected = true
	fakes[datalog].last = start
	now = start.Add(42 * time.Second)
	c.emitHealthGauges()
	assert.Equal(t, 1.0, sink.gauges["lxp_bridge_connected"])
	assert.InDelta(t, 42.0, sink.gauges["lxp_bridge_last_frame_age_s"], 0.001)

	obs := &observer{c: c, tags: []string{"deploymentID:x"}}
	obs.Frame("crc_error")
	obs.Frame("crc_error")
	obs.Reconnect()
	obs.CommandDropped()
	assert.Equal(t, int64(2), sink.counts["lxp_bridge_frames"])
	assert.Equal(t, int64(1), sink.counts["lxp_bridge_reconnects"])
	assert.Equal(t, int64(1), sink.counts["lxp_bridge_commands_dropped"])
}

func TestInvalidCRCModeRejected(t *testing.T) {
	_, err := New(models.Config{Inverters: []models.LXPConfig{{Enabled: true, DataLog: "A", CRCMode: "bogus"}}})
	assert.Error(t, err)
}

func TestCloseIsIdempotentWithoutStart(t *testing.T) {
	c, _, _ := newTest(t, []models.LXPConfig{{Enabled: true, DataLog: datalog}}, nil)
	c.Close()
	c.Close()
}

func TestFieldWriteReadModifyWrite(t *testing.T) {
	cfg := []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial}}
	topic := func(reg, field string) string { return "lxp/cmd/" + datalog + "/set/hold/" + reg + "/" + field }
	read := func(reg uint16) []byte {
		return modbus.BuildPacket(datalog, serial, 0, 0x3, reg, modbus.GetBytesForUInt16(1)...)
	}
	write := func(reg, val uint16) []byte {
		return modbus.BuildPacket(datalog, serial, 1, 0x6, reg, modbus.GetBytesForUInt16(val)...)
	}
	const reg21 = 25429 // live value: AC charge (bit 7) off

	t.Run("switch on reads then writes one bit", func(t *testing.T) {
		c, fakes, _ := newTest(t, cfg, nil)
		c.HandleCommand(topic("21", "ac_charge_enable"), []byte("ON"))
		require.Equal(t, [][]byte{read(21)}, fakes[datalog].sent())
		c.ReportHolding(datalog, 21, []uint16{reg21})
		assert.Equal(t, [][]byte{read(21), write(21, reg21|1<<7)}, fakes[datalog].sent())
	})
	t.Run("two fields of one register are merged", func(t *testing.T) {
		c, fakes, _ := newTest(t, cfg, nil)
		c.HandleCommand(topic("21", "ac_charge_enable"), []byte("1"))
		c.HandleCommand(topic("21", "EPS_ENABLE"), []byte("0"))
		c.ReportHolding(datalog, 20, []uint16{7, reg21, 0})
		sent := fakes[datalog].sent()
		assert.Equal(t, write(21, (reg21|1<<7)&^1), sent[len(sent)-1])
	})
	t.Run("no write when already set", func(t *testing.T) {
		c, fakes, _ := newTest(t, cfg, nil)
		c.HandleCommand(topic("179", "gen_peak_shaving_enable"), []byte("1"))
		c.ReportHolding(datalog, 179, []uint16{4416}) // bit 8 already on
		assert.Equal(t, [][]byte{read(179)}, fakes[datalog].sent())
	})
	t.Run("multi-bit field", func(t *testing.T) {
		c, fakes, _ := newTest(t, cfg, nil)
		c.HandleCommand(topic("120", "ac_charge_type"), []byte("2"))
		c.ReportHolding(datalog, 120, []uint16{0x2 | 0x80}) // type 1, bit 7 set
		sent := fakes[datalog].sent()
		assert.Equal(t, write(120, 0x4|0x80), sent[len(sent)-1])
	})
	t.Run("expired change is dropped", func(t *testing.T) {
		now := time.Unix(1000, 0)
		c, fakes, _ := newTest(t, cfg, func() time.Time { return now })
		c.HandleCommand(topic("21", "ac_charge_enable"), []byte("1"))
		now = now.Add(fieldWriteTimeout + time.Second)
		c.ReportHolding(datalog, 21, []uint16{reg21})
		assert.Equal(t, [][]byte{read(21)}, fakes[datalog].sent())
	})
	t.Run("only the next read applies the change", func(t *testing.T) {
		c, fakes, _ := newTest(t, cfg, nil)
		c.HandleCommand(topic("21", "ac_charge_enable"), []byte("1"))
		c.ReportHolding(datalog, 21, []uint16{reg21})
		c.ReportHolding(datalog, 21, []uint16{reg21})
		assert.Len(t, fakes[datalog].sent(), 2)
	})
	for name, tc := range map[string]struct{ topic, payload string }{
		"unknown field":      {topic("21", "nope"), "1"},
		"field of wrong reg": {topic("179", "ac_charge_enable"), "1"},
		"switch value 2":     {topic("21", "ac_charge_enable"), "2"},
		"switch garbage":     {topic("21", "ac_charge_enable"), "maybe"},
		"ac charge type 3":   {topic("120", "ac_charge_type"), "3"},
		"field on value reg": {topic("196", "gen_chg_start_soc"), "1"},
		"too many segments":  {topic("21", "ac_charge_enable") + "/x", "1"},
	} {
		t.Run(name, func(t *testing.T) {
			c, fakes, _ := newTest(t, cfg, nil)
			c.HandleCommand(tc.topic, []byte(tc.payload))
			assert.Empty(t, fakes[datalog].sent())
		})
	}
}

func TestHoldRegisterDefinitions(t *testing.T) {
	names := map[string]bool{}
	used := map[int]uint16{}
	for hr, r := range registers.HoldRegisters {
		addr, ok := registers.HoldRegisterToIDMap[hr]
		require.True(t, ok, "%s has no address", r.ShortName)
		assert.False(t, names[r.ShortName], "duplicate %s", r.ShortName)
		names[r.ShortName] = true
		if r.BitMask == 0 {
			if r.Writeable {
				assert.Equal(t, hr, registers.HoldIDToRegisterMap[addr], "%s", r.ShortName)
			}
			continue
		}
		_, whole := registers.HoldIDToRegisterMap[addr]
		assert.False(t, whole, "%d is both a value register and has bit fields", addr)
		assert.Zero(t, used[addr]&r.BitMask, "%s overlaps another field", r.ShortName)
		used[addr] |= r.BitMask
		max := r.BitMask >> bits.TrailingZeros16(r.BitMask)
		assert.LessOrEqual(t, r.Max, int(max), "%s max does not fit its bits", r.ShortName)
	}
}

func TestLongReadDefault(t *testing.T) {
	c, _, _ := newTest(t, []models.LXPConfig{
		{Enabled: true, DataLog: "A", Serial: serial},
		{Enabled: true, DataLog: "B", Serial: serial, LongRead: 10 * time.Second},
		{Enabled: true, DataLog: "C", Serial: serial, LongRead: -1 * time.Second},
	}, nil)
	assert.Equal(t, 30*time.Second, c.inverters["A"].config.LongRead)
	assert.Equal(t, 10*time.Second, c.inverters["B"].config.LongRead)
	assert.Equal(t, -1*time.Second, c.inverters["C"].config.LongRead)
}
