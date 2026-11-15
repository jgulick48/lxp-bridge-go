package coordinator

import (
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/metrics"
	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/sirupsen/logrus"
	_ "time/tzdata" // zone database for minimal images without /usr/share/zoneinfo
)

const (
	// Holding registers 12-14 hold the inverter clock in local time.
	clockRegStart = 12
	clockRegCount = 3

	defaultClockMaxDrift = 60 * time.Second
	// clockSyncInterval is the minimum time between clock writes per inverter.
	clockSyncInterval = time.Hour
	// clockWarnInterval rate limits the invalid clock warning.
	clockWarnInterval = time.Hour
	// clockMaxFailures is how many consecutive failed verifications stop syncing.
	clockMaxFailures = 3
	// clockVerifyDelay ignores reads right after a write (stale broadcasts).
	clockVerifyDelay = 2 * time.Second
)

// clockState is the per inverter clock sync state, guarded by inverter.mu.
type clockState struct {
	loc        *time.Location // nil when no timezone is configured
	maxDrift   time.Duration
	lastWrite  time.Time
	verifying  bool
	failures   int
	gaveUp     bool
	lastWarned time.Time
}

// encodeClock encodes t (already in the inverter's local zone) into holding
// registers 12-14: month<<8|year-2000, hour<<8|day, second<<8|minute.
func encodeClock(t time.Time) [3]uint16 {
	return [3]uint16{
		uint16(t.Month())<<8 | uint16(uint8(t.Year()-2000)),
		uint16(t.Hour())<<8 | uint16(t.Day()),
		uint16(t.Second())<<8 | uint16(t.Minute()),
	}
}

// decodeClock decodes registers 12-14 as a wall clock time in loc. ok is
// false when a field is out of range or the date does not exist.
func decodeClock(r12, r13, r14 uint16, loc *time.Location) (time.Time, bool) {
	year := 2000 + int(r12&0xff)
	month, day := int(r12>>8), int(r13&0xff)
	hour, minute, second := int(r13>>8), int(r14&0xff), int(r14>>8)
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, false
	}
	t := time.Date(year, time.Month(month), day, hour, minute, second, 0, loc)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, false
	}
	return t, true
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// checkClock handles a holding read covering registers 12-14. It is called
// with inv.mu held, like the rest of ReportHolding. regs are registers 12-14.
func (c *client) checkClock(inv *inverter, now time.Time, regs []uint16) {
	cs := &inv.clock
	if cs.loc == nil {
		return // no timezone configured: nothing to compare against
	}
	dl := inv.config.DataLog
	invTime, valid := decodeClock(regs[0], regs[1], regs[2], cs.loc)
	var drift time.Duration
	if valid {
		drift = invTime.Sub(now)
		metrics.Gauge(c.metrics, c.metricName("bridge_clock_drift_s"), []string{metrics.FormatTag("deploymentID", dl)}, drift.Seconds())
	} else if cs.lastWarned.IsZero() || now.Sub(cs.lastWarned) >= clockWarnInterval {
		cs.lastWarned = now
		logrus.Warnf("Inverter %s reports an invalid clock (registers 12-14 = %#04x %#04x %#04x)", dl, regs[0], regs[1], regs[2])
	}
	inSync := valid && absDuration(drift) <= cs.maxDrift

	if cs.verifying && now.Sub(cs.lastWrite) >= clockVerifyDelay {
		cs.verifying = false
		if inSync {
			cs.failures = 0
			logrus.Infof("Inverter %s clock verified after sync, drift %s", dl, drift)
		} else {
			cs.failures++
			logrus.Warnf("Inverter %s clock still off after sync (drift %s), failed attempt %d of %d", dl, drift, cs.failures, clockMaxFailures)
			if cs.failures >= clockMaxFailures {
				cs.gaveUp = true
				logrus.Errorf("Inverter %s: giving up clock sync after %d failed attempts; function 0x10 (write multiple) may not be supported by the datalogger", dl, cs.failures)
			}
		}
	} else if inSync && !cs.verifying {
		cs.failures = 0
	}

	if !inv.config.ClockSync.Enabled || cs.gaveUp || inSync || cs.verifying {
		return
	}
	if !cs.lastWrite.IsZero() && now.Sub(cs.lastWrite) < clockSyncInterval {
		return
	}
	target := now.In(cs.loc)
	cs.lastWrite = now
	cs.verifying = true
	if valid {
		logrus.Infof("Syncing clock on %s: inverter %s -> %s (drift %s)", dl, invTime.Format(time.RFC3339), target.Format(time.RFC3339), drift)
	} else {
		logrus.Infof("Syncing invalid clock on %s -> %s", dl, target.Format(time.RFC3339))
	}
	vals := encodeClock(target)
	inv.client.SendCommand(modbus.BuildWriteMulti(dl, inv.config.Serial, clockRegStart, vals[:]))
	inv.client.SendCommand(modbus.BuildPacket(dl, inv.config.Serial, 0, 0x3, 0, modbus.GetBytesForUInt16(40)...))
}
