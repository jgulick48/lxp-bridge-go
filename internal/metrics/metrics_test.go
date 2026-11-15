package metrics

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type call struct {
	name  string
	value float64
	tags  []string
}

type recordingSink struct {
	Nop
	calls []call
}

func (r *recordingSink) Gauge(name string, value float64, tags []string, _ float64) error {
	r.calls = append(r.calls, call{name, value, tags})
	return nil
}

func TestGaugeResetsBeforeNegative(t *testing.T) {
	tags := []string{"deploymentID:abc"}
	cases := []struct {
		name  string
		value float64
		want  []call
	}{
		{"negative", -1185, []call{{"m", 0, tags}, {"m", -1185, tags}}},
		{"positive", 1185, []call{{"m", 1185, tags}}},
		{"zero", 0, []call{{"m", 0, tags}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &recordingSink{}
			Gauge(s, "m", tags, tc.value)
			assert.Equal(t, tc.want, s.calls)
		})
	}
}

// readLines collects statsd lines from conn until no packet arrives for a while.
func readLines(t *testing.T, conn net.PacketConn) []string {
	t.Helper()
	var lines []string
	buf := make([]byte, 65536)
	for {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(500*time.Millisecond)))
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			return lines
		}
		for _, l := range strings.Split(strings.TrimSpace(string(buf[:n])), "\n") {
			if l != "" {
				lines = append(lines, l)
			}
		}
	}
}

func TestGaugeWireFormat(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer conn.Close()

	sink, err := New(conn.LocalAddr().String())
	require.NoError(t, err)

	tags := []string{"deploymentID:abc"}
	// Duplicate readings within one flush must not accumulate.
	Gauge(sink, "lxp_p_battery", tags, -1185)
	Gauge(sink, "lxp_p_battery", tags, -1185)
	Gauge(sink, "lxp_p_pv", tags, 2500)
	Gauge(sink, "lxp_zero", tags, 0)
	require.NoError(t, sink.Close())

	lines := readLines(t, conn)
	var battery, pv, zero []string
	for _, l := range lines {
		switch {
		case strings.HasPrefix(l, "lxp_p_battery:"):
			battery = append(battery, l)
		case strings.HasPrefix(l, "lxp_p_pv:"):
			pv = append(pv, l)
		case strings.HasPrefix(l, "lxp_zero:"):
			zero = append(zero, l)
		}
	}
	assert.Equal(t, []string{
		"lxp_p_battery:0|g|#deploymentID:abc",
		"lxp_p_battery:-1185|g|#deploymentID:abc",
		"lxp_p_battery:0|g|#deploymentID:abc",
		"lxp_p_battery:-1185|g|#deploymentID:abc",
	}, battery)
	assert.Equal(t, []string{"lxp_p_pv:2500|g|#deploymentID:abc"}, pv)
	assert.Equal(t, []string{"lxp_zero:0|g|#deploymentID:abc"}, zero)
}
