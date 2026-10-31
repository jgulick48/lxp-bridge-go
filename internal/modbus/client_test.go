package modbus

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFrame(t *testing.T) {
	good := clone(holdVectors["hold40"])
	garbage := []byte{0x00, 0xA1, 0x00, 0x1A, 0xA1, 0xA1}
	// Valid magic but absurd length (5000) -> must be rejected and resynced.
	oversized := []byte{0xA1, 0x1A, 5, 0, 0x88, 0x13, 1, 2, 3}

	tests := []struct {
		name        string
		input       []byte
		wantFrames  int
		wantSkipped int
	}{
		{"good frame", good, 1, 0},
		{"garbage then frame", append(clone(garbage), good...), 1, len(garbage)},
		{"oversized length rejected then frame", append(clone(oversized), good...), 1, len(oversized)},
		{"two frames back to back", append(clone(good), good...), 2, 0},
		{"only garbage", garbage, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(tt.input))
			frames, skipped := 0, 0
			for {
				f, s, err := readFrame(r)
				skipped += s
				if err != nil {
					break
				}
				frames++
				assert.Equal(t, good, f)
			}
			assert.Equal(t, tt.wantFrames, frames)
			if tt.wantFrames > 0 {
				assert.Equal(t, tt.wantSkipped, skipped)
			}
		})
	}
}

func TestReadFrameSplitReads(t *testing.T) {
	good := clone(holdVectors["hold40"])
	server, clientConn := net.Pipe()
	defer server.Close()
	defer clientConn.Close()
	go func() {
		// Dribble the frame out in awkward pieces, with garbage in front.
		stream := append([]byte{1, 2, 3}, good...)
		for i := 0; i < len(stream); i += 5 {
			end := i + 5
			if end > len(stream) {
				end = len(stream)
			}
			_, _ = server.Write(stream[i:end])
		}
	}()
	_ = clientConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	f, skipped, err := readFrame(bufio.NewReader(clientConn))
	require.NoError(t, err)
	assert.Equal(t, 3, skipped)
	assert.Equal(t, good, f)
}

func TestReadFrameEOFMidFrame(t *testing.T) {
	good := holdVectors["hold40"]
	_, _, err := readFrame(bufio.NewReader(bytes.NewReader(good[:50])))
	assert.Error(t, err)
}

type testObserver struct {
	reconnects, dropped atomic.Int64
	mu                  sync.Mutex
	frames              map[string]int
}

func (o *testObserver) Reconnect()      { o.reconnects.Add(1) }
func (o *testObserver) CommandDropped() { o.dropped.Add(1) }
func (o *testObserver) Frame(r string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.frames == nil {
		o.frames = map[string]int{}
	}
	o.frames[r]++
}
func (o *testObserver) count(r string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.frames[r]
}

func eventually(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", msg)
}

// fastClient builds a client pointed at addr with test-friendly timings.
func fastClient(t *testing.T, addr string, cb ParserCallback, obs Observer) *client {
	t.Helper()
	host, port, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	c := newClient(models.LXPConfig{Host: host, Port: port, DataLog: "BA31101550", ReadTimeout: 5 * time.Second}, cb, obs, CRCModeLog)
	c.backoffMin = 5 * time.Millisecond
	c.backoffMax = 20 * time.Millisecond
	c.sendSpacing = time.Millisecond
	t.Cleanup(c.Close)
	return c
}

// TestReconnect covers both failure styles: the server closing cleanly (EOF)
// and the server resetting the connection (read error).
func TestReconnect(t *testing.T) {
	for _, name := range []string{"eof", "reset"} {
		t.Run(name, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer ln.Close()

			var accepts atomic.Int64
			received := make(chan []byte, 16)
			go func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						return
					}
					n := accepts.Add(1)
					go func(conn net.Conn, n int64) {
						defer conn.Close()
						if n == 1 {
							// First connection: send a frame, then drop it.
							_, _ = conn.Write(holdVectors["hold160"])
							time.Sleep(20 * time.Millisecond)
							if name == "reset" {
								_ = conn.(*net.TCPConn).SetLinger(0) // close sends RST
							}
							return
						}
						_, _ = conn.Write(holdVectors["hold160"])
						buf := make([]byte, 4096)
						for {
							k, err := conn.Read(buf)
							if err != nil {
								return
							}
							received <- clone(buf[:k])
						}
					}(conn, n)
				}
			}()

			obs := &testObserver{}
			c := fastClient(t, ln.Addr().String(), newRecorder(), obs)
			c.Connect()

			eventually(t, func() bool { return obs.count("ok") >= 1 }, "first frame")
			// A poller hammering SendCommand across the drop must never panic or block.
			cmd := BuildPacket("BA31101550", "3192670065", 0, 0x4, 0, GetBytesForUInt16(40)...)
			done := make(chan struct{})
			go func() {
				defer close(done)
				for i := 0; i < 200; i++ {
					c.SendCommand(cmd)
					time.Sleep(time.Millisecond)
				}
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("SendCommand blocked")
			}
			eventually(t, func() bool { return accepts.Load() >= 2 && c.Connected() }, "reconnect")
			eventually(t, func() bool { return obs.reconnects.Load() >= 1 }, "reconnect observed")
			select {
			case got := <-received:
				assert.Equal(t, cmd[:2], got[:2])
			case <-time.After(10 * time.Second):
				t.Fatal("no command delivered after reconnect")
			}
			assert.False(t, c.LastFrame().IsZero())
		})
	}
}

func TestCommandQueuedWhileDisconnectedIsDelivered(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close()) // nothing listening: dials fail

	obs := &testObserver{}
	c := fastClient(t, addr, newRecorder(), obs)
	c.Connect()

	cmd := BuildPacket("BA31101550", "3192670065", 1, 0x6, 196, GetBytesForUInt16(20)...)
	c.SendCommand(cmd) // queued while disconnected
	time.Sleep(30 * time.Millisecond)
	assert.False(t, c.Connected())

	ln, err = net.Listen("tcp", addr)
	require.NoError(t, err)
	defer ln.Close()
	got := make(chan []byte, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, len(cmd))
		if _, err := io.ReadFull(conn, buf); err == nil {
			got <- buf
		}
	}()
	select {
	case b := <-got:
		assert.Equal(t, cmd, b)
	case <-time.After(10 * time.Second):
		t.Fatal("queued command not delivered after server came up")
	}
}

func TestSendCommandDropsWhenFull(t *testing.T) {
	obs := &testObserver{}
	c := newClient(models.LXPConfig{Host: "127.0.0.1", Port: "1"}, newRecorder(), obs, CRCModeLog)
	// Not connected: nothing drains the queue.
	for i := 0; i < commandQueue+5; i++ {
		c.SendCommand([]byte{1})
	}
	assert.Equal(t, int64(5), obs.dropped.Load())
	c.Close()
	c.SendCommand([]byte{1}) // after close: no panic, no block
}

func TestCloseIsIdempotentAndFast(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			defer conn.Close() // hold open, silent
		}
	}()
	c := fastClient(t, ln.Addr().String(), newRecorder(), nil)
	c.Connect()
	eventually(t, c.Connected, "connected")
	start := time.Now()
	c.Close()
	c.Close()
	assert.Less(t, time.Since(start), 2*time.Second)
	assert.False(t, c.Connected())

	// Close without Connect must not block either.
	c2 := newClient(models.LXPConfig{}, newRecorder(), nil, CRCModeLog)
	c2.Close()
	c2.Close()
}

func TestBackoffGrowsAndCaps(t *testing.T) {
	d := time.Second
	var seq []time.Duration
	for i := 0; i < 8; i++ {
		seq = append(seq, d)
		d = nextBackoff(d, 30*time.Second)
	}
	assert.Equal(t, []time.Duration{1, 2, 4, 8, 16, 30, 30, 30}, func() []time.Duration {
		out := make([]time.Duration, len(seq))
		for i, s := range seq {
			out[i] = s / time.Second
		}
		return out
	}())
}

func TestFrameResultsByCRCMode(t *testing.T) {
	corrupt := clone(holdVectors["hold160"])
	corrupt[40] ^= 0xFF
	tests := []struct {
		name         string
		mode         CRCMode
		frame        []byte
		wantResult   string
		wantLastTime bool
	}{
		{"log/valid", CRCModeLog, holdVectors["hold160"], "ok", true},
		{"log/corrupt still decoded", CRCModeLog, corrupt, "crc_error", true},
		{"drop/valid", CRCModeDrop, holdVectors["hold160"], "ok", true},
		{"drop/corrupt discarded", CRCModeDrop, corrupt, "crc_error", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &testObserver{}
			c := newClient(models.LXPConfig{}, newRecorder(), obs, tt.mode)
			c.handleFrame(tt.frame)
			assert.Equal(t, 1, obs.count(tt.wantResult))
			assert.Equal(t, tt.wantLastTime, !c.LastFrame().IsZero())
		})
	}
}

func TestParseErrorDoesNotRefreshLastFrame(t *testing.T) {
	obs := &testObserver{}
	c := newClient(models.LXPConfig{}, newRecorder(), obs, CRCModeLog)
	c.handleFrame([]byte{0xA1, 0x1A, 0, 0, 1, 0, 1})
	assert.Equal(t, 1, obs.count("parse_error"))
	assert.True(t, c.LastFrame().IsZero())
}
