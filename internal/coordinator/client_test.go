package coordinator

import (
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
		{"max gen chg current (not Writeable)", topic("198"), "60", nil}, // not Writeable
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
		{"non-writeable number (charge power percent)", topic("64"), "50", nil},
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
