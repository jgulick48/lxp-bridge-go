package mirror

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	datalog = "BA31101550"
	serial  = "3192670065"
)

// reply builds a read reply frame as a dongle sends it (byte count, then the values).
func reply(dl string, function byte, start uint16, vals []uint16) []byte {
	data := []byte{byte(len(vals) * 2)}
	for _, v := range vals {
		data = append(data, modbus.GetBytesForUInt16(v)...)
	}
	return modbus.BuildPacket(dl, serial, 1, function, start, data...)
}

// request builds a client's read request for count registers from start.
func request(function byte, start, count uint16) []byte {
	return modbus.BuildPacket(datalog, serial, 0, function, start, modbus.GetBytesForUInt16(count)...)
}

func msg(t time.Time, frame []byte) []byte {
	b, _ := json.Marshal(map[string]any{"t": t.UnixMilli(), "frame": hex.EncodeToString(frame)})
	return b
}

func newMirror(t *testing.T, now *time.Time) *Mirror {
	t.Helper()
	m, err := New(models.MirrorConfig{Enabled: true, DataLog: datalog, MQTT: models.MQTTConfig{Host: "broker"}})
	require.NoError(t, err)
	m.now = func() time.Time { return *now }
	return m
}

func TestStoreAndReply(t *testing.T) {
	now := time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC)
	m := newMirror(t, &now)
	assert.Equal(t, "lxp/"+datalog+"/raw/+/+", m.Topic())

	in := reply(datalog, 4, 0, make([]uint16, 40))
	hold := reply(datalog, 3, 0, make([]uint16, 40))
	require.NoError(t, m.Store("lxp/"+datalog+"/raw/4/0", msg(now, in)))
	require.NoError(t, m.Store("lxp/"+datalog+"/raw/3/0", msg(now, hold)))
	assert.Equal(t, in, m.Reply(request(4, 0, 40)))
	assert.Equal(t, hold, m.Reply(request(3, 0, 40)))
	assert.Nil(t, m.Reply(request(4, 40, 40)), "nothing kept for that block")
	assert.True(t, m.Fresh())

	now = now.Add(3 * time.Minute) // the source bridge stopped getting data
	assert.Nil(t, m.Reply(request(4, 0, 40)), "input frames older than 2m aren't served")
	assert.Equal(t, hold, m.Reply(request(3, 0, 40)), "settings are kept for a day")
	assert.False(t, m.Fresh())
	code, why := m.Health()
	assert.Equal(t, 503, code)
	assert.NotEmpty(t, why)

	// a retained message older than what's kept doesn't replace it
	newer := reply(datalog, 4, 0, append(make([]uint16, 39), 7))
	require.NoError(t, m.Store("lxp/"+datalog+"/raw/4/0", msg(now, newer)))
	assert.Error(t, m.Store("lxp/"+datalog+"/raw/4/0", msg(now.Add(-time.Hour), in)))
	assert.Equal(t, newer, m.Reply(request(4, 0, 40)))
	assert.EqualValues(t, 1, m.Stats()["stale"])
}

func TestStoreRejects(t *testing.T) {
	now := time.Now()
	m := newMirror(t, &now)
	good := reply(datalog, 4, 40, make([]uint16, 40))
	bad := append([]byte(nil), good...)
	bad[len(bad)-3] ^= 0xff // corrupt a value: CRC fails
	for name, c := range map[string]struct {
		topic   string
		payload []byte
	}{
		"not json":            {"lxp/" + datalog + "/raw/4/40", []byte("hello")},
		"not hex":             {"lxp/" + datalog + "/raw/4/40", []byte(`{"t":1,"frame":"zz"}`)},
		"bad crc":             {"lxp/" + datalog + "/raw/4/40", msg(now, bad)},
		"other datalogger":    {"lxp/" + datalog + "/raw/4/40", msg(now, reply("BA00000000", 4, 40, make([]uint16, 40)))},
		"topic doesn't match": {"lxp/" + datalog + "/raw/4/0", msg(now, good)},
		"not a raw topic":     {"lxp/" + datalog + "/input/p_load", msg(now, good)},
	} {
		assert.Error(t, m.Store(c.topic, c.payload), name)
	}
	assert.EqualValues(t, 0, m.Stats()["received"])
}

func TestServeOverTCP(t *testing.T) {
	now := time.Now()
	m := newMirror(t, &now)
	in := reply(datalog, 4, 0, make([]uint16, 40))
	require.NoError(t, m.Store("lxp/"+datalog+"/raw/4/0", msg(now, in)))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Serve(ctx, ln) }()
	defer ln.Close()

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer conn.Close()
	r := bufio.NewReader(conn)

	// a write is dropped (no reply), then a read is answered with the kept frame
	_, err = conn.Write(modbus.BuildPacket(datalog, serial, 1, 6, 21, modbus.GetBytesForUInt16(1)...))
	require.NoError(t, err)
	_, err = conn.Write(request(4, 0, 40))
	require.NoError(t, err)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	got, err := modbus.ReadFrame(r)
	require.NoError(t, err)
	assert.Equal(t, in, got)
	assert.Eventually(t, func() bool { return m.Stats()["writes_dropped"] == 1 }, time.Second, 10*time.Millisecond)

	// a block it hasn't got: no answer
	_, err = conn.Write(request(4, 80, 40))
	require.NoError(t, err)
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	_, err = modbus.ReadFrame(r)
	assert.Error(t, err, "no reply for an unknown block")
	assert.EqualValues(t, 1, m.Stats()["served"])
}

func TestNewChecksAndDefaults(t *testing.T) {
	_, err := New(models.MirrorConfig{MQTT: models.MQTTConfig{Host: "b"}})
	assert.Error(t, err)
	_, err = New(models.MirrorConfig{DataLog: datalog})
	assert.Error(t, err)
	m, err := New(models.MirrorConfig{DataLog: datalog, MQTT: models.MQTTConfig{Host: "b"}})
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0:8000", m.cfg.Listen)
	assert.Equal(t, 2*time.Minute, m.cfg.InputMaxAge)
	assert.Equal(t, 24*time.Hour, m.cfg.HoldMaxAge)
	assert.Equal(t, 4, m.cfg.MaxClients)
}
