// Package mirror serves a dongle's data to other LuxPower clients without them connecting to the dongle.
//
// A bridge with mqtt.publish_raw_frames publishes every valid read reply it gets from the dongle, as received, to
// <namespace>/<datalog>/raw/<function>/<start register>. The mirror subscribes to those (read only: no will, no
// publishes), keeps the latest frame of each, and listens like a dongle: a client's read request for a block is
// answered with the latest frame for it, while it is fresh. Writes are dropped, so a mirror can never change the
// inverter. When the source bridge stops getting data, its frames go stale and the mirror stops answering, so its
// clients see the inverter offline too.
//
// It exists because a WiFi dongle copes badly with several clients polling it: it can go quiet for minutes or
// drop off WiFi.
package mirror

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/jgulick48/lxp-bridge-go/internal/mqtt"
	"github.com/sirupsen/logrus"
)

const (
	defaultListen       = "0.0.0.0:8000"
	defaultInputMaxAge  = 2 * time.Minute
	defaultHoldMaxAge   = 24 * time.Hour
	defaultMaxClients   = 4
	clientIdleTimeout   = 10 * time.Minute
	subscribeTimeout    = 10 * time.Second
	disconnectQuiesceMs = 250
)

type key struct {
	function byte
	register uint16
}

type entry struct {
	frame []byte
	at    time.Time
}

// Mirror is one dongle's mirror.
type Mirror struct {
	cfg models.MirrorConfig
	now func() time.Time

	mu     sync.Mutex
	frames map[key]entry

	clients  atomic.Int32
	served   atomic.Int64
	stale    atomic.Int64
	dropped  atomic.Int64
	received atomic.Int64
	mqttUp   atomic.Bool

	mqtt paho.Client
}

// New checks the settings and fills in defaults.
func New(cfg models.MirrorConfig) (*Mirror, error) {
	if cfg.DataLog == "" {
		return nil, errors.New("mirror.datalog is required")
	}
	if cfg.MQTT.Host == "" {
		return nil, errors.New("mirror.mqtt.host is required")
	}
	if cfg.MQTT.NameSpace == "" {
		cfg.MQTT.NameSpace = "lxp"
	}
	if cfg.Listen == "" {
		cfg.Listen = defaultListen
	}
	if cfg.InputMaxAge <= 0 {
		cfg.InputMaxAge = defaultInputMaxAge
	}
	if cfg.HoldMaxAge <= 0 {
		cfg.HoldMaxAge = defaultHoldMaxAge
	}
	if cfg.MaxClients <= 0 {
		cfg.MaxClients = defaultMaxClients
	}
	return &Mirror{cfg: cfg, now: time.Now, frames: map[key]entry{}}, nil
}

// Topic is the subscription: every raw frame of this datalogger.
func (m *Mirror) Topic() string {
	return fmt.Sprintf("%s/%s/raw/+/+", m.cfg.MQTT.NameSpace, m.cfg.DataLog)
}

// Store takes one raw frame message. It returns an error (and keeps nothing) for anything that isn't a valid read
// reply from this datalogger, or that is older than what it already has.
func (m *Mirror) Store(topic string, payload []byte) error {
	parts := strings.Split(topic, "/")
	if len(parts) < 4 || parts[len(parts)-3] != "raw" {
		return fmt.Errorf("not a raw frame topic: %s", topic)
	}
	var msg struct {
		Time  int64  `json:"t"`
		Frame string `json:"frame"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("%s: %w", topic, err)
	}
	frame, err := hex.DecodeString(msg.Frame)
	if err != nil {
		return fmt.Errorf("%s: frame isn't hex: %w", topic, err)
	}
	if _, err := modbus.DecodeWithMode(frame, &modbus.BaseParserCallback{}, modbus.CRCModeDrop); err != nil {
		return fmt.Errorf("%s: %w", topic, err)
	}
	fn, reg, ok := modbus.ReadKey(frame)
	if !ok {
		return fmt.Errorf("%s: not a read reply", topic)
	}
	if dl := modbus.FrameDataLogger(frame); dl != m.cfg.DataLog {
		return fmt.Errorf("%s: frame is from datalogger %q", topic, dl)
	}
	if t, err := strconv.Atoi(parts[len(parts)-2]); err != nil || byte(t) != fn {
		return fmt.Errorf("%s: topic function doesn't match the frame's (%d)", topic, fn)
	}
	if r, err := strconv.Atoi(parts[len(parts)-1]); err != nil || uint16(r) != reg {
		return fmt.Errorf("%s: topic register doesn't match the frame's (%d)", topic, reg)
	}
	at := time.UnixMilli(msg.Time)
	k := key{fn, reg}
	m.mu.Lock()
	defer m.mu.Unlock()
	if cur, ok := m.frames[k]; ok && cur.at.After(at) {
		return fmt.Errorf("%s: older than the frame already kept", topic)
	}
	m.frames[k] = entry{frame: frame, at: at}
	m.received.Add(1)
	return nil
}

// Reply is the frame to send for a client's request frame, or nil: not a read, nothing kept for it, or too old.
func (m *Mirror) Reply(request []byte) []byte {
	fn, reg, ok := modbus.ReadKey(request)
	if !ok {
		return nil
	}
	m.mu.Lock()
	e, have := m.frames[key{fn, reg}]
	m.mu.Unlock()
	if !have {
		return nil
	}
	maxAge := m.cfg.InputMaxAge
	if fn == 3 {
		maxAge = m.cfg.HoldMaxAge
	}
	if m.now().Sub(e.at) > maxAge {
		m.stale.Add(1)
		return nil
	}
	m.served.Add(1)
	return e.frame
}

// Fresh reports whether the newest input frame is within InputMaxAge: the source bridge is getting data.
func (m *Mirror) Fresh() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, e := range m.frames {
		if k.function == 4 && m.now().Sub(e.at) <= m.cfg.InputMaxAge {
			return true
		}
	}
	return false
}

// Run subscribes and serves until ctx ends.
func (m *Mirror) Run(ctx context.Context) error {
	if err := m.connect(); err != nil {
		return err
	}
	defer m.mqtt.Disconnect(disconnectQuiesceMs)
	ln, err := net.Listen("tcp", m.cfg.Listen)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", m.cfg.Listen, err)
	}
	logrus.Infof("Mirroring datalogger %s on %s (from %s)", m.cfg.DataLog, m.cfg.Listen, m.Topic())
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	return m.Serve(ctx, ln)
}

// Serve accepts clients on ln until it is closed.
func (m *Mirror) Serve(ctx context.Context, ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		if int(m.clients.Load()) >= m.cfg.MaxClients {
			logrus.Warnf("Refused %s: already %d clients", conn.RemoteAddr(), m.cfg.MaxClients)
			_ = conn.Close()
			continue
		}
		m.clients.Add(1)
		go func() {
			defer m.clients.Add(-1)
			m.handle(ctx, conn)
		}()
	}
}

func (m *Mirror) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	logrus.Infof("Client %s connected", conn.RemoteAddr())
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	r := bufio.NewReader(conn)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(clientIdleTimeout))
		req, err := modbus.ReadFrame(r)
		if err != nil {
			logrus.Infof("Client %s gone: %v", conn.RemoteAddr(), err)
			return
		}
		if len(req) > 7 && req[7] == 0xc2 && len(req) > 21 && (req[21] == 6 || req[21] == 0x10) {
			m.dropped.Add(1)
			logrus.Warnf("Dropped a write from %s: the mirror is read only", conn.RemoteAddr())
			continue
		}
		if reply := m.Reply(req); reply != nil {
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := conn.Write(reply); err != nil {
				logrus.Infof("Client %s gone: %v", conn.RemoteAddr(), err)
				return
			}
		}
	}
}

// connect subscribes to the raw frames. The client never publishes and has no will: it must not touch the source
// bridge's status topics.
func (m *Mirror) connect() error {
	opts := paho.NewClientOptions()
	opts.AddBroker(mqtt.BrokerURL(m.cfg.MQTT))
	if m.cfg.MQTT.TLS.Enabled {
		tlsCfg, err := mqtt.TLSConfig(m.cfg.MQTT.TLS, m.cfg.MQTT.Host)
		if err != nil {
			return err
		}
		opts.SetTLSConfig(tlsCfg)
	}
	name := m.cfg.MQTT.ClientName
	if name == "" {
		name = "lxp-mirror-" + m.cfg.DataLog
	}
	opts.SetClientID(name)
	opts.SetUsername(m.cfg.MQTT.Username)
	opts.SetPassword(m.cfg.MQTT.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(time.Minute)
	opts.SetCleanSession(true)
	opts.SetOnConnectHandler(func(c paho.Client) {
		m.mqttUp.Store(true)
		t := c.Subscribe(m.Topic(), 1, func(_ paho.Client, msg paho.Message) {
			if err := m.Store(msg.Topic(), msg.Payload()); err != nil {
				logrus.Debugf("Ignored: %v", err)
			}
		})
		if !t.WaitTimeout(subscribeTimeout) || t.Error() != nil {
			logrus.Errorf("Subscribing to %s failed: %v", m.Topic(), t.Error())
			return
		}
		logrus.Infof("MQTT connected, subscribed to %s", m.Topic())
	})
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		m.mqttUp.Store(false)
		logrus.Warnf("MQTT connection lost: %v (will reconnect)", err)
	})
	m.mqtt = paho.NewClient(opts)
	go func() {
		if t := m.mqtt.Connect(); t.Wait() && t.Error() != nil {
			logrus.Errorf("MQTT connect failed: %v", t.Error())
		}
	}()
	return nil
}

// Health is 204 while the newest input frame is fresh, else 503 with the reason.
func (m *Mirror) Health() (int, string) {
	switch {
	case !m.mqttUp.Load():
		return 503, "not connected to the MQTT broker"
	case !m.Fresh():
		return 503, fmt.Sprintf("no input frame newer than %s from the source bridge", m.cfg.InputMaxAge)
	}
	return 204, ""
}

// Stats for /metrics: frames received from MQTT, replies served, requests not answered because stale, writes
// dropped, clients connected.
func (m *Mirror) Stats() map[string]int64 {
	return map[string]int64{"received": m.received.Load(), "served": m.served.Load(), "stale": m.stale.Load(),
		"writes_dropped": m.dropped.Load(), "clients": int64(m.clients.Load())}
}
