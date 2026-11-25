package modbus

import (
	"sync"
	"testing"

	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
)

type rawRecorder struct {
	BaseParserCallback
	mu  sync.Mutex
	got []string
}

func (r *rawRecorder) ReportRawFrame(dl string, fn byte, reg uint16, frame []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, dl)
	_ = fn
	_ = reg
	_ = frame
}

func TestReadKey(t *testing.T) {
	req := BuildPacket("BA31101550", "3192670065", 0, 4, 40, GetBytesForUInt16(40)...)
	fn, reg, ok := ReadKey(req)
	assert.True(t, ok)
	assert.Equal(t, byte(4), fn)
	assert.Equal(t, uint16(40), reg)
	assert.Equal(t, "BA31101550", FrameDataLogger(req))
	_, _, ok = ReadKey(BuildPacket("BA31101550", "3192670065", 1, 6, 21, GetBytesForUInt16(1)...))
	assert.False(t, ok, "a write isn't a read")
	_, _, ok = ReadKey([]byte{0xa1, 0x1a, 1, 0, 0, 0, 1, 0xc1})
	assert.False(t, ok, "a heartbeat isn't a read")
}

func TestHandleFrameReportsRawReads(t *testing.T) {
	rec := &rawRecorder{}
	c := newClient(testConfig(), rec, nopObserver{}, CRCModeDrop)
	good := BuildPacket("BA31101550", "3192670065", 1, 4, 40, append([]byte{80}, make([]byte, 80)...)...)
	c.handleFrame(good)
	bad := append([]byte(nil), good...)
	bad[len(bad)-3] ^= 0xff
	c.handleFrame(bad) // dropped by its CRC: not reported
	c.handleFrame(BuildPacket("BA31101550", "3192670065", 1, 6, 21, GetBytesForUInt16(1)...))
	assert.Equal(t, []string{"BA31101550"}, rec.got)
}

func testConfig() models.LXPConfig {
	return models.LXPConfig{Host: "127.0.0.1", Port: "1", DataLog: "BA31101550"}
}
