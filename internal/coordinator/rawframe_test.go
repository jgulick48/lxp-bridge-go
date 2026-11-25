package coordinator

import (
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type enabledRecMQTT struct{ recMQTT }

func (*enabledRecMQTT) IsEnabled() bool { return true }

func TestReportRawFrame(t *testing.T) {
	now := time.Date(2026, 11, 25, 6, 0, 0, 0, time.UTC)
	build1 := func(publish bool) (*client, *enabledRecMQTT) {
		m := &enabledRecMQTT{recMQTT{msgs: map[string]string{}}}
		c, err := build(models.Config{
			MQTTConfig: models.MQTTConfig{NameSpace: "lxp", PublishRawFrames: publish},
			Inverters:  []models.LXPConfig{{Enabled: true, DataLog: datalog, Serial: serial}},
		}, deps{mqtt: m, now: func() time.Time { return now }, newModbus: func(models.LXPConfig, modbus.ParserCallback, modbus.Observer) (modbus.Client, error) {
			return &fakeModbus{}, nil
		}})
		require.NoError(t, err)
		return c, m
	}
	frame := response(4, 40, make([]uint16, 40))

	c, m := build1(false)
	c.ReportRawFrame(datalog, 4, 40, frame)
	assert.Empty(t, m.msgs, "off unless mqtt.publish_raw_frames")

	c, m = build1(true)
	c.ReportRawFrame("BA00000000", 4, 40, frame)
	assert.Empty(t, m.msgs, "only configured dataloggers")
	c.ReportRawFrame(datalog, 4, 40, frame)
	topic := "lxp/" + datalog + "/raw/4/40"
	require.Contains(t, m.msgs, topic)
	var got RawFrame
	require.NoError(t, json.Unmarshal([]byte(m.msgs[topic]), &got))
	assert.Equal(t, now.UnixMilli(), got.Time)
	assert.Equal(t, hex.EncodeToString(frame), got.Frame)
	assert.Equal(t, topic, RawTopic("lxp", datalog, 4, 40))
}
