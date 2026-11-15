package coordinator

import (
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func denver(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)
	return loc
}

func TestClockEncodeDecode(t *testing.T) {
	loc := denver(t)
	want := time.Date(2026, 11, 14, 23, 53, 44, 0, loc)

	got, ok := decodeClock(0x0b1a, 0x170e, 0x2c35, loc)
	require.True(t, ok)
	assert.True(t, want.Equal(got))
	assert.Equal(t, "MST", got.Format("MST"))
	assert.Equal(t, [3]uint16{0x0b1a, 0x170e, 0x2c35}, encodeClock(want))

	// Round trip, including either side of both DST transitions (2026-03-08, 2026-11-01).
	for _, tm := range []time.Time{
		time.Date(2026, 3, 8, 1, 59, 59, 0, loc),
		time.Date(2026, 3, 8, 3, 0, 0, 0, loc),
		time.Date(2026, 11, 1, 0, 59, 59, 0, loc),
		time.Date(2026, 11, 1, 2, 0, 0, 0, loc),
		time.Date(2026, 7, 4, 12, 0, 0, 0, loc),
		time.Date(2000, 1, 1, 0, 0, 0, 0, loc),
	} {
		r := encodeClock(tm)
		back, ok := decodeClock(r[0], r[1], r[2], loc)
		require.True(t, ok, tm)
		assert.True(t, tm.Equal(back), tm)
	}
	// Encoding the same instant in UTC differs from local time.
	assert.NotEqual(t, encodeClock(want), encodeClock(want.UTC()))
}

func TestClockDecodeInvalid(t *testing.T) {
	loc := time.UTC
	pack := func(year, month, day, hour, minute, second int) (uint16, uint16, uint16) {
		return uint16(month<<8 | (year - 2000)), uint16(hour<<8 | day), uint16(second<<8 | minute)
	}
	for name, f := range map[string][6]int{
		"month 0":   {2026, 0, 14, 1, 1, 1},
		"month 13":  {2026, 13, 14, 1, 1, 1},
		"day 0":     {2026, 11, 0, 1, 1, 1},
		"day 32":    {2026, 11, 32, 1, 1, 1},
		"hour 24":   {2026, 11, 14, 24, 1, 1},
		"minute 60": {2026, 11, 14, 1, 60, 1},
		"second 60": {2026, 11, 14, 1, 1, 60},
		"feb 30":    {2026, 2, 30, 1, 1, 1},
		"feb 29 26": {2026, 2, 29, 1, 1, 1},
		"apr 31":    {2026, 4, 31, 1, 1, 1},
	} {
		a, b, c := pack(f[0], f[1], f[2], f[3], f[4], f[5])
		_, ok := decodeClock(a, b, c, loc)
		assert.False(t, ok, name)
	}
	a, b, c := pack(2028, 2, 29, 1, 1, 1)
	_, ok := decodeClock(a, b, c, loc)
	assert.True(t, ok, "leap day")
	_, ok = decodeClock(0, 0, 0, loc)
	assert.False(t, ok)
}

// clockHarness drives ReportHolding with a controllable host clock.
type clockHarness struct {
	t    *testing.T
	c    *client
	f    *fakeModbus
	sink *fakeSink
	loc  *time.Location
	now  time.Time
}

func newClockHarness(t *testing.T, cs models.ClockSyncConfig) *clockHarness {
	h := &clockHarness{t: t, loc: denver(t), now: time.Date(2026, 11, 15, 6, 53, 44, 0, time.UTC)} // 23:53:44 MST
	var fakes map[string]*fakeModbus
	h.c, fakes, h.sink = newTest(t, []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial, ClockSync: cs}}, func() time.Time { return h.now })
	h.f = fakes[datalog]
	return h
}

func (h *clockHarness) enabled() *clockHarness { return h }

// report feeds a block-0 read whose clock registers show host time + offset.
func (h *clockHarness) report(offset time.Duration) {
	r := encodeClock(h.now.Add(offset).In(h.loc))
	vals := make([]uint16, 40)
	vals[12], vals[13], vals[14] = r[0], r[1], r[2]
	h.c.ReportHolding(datalog, 0, vals)
}

func (h *clockHarness) wantSync() []byte {
	r := encodeClock(h.now.In(h.loc))
	return modbus.BuildWriteMulti(datalog, serial, 12, r[:])
}

func (h *clockHarness) driftGauge() (float64, bool) {
	h.sink.mu.Lock()
	defer h.sink.mu.Unlock()
	v, ok := h.sink.gauges["lxp_bridge_clock_drift_s"]
	return v, ok
}

var syncOn = models.ClockSyncConfig{Enabled: true, Timezone: "America/Denver"}

func TestClockSyncDisabledOnlyReportsDrift(t *testing.T) {
	h := newClockHarness(t, models.ClockSyncConfig{Timezone: "America/Denver"})
	h.report(-10 * time.Minute)
	assert.Empty(t, h.f.sent())
	v, ok := h.driftGauge()
	require.True(t, ok)
	assert.InDelta(t, -600, v, 0.001)
}

func TestClockNoTimezoneNoGauge(t *testing.T) {
	h := newClockHarness(t, models.ClockSyncConfig{})
	h.report(-10 * time.Minute)
	assert.Empty(t, h.f.sent())
	_, ok := h.driftGauge()
	assert.False(t, ok)
}

func TestClockWithinDriftNoWrite(t *testing.T) {
	h := newClockHarness(t, syncOn)
	h.report(45 * time.Second)
	assert.Empty(t, h.f.sent())
	v, ok := h.driftGauge()
	require.True(t, ok)
	assert.InDelta(t, 45, v, 0.001)

	h = newClockHarness(t, models.ClockSyncConfig{Enabled: true, Timezone: "America/Denver", MaxDrift: 5 * time.Minute})
	h.report(-4 * time.Minute)
	assert.Empty(t, h.f.sent())
}

func TestClockSyncWritesAndVerifies(t *testing.T) {
	h := newClockHarness(t, syncOn)
	h.report(-5 * time.Minute)
	sent := h.f.sent()
	require.Len(t, sent, 2)
	assert.Equal(t, h.wantSync(), sent[0])
	assert.Equal(t, modbus.BuildPacket(datalog, serial, 0, 0x3, 0, modbus.GetBytesForUInt16(40)...), sent[1])

	// The queued verification read answers about a second later and shows a
	// good clock: accepted straight away, no more writes.
	h.now = h.now.Add(time.Second)
	h.report(0)
	assert.Len(t, h.f.sent(), 2)
	assert.Equal(t, 0, h.c.inverters[datalog].clock.failures)
	v, _ := h.driftGauge()
	assert.InDelta(t, 0, v, 0.001)
}

func TestClockSyncRateLimited(t *testing.T) {
	h := newClockHarness(t, syncOn)
	h.report(-5 * time.Minute)
	require.Len(t, h.f.sent(), 2)

	// Still wrong 30 minutes later: counted as a failure but no new write.
	h.now = h.now.Add(30 * time.Minute)
	h.report(-5 * time.Minute)
	assert.Len(t, h.f.sent(), 2)
	assert.Equal(t, 1, h.c.inverters[datalog].clock.failures)

	// After an hour a second write goes out.
	h.now = h.now.Add(31 * time.Minute)
	h.report(-5 * time.Minute)
	assert.Len(t, h.f.sent(), 4)
}

func TestClockSyncGivesUpAfterThreeFailures(t *testing.T) {
	h := newClockHarness(t, syncOn)
	for i := 0; i < 3; i++ {
		h.report(-5 * time.Minute) // triggers write
		require.Len(t, h.f.sent(), 2*(i+1))
		h.now = h.now.Add(clockVerifyDelay)
		h.report(-5 * time.Minute) // failed verification
		h.now = h.now.Add(clockSyncInterval)
	}
	assert.True(t, h.c.inverters[datalog].clock.gaveUp)
	n := len(h.f.sent())
	h.report(-5 * time.Minute)
	assert.Len(t, h.f.sent(), n, "no more writes once given up")
}

func TestClockGoodReadResetsFailures(t *testing.T) {
	h := newClockHarness(t, syncOn)
	h.report(-5 * time.Minute)
	h.now = h.now.Add(clockVerifyDelay)
	h.report(-5 * time.Minute)
	require.Equal(t, 1, h.c.inverters[datalog].clock.failures)
	h.now = h.now.Add(time.Minute)
	h.report(time.Second)
	assert.Equal(t, 0, h.c.inverters[datalog].clock.failures)
}

func TestClockStaleReadRightAfterWriteIsNotAFailure(t *testing.T) {
	h := newClockHarness(t, syncOn)
	h.report(-5 * time.Minute)
	h.report(-5 * time.Minute) // another client's reply arriving immediately
	h.now = h.now.Add(clockVerifyDelay - time.Second)
	h.report(-5 * time.Minute) // still inside the window
	assert.Equal(t, 0, h.c.inverters[datalog].clock.failures)
	assert.True(t, h.c.inverters[datalog].clock.verifying)
	assert.Len(t, h.f.sent(), 2)
}

func TestClockInvalidTriggersSync(t *testing.T) {
	h := newClockHarness(t, syncOn)
	h.c.ReportHolding(datalog, 0, make([]uint16, 40)) // all zeros: month 0
	sent := h.f.sent()
	require.Len(t, sent, 2)
	assert.Equal(t, h.wantSync(), sent[0])
	_, ok := h.driftGauge()
	assert.False(t, ok, "no gauge for an invalid clock")
}

func TestClockBlockNotCoveringRegisters(t *testing.T) {
	h := newClockHarness(t, syncOn)
	h.c.ReportHolding(datalog, 40, make([]uint16, 40))
	h.c.ReportHolding(datalog, 0, make([]uint16, 14)) // registers 0-13
	h.c.ReportHolding(datalog, 14, make([]uint16, 20))
	h.c.ReportHolding(datalog, 12, make([]uint16, 2))
	assert.Empty(t, h.f.sent())
	_, ok := h.driftGauge()
	assert.False(t, ok)
}

func TestClockPartialBlockStartingAtClock(t *testing.T) {
	h := newClockHarness(t, syncOn)
	r := encodeClock(h.now.Add(-time.Hour).In(h.loc))
	h.c.ReportHolding(datalog, 12, r[:])
	require.Len(t, h.f.sent(), 2)
	assert.Equal(t, h.wantSync(), h.f.sent()[0])
}

func TestClockConfigValidation(t *testing.T) {
	for name, cs := range map[string]models.ClockSyncConfig{
		"missing": {Enabled: true},
		"invalid": {Enabled: true, Timezone: "Mars/Olympus"},
	} {
		_, err := build(models.Config{Inverters: []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial, ClockSync: cs}}}, deps{
			mqtt: fakeMQTT{},
			newModbus: func(models.LXPConfig, modbus.ParserCallback, modbus.Observer) (modbus.Client, error) {
				return &fakeModbus{}, nil
			},
		})
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), datalog, name)
		assert.Contains(t, err.Error(), "timezone", name)
	}
}
