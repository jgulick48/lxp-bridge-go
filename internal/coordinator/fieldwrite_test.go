package coordinator

import (
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFieldWriteRetries(t *testing.T) {
	cmd := "lxp/cmd/" + datalog + "/set/hold/21/ac_charge_enable"
	read := modbus.BuildPacket(datalog, serial, 0, 0x3, 21, modbus.GetBytesForUInt16(1)...)
	inverters := []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial}}
	start := time.Unix(1000, 0)

	setup := func(t *testing.T, connected bool) (*client, *inverter, *fakeModbus, *fakeSink, *time.Time) {
		now := start
		c, fakes, sink := newTest(t, inverters, func() time.Time { return now })
		fakes[datalog].connected = connected
		c.HandleCommand(cmd, []byte("1"))
		return c, c.inverters[datalog], fakes[datalog], sink, &now
	}

	t.Run("read is resent at 3s and 10s, at most 3 reads", func(t *testing.T) {
		c, inv, f, _, now := setup(t, true)
		assert.Len(t, f.sent(), 1)
		step := func(d time.Duration, want int) {
			*now = start.Add(d)
			c.checkFieldWrites(inv, *now)
			assert.Len(t, f.sent(), want, "at +%s", d)
		}
		step(2900*time.Millisecond, 1)
		step(3*time.Second, 2)
		step(9900*time.Millisecond, 2)
		step(10*time.Second, 3)
		step(25*time.Second, 3)
		for _, b := range f.sent() {
			assert.Equal(t, read, b)
		}
	})
	t.Run("no retry while disconnected", func(t *testing.T) {
		c, inv, f, _, now := setup(t, false)
		*now = start.Add(20 * time.Second)
		c.checkFieldWrites(inv, *now)
		assert.Len(t, f.sent(), 1)
	})
	t.Run("dropped at the deadline and counted", func(t *testing.T) {
		c, inv, _, sink, now := setup(t, true)
		*now = start.Add(fieldWriteTimeout)
		c.checkFieldWrites(inv, *now)
		assert.Len(t, inv.pending, 1)
		*now = start.Add(fieldWriteTimeout + time.Second)
		c.checkFieldWrites(inv, *now)
		assert.Empty(t, inv.pending)
		assert.Equal(t, int64(1), sink.counts["lxp_bridge_field_writes_dropped"])
	})
	t.Run("late drop in ReportHolding is counted", func(t *testing.T) {
		c, _, _, sink, now := setup(t, true)
		*now = start.Add(fieldWriteTimeout + time.Second)
		c.ReportHolding(datalog, 21, []uint16{25429})
		assert.Equal(t, int64(1), sink.counts["lxp_bridge_field_writes_dropped"])
	})
	t.Run("reply after a retry still writes the value", func(t *testing.T) {
		c, inv, f, _, now := setup(t, true)
		*now = start.Add(3 * time.Second)
		c.checkFieldWrites(inv, *now)
		c.ReportHolding(datalog, 21, []uint16{25429})
		sent := f.sent()
		want := modbus.BuildPacket(datalog, serial, 1, 0x6, 21, modbus.GetBytesForUInt16(25429|1<<7)...)
		assert.Equal(t, [][]byte{read, read, want}, sent)
		assert.Empty(t, inv.pending)
	})
	t.Run("read only sends nothing", func(t *testing.T) {
		now := start
		c, fakes, _ := newTest(t, inverters, func() time.Time { return now })
		c.config.ReadOnly = true
		fakes[datalog].connected = true
		c.HandleCommand(cmd, []byte("1"))
		c.inverters[datalog].pending[21] = &fieldWrite{mask: 1 << 7, value: 1 << 7, deadline: start.Add(fieldWriteTimeout)}
		now = start.Add(20 * time.Second)
		c.checkFieldWrites(c.inverters[datalog], now)
		assert.Empty(t, fakes[datalog].sent())
	})
}
