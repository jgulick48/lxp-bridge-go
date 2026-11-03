package modbus

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/sirupsen/logrus"
)

const (
	maxFrameBody  = 2048
	commandQueue  = 32
	closeWait     = 3 * time.Second
	frameOK       = "ok"
	frameCRCError = "crc_error"
	frameParseErr = "parse_error"
)

// Client talks to a single LuxPower datalogger.
type Client interface {
	// Connect starts the connection loop. It does not block.
	Connect()
	// SendCommand queues a packet for sending. It never blocks and never
	// panics; if the queue is full the command is dropped.
	SendCommand(command []byte)
	// Close stops the client. It is idempotent.
	Close()
	// LastFrame is the time the last valid frame was received (zero if none).
	LastFrame() time.Time
	// Connected reports whether a TCP connection is currently established.
	Connected() bool
}

// Observer receives events used for metrics. All methods must be cheap and non-blocking.
type Observer interface {
	Reconnect()
	CommandDropped()
	// Frame is called for every framed packet with result ok, crc_error or parse_error.
	Frame(result string)
}

type nopObserver struct{}

func (nopObserver) Reconnect()      {}
func (nopObserver) CommandDropped() {}
func (nopObserver) Frame(string)    {}

type client struct {
	config   models.LXPConfig
	log      *logrus.Entry
	callBack ParserCallback
	obs      Observer
	crcMode  CRCMode

	commands chan []byte
	ctx      context.Context
	cancel   context.CancelFunc
	started  sync.Once
	stopped  chan struct{}
	isStart  atomic.Bool

	connected atomic.Bool
	lastFrame atomic.Int64 // unix nanoseconds

	// tunables, overridden by tests before Connect.
	dial         func(ctx context.Context, addr string) (net.Conn, error)
	dialTimeout  time.Duration
	backoffMin   time.Duration
	backoffMax   time.Duration
	sendSpacing  time.Duration
	writeTimeout time.Duration
	readTimeout  time.Duration
}

// NewClient builds a client; call Connect to start it. obs may be nil.
func NewClient(config models.LXPConfig, callback ParserCallback, obs Observer) (Client, error) {
	mode, err := ParseCRCMode(config.CRCMode)
	if err != nil {
		return nil, err
	}
	return newClient(config, callback, obs, mode), nil
}

func newClient(config models.LXPConfig, callback ParserCallback, obs Observer, mode CRCMode) *client {
	if obs == nil {
		obs = nopObserver{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &client{
		config:       config,
		log:          logrus.WithField("datalog", config.DataLog),
		callBack:     callback,
		obs:          obs,
		crcMode:      mode,
		commands:     make(chan []byte, commandQueue),
		ctx:          ctx,
		cancel:       cancel,
		stopped:      make(chan struct{}),
		dialTimeout:  10 * time.Second,
		backoffMin:   time.Second,
		backoffMax:   30 * time.Second,
		sendSpacing:  time.Second,
		writeTimeout: 10 * time.Second,
		readTimeout:  config.ReadTimeout,
	}
	if c.readTimeout <= 0 {
		c.readTimeout = 300 * time.Second
	}
	c.dial = func(ctx context.Context, addr string) (net.Conn, error) {
		d := net.Dialer{Timeout: c.dialTimeout, KeepAlive: 30 * time.Second}
		return d.DialContext(ctx, "tcp", addr)
	}
	return c
}

func (c *client) Connect() {
	c.started.Do(func() {
		c.isStart.Store(true)
		go c.run()
	})
}

func (c *client) Connected() bool { return c.connected.Load() }

func (c *client) LastFrame() time.Time {
	n := c.lastFrame.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}

// Close stops the connection loop and waits briefly for it to exit.
func (c *client) Close() {
	c.cancel()
	if !c.isStart.Load() {
		return
	}
	select {
	case <-c.stopped:
	case <-time.After(closeWait):
		c.log.Warn("Timed out waiting for inverter client to stop")
	}
}

func (c *client) SendCommand(command []byte) {
	if c.ctx.Err() != nil {
		return
	}
	select {
	case c.commands <- command:
	default:
		c.obs.CommandDropped()
		c.log.Warnf("Command queue full (%d), dropping command", commandQueue)
	}
}

// sleep waits for d or until the client is closed; it returns false if closed.
func (c *client) sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// run owns the connection lifecycle.
func (c *client) run() {
	defer close(c.stopped)
	addr := net.JoinHostPort(c.config.Host, c.config.Port)
	backoff := c.backoffMin
	everConnected := false
	for c.ctx.Err() == nil {
		c.log.Infof("Connecting to %s", addr)
		conn, err := c.dial(c.ctx, addr)
		if err != nil {
			if c.ctx.Err() != nil {
				return
			}
			c.log.Warnf("Error connecting to %s: %s (retrying in %s)", addr, err, backoff)
			if !c.sleep(c.ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff, c.backoffMax)
			continue
		}
		if everConnected {
			c.obs.Reconnect()
		}
		everConnected = true
		c.log.Infof("Connected to %s", addr)
		c.connected.Store(true)
		gotFrame, err := c.serve(conn)
		c.connected.Store(false)
		if c.ctx.Err() != nil {
			return
		}
		c.log.Warnf("Connection lost: %v", err)
		if gotFrame {
			backoff = c.backoffMin
		}
		if !c.sleep(c.ctx, backoff) {
			return
		}
		backoff = nextBackoff(backoff, c.backoffMax)
	}
}

func nextBackoff(cur, max time.Duration) time.Duration {
	cur *= 2
	if cur > max {
		cur = max
	}
	return cur
}

// serve runs reader and writer for one connection and returns when it fails.
func (c *client) serve(conn net.Conn) (gotFrame bool, err error) {
	connCtx, cancel := context.WithCancel(c.ctx)
	stopClose := context.AfterFunc(connCtx, func() { _ = conn.Close() })
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.writeLoop(connCtx, conn, cancel)
	}()
	gotFrame, err = c.readLoop(conn)
	cancel()
	stopClose()
	_ = conn.Close()
	wg.Wait()
	return gotFrame, err
}

func (c *client) writeLoop(ctx context.Context, conn net.Conn, fail context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-c.commands:
			_ = conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if _, err := conn.Write(cmd); err != nil {
				if ctx.Err() == nil {
					c.log.Errorf("Error sending command: %s", err)
				}
				fail() // reader will notice the closed conn and reconnect
				return
			}
			if !c.sleep(ctx, c.sendSpacing) {
				return
			}
		}
	}
}

func (c *client) readLoop(conn net.Conn) (gotFrame bool, err error) {
	reader := bufio.NewReaderSize(conn, 4096)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		frame, skipped, err := readFrame(reader)
		if skipped > 0 {
			c.log.Debugf("Skipped %d bytes of garbage while resyncing", skipped)
		}
		if err != nil {
			if c.ctx.Err() != nil {
				return gotFrame, c.ctx.Err()
			}
			return gotFrame, fmt.Errorf("reading frame: %w", err)
		}
		gotFrame = true
		c.handleFrame(frame)
	}
}

func (c *client) handleFrame(frame []byte) {
	packet, err := DecodeWithMode(frame, c.callBack, c.crcMode)
	switch {
	case errors.Is(err, ErrCRCMismatch):
		c.obs.Frame(frameCRCError)
		c.log.Debug("Dropped frame with CRC mismatch")
	case err != nil:
		c.obs.Frame(frameParseErr)
		c.log.Debugf("Error decoding frame: %s", err)
	default:
		c.lastFrame.Store(time.Now().UnixNano())
		if packet != nil && packet.CRCMismatch {
			c.obs.Frame(frameCRCError)
		} else {
			c.obs.Frame(frameOK)
		}
	}
}

var frameMagic = [2]byte{0xA1, 0x1A}

// readFrame reads one complete frame (6 byte header plus body). It scans
// forward past garbage until a plausible header is found. skipped reports how
// many bytes were discarded.
func readFrame(r *bufio.Reader) (frame []byte, skipped int, err error) {
	for {
		hdr, err := r.Peek(frameHeaderLen)
		if err != nil {
			return nil, skipped, err
		}
		if hdr[0] == frameMagic[0] && hdr[1] == frameMagic[1] &&
			int(binary.LittleEndian.Uint16(hdr[4:6])) <= maxFrameBody {
			n := frameHeaderLen + int(binary.LittleEndian.Uint16(hdr[4:6]))
			frame = make([]byte, n)
			if _, err := io.ReadFull(r, frame); err != nil {
				return nil, skipped, err
			}
			return frame, skipped, nil
		}
		// Not a header: drop one byte and look again.
		if _, err := r.Discard(1); err != nil {
			return nil, skipped, err
		}
		skipped++
	}
}
