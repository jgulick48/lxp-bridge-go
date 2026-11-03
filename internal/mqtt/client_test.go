package mqtt

import (
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDisabledClientIsSafeNoop(t *testing.T) {
	c := NewClient(models.MQTTConfig{}, false, &BaseMessageCallback{}, func() { t.Error("onConnect must not fire") })
	assert.False(t, c.IsEnabled())
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.Connect()
		c.SubMultiple([]string{"a/b"})
		assert.Error(t, c.SendMessage("t", "v", true))
		c.Close()
		c.Close()
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("disabled client blocked")
	}
}

type recordingCallback struct{ topics []string }

func (r *recordingCallback) HandleCommand(topic string, _ []byte) { r.topics = append(r.topics, topic) }

func TestCommandRouting(t *testing.T) {
	cb := &recordingCallback{}
	c := NewClient(models.MQTTConfig{Host: "x"}, false, cb, nil).(*client)
	c.ProcessData("lxp/cmd/DL/set/hold/196", []byte("20"))
	c.ProcessData("lxp/sensor/DL/read/power1", []byte("1"))
	c.ProcessData("lxp/cmd/short", []byte("1"))
	assert.Equal(t, []string{"lxp/cmd/DL/set/hold/196"}, cb.topics)
}

func TestUnreachableBrokerDoesNotPanicOrBlock(t *testing.T) {
	// Port 1 on localhost: connection refused. Connect must return immediately.
	c := NewClient(models.MQTTConfig{Host: "127.0.0.1", Port: 1, ClientName: "t"}, false, &BaseMessageCallback{}, nil)
	start := time.Now()
	c.Connect()
	c.SubMultiple([]string{"lxp/cmd/DL/set/#"})
	_ = c.SendMessage("t", "v", false)
	c.Close()
	assert.Less(t, time.Since(start), 5*time.Second)
}
