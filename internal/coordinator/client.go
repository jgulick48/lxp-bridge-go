package coordinator

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/metrics"
	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/jgulick48/lxp-bridge-go/internal/mqtt"
	"github.com/jgulick48/lxp-bridge-go/internal/registers"
	"github.com/sirupsen/logrus"
)

const (
	defaultHealthCheckDuration = 5 * time.Minute
	healthMetricsInterval      = 10 * time.Second
	maxRawRegisterValue        = 65535
	// defaultLongRead is how often input block 80 (BMS data) is requested.
	defaultLongRead = 30 * time.Second
	// defaultHoldRead is how often settings are re-read so changes made
	// elsewhere (EG4 app, inverter screen) reach Home Assistant.
	defaultHoldRead = 15 * time.Minute
	// fieldWriteTimeout is how long a bit field change waits for the read
	// of its register before it is dropped.
	fieldWriteTimeout = 30 * time.Second
	// fieldWriteMaxTries is how many times the register is read for one
	// field write (the first read plus retries).
	fieldWriteMaxTries = 3
)

// fieldWriteRetryDelays is how long to wait after the first and second read
// before sending the next one (about 3s and 10s after the write is accepted).
var fieldWriteRetryDelays = []time.Duration{3 * time.Second, 7 * time.Second}

// Client ties the inverter connections, MQTT and metrics together.
type Client interface {
	// Start begins polling and metrics reporting; it returns immediately.
	Start(ctx context.Context)
	// Close stops everything: inverter clients, MQTT and statsd.
	Close()
	// Health reports an HTTP status and short text body.
	Health() (int, string)
	// HealthHandler serves the liveness probe.
	HealthHandler() http.Handler
	// MetricsHandler serves Prometheus metrics at /metrics; it answers 404
	// when the prometheus config is not enabled.
	MetricsHandler() http.Handler
}

type inverter struct {
	client modbus.Client
	config models.LXPConfig
	model  registers.Model // as configured; may be ModelAuto

	// detected is the model chosen from the device type code when model is
	// ModelAuto. It has its own lock because it is written by the read loop
	// and read by the MQTT goroutine, and mu must not be held while
	// publishing discovery.
	detMu    sync.RWMutex
	detected registers.Model

	// mu guards hold and pending, which are written by the inverter read
	// loop and by MQTT commands.
	mu      sync.Mutex
	hold    map[uint16]uint16      // last raw value of each holding register
	pending map[uint16]*fieldWrite // bit field changes waiting for a fresh read
	clock   clockState
	// modelHintLogged is set once the device type code has been logged.
	modelHintLogged bool
}

// effectiveModel is the configured model, or for ModelAuto the detected one
// (ModelAuto while nothing has been detected).
func (inv *inverter) effectiveModel() registers.Model {
	if inv.model != registers.ModelAuto {
		return inv.model
	}
	inv.detMu.RLock()
	defer inv.detMu.RUnlock()
	if inv.detected == "" {
		return registers.ModelAuto
	}
	return inv.detected
}

// fieldWrite is a change to part of a holding register. It is applied to the
// next value read from the inverter so other bits are never overwritten with
// stale data.
type fieldWrite struct {
	mask     uint16
	value    uint16
	deadline time.Time
	sentAt   time.Time // when the register was last read for this change
	attempts int       // reads sent so far
}

type client struct {
	config  models.Config
	mqtt    mqtt.Client
	metrics metrics.Sink
	prom    *metrics.Prometheus // nil unless prometheus is enabled
	now     func() time.Time
	started time.Time

	// statusMu guards published, the last availability payload published for
	// each datalog. It is touched by the health tick and the MQTT on-connect
	// callback.
	statusMu  sync.Mutex
	published map[string]string

	// inverters is populated before Start and read-only afterwards.
	inverters map[string]*inverter
	order     []string

	wg        sync.WaitGroup
	closeOnce sync.Once
	cancel    context.CancelFunc
}

// deps are the injectable collaborators; tests replace them with fakes.
type deps struct {
	mqtt      mqtt.Client
	metrics   metrics.Sink
	prom      *metrics.Prometheus
	now       func() time.Time
	newModbus func(cfg models.LXPConfig, cb modbus.ParserCallback, obs modbus.Observer) (modbus.Client, error)
}

// New builds the coordinator with real MQTT, statsd and inverter clients.
func New(config models.Config) (Client, error) {
	sink, err := metrics.New(config.StatsDConfig.StatsServer)
	if err != nil {
		logrus.Errorf("Error creating stats client, metrics disabled: %s", err)
		sink = metrics.Nop{}
	}
	var prom *metrics.Prometheus
	if config.Prometheus.Enabled {
		prom = metrics.NewPrometheus(config.Prometheus.StaleAfter)
		sink = metrics.Multi{sink, prom}
	}
	return build(config, deps{
		metrics: sink,
		prom:    prom,
		now:     time.Now,
		newModbus: func(cfg models.LXPConfig, cb modbus.ParserCallback, obs modbus.Observer) (modbus.Client, error) {
			return modbus.NewClient(cfg, cb, obs)
		},
	})
}

func build(config models.Config, d deps) (*client, error) {
	c := &client{
		config:    config,
		metrics:   d.metrics,
		prom:      d.prom,
		now:       d.now,
		mqtt:      d.mqtt,
		inverters: make(map[string]*inverter),
		published: make(map[string]string),
	}
	if c.now == nil {
		c.now = time.Now
	}
	if c.metrics == nil {
		c.metrics = metrics.Nop{}
	}
	c.started = c.now()
	if c.mqtt == nil {
		m, err := mqtt.NewClient(config.MQTTConfig, false, c, c.onMQTTConnect)
		if err != nil {
			return nil, err
		}
		c.mqtt = m
	}
	if config.ReadOnly {
		logrus.Info("Read-only mode: writes to the inverter are disabled")
	}
	for _, modConfig := range config.Inverters {
		if !modConfig.Enabled {
			logrus.Warnf("Inverter %s is not enabled in config (enabled: false or missing), skipping", modConfig.DataLog)
			continue
		}
		if modConfig.ReadTimeout == 0 {
			modConfig.ReadTimeout = 300 * time.Second
		}
		if modConfig.LongRead == 0 {
			modConfig.LongRead = defaultLongRead
		}
		if modConfig.HoldRead == 0 {
			modConfig.HoldRead = defaultHoldRead
		}
		if modConfig.ShortRead == 0 {
			modConfig.ShortRead = 10 * time.Second
		}
		if modConfig.HealthCheckDuration == 0 {
			modConfig.HealthCheckDuration = defaultHealthCheckDuration
		}
		if _, dup := c.inverters[modConfig.DataLog]; dup {
			return nil, fmt.Errorf("duplicate datalog %q in config", modConfig.DataLog)
		}
		model, err := registers.ParseModel(modConfig.Model)
		if err != nil {
			return nil, fmt.Errorf("inverter %s: invalid model: %w", modConfig.DataLog, err)
		}
		mc, err := d.newModbus(modConfig, c, &observer{c: c, tags: []string{metrics.FormatTag("deploymentID", modConfig.DataLog)}})
		if err != nil {
			return nil, fmt.Errorf("inverter %s: %w", modConfig.DataLog, err)
		}
		if config.ReadOnly && modConfig.ClockSync.Enabled {
			logrus.Warnf("Inverter %s: clock sync is disabled by read_only; the clock drift metric is still reported", modConfig.DataLog)
		}
		cs := clockState{maxDrift: modConfig.ClockSync.MaxDrift}
		if cs.maxDrift == 0 {
			cs.maxDrift = defaultClockMaxDrift
		}
		if tz := modConfig.ClockSync.Timezone; tz != "" {
			loc, err := time.LoadLocation(tz)
			switch {
			case err == nil:
				cs.loc = loc
			case modConfig.ClockSync.Enabled:
				return nil, fmt.Errorf("inverter %s: invalid clock_sync timezone %q: %w", modConfig.DataLog, tz, err)
			default:
				logrus.Warnf("Inverter %s: ignoring invalid clock_sync timezone %q: %s", modConfig.DataLog, tz, err)
			}
		} else if modConfig.ClockSync.Enabled {
			return nil, fmt.Errorf("inverter %s: clock_sync is enabled but clock_sync.timezone is not set (IANA name such as America/Denver)", modConfig.DataLog)
		}
		c.inverters[modConfig.DataLog] = &inverter{client: mc, config: modConfig, model: model, hold: map[uint16]uint16{}, pending: map[uint16]*fieldWrite{}, clock: cs}
		c.order = append(c.order, modConfig.DataLog)
	}
	return c, nil
}

func (c *client) Start(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)
	subTopics := make([]string, 0, len(c.order))
	for _, id := range c.order {
		inv := c.inverters[id]
		inv.client.Connect()
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.setupPolling(ctx, inv)
		}()
		subTopics = append(subTopics, fmt.Sprintf("%s/cmd/%s/set/#", c.config.MQTTConfig.NameSpace, inv.config.DataLog))
	}
	if !c.config.ReadOnly {
		c.mqtt.SubMultiple(subTopics)
	}
	c.mqtt.Connect()
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.reportHealthMetrics(ctx)
	}()
}

func (c *client) Close() {
	c.closeOnce.Do(func() {
		if c.cancel != nil {
			c.cancel()
		}
		c.wg.Wait()
		for _, id := range c.order {
			c.inverters[id].client.Close()
		}
		for _, id := range c.order {
			c.publishStatus(c.inverters[id], mqtt.PayloadOffline)
		}
		c.mqtt.Close()
		_ = c.metrics.Close()
	})
}

// inverterStatusTopic is "<namespace>/<datalog>/status".
func (c *client) inverterStatusTopic(inv *inverter) string {
	return fmt.Sprintf("%s/%s/status", c.config.MQTTConfig.NameSpace, inv.config.DataLog)
}

// inverterOnline reports whether the inverter is connected and has delivered
// a valid frame since process start within its health check duration.
func (c *client) inverterOnline(inv *inverter, now time.Time) bool {
	if !inv.client.Connected() {
		return false
	}
	last := inv.client.LastFrame()
	if last.IsZero() || last.Before(c.started) {
		return false
	}
	return now.Sub(last) <= inv.config.HealthCheckDuration
}

// publishStatus sends a retained availability payload and records it as
// published only when the send succeeded.
func (c *client) publishStatus(inv *inverter, payload string) {
	if err := c.mqtt.SendMessage(c.inverterStatusTopic(inv), payload, true); err != nil {
		logrus.Debugf("Error publishing status for %s: %v", inv.config.DataLog, err)
		return
	}
	c.statusMu.Lock()
	c.published[inv.config.DataLog] = payload
	c.statusMu.Unlock()
}

func statusPayload(online bool) string {
	if online {
		return mqtt.PayloadOnline
	}
	return mqtt.PayloadOffline
}

// updateInverterStatus publishes each inverter's availability when it differs
// from what was last published.
func (c *client) updateInverterStatus() {
	now := c.now()
	for _, id := range c.order {
		inv := c.inverters[id]
		payload := statusPayload(c.inverterOnline(inv, now))
		c.statusMu.Lock()
		same := c.published[id] == payload
		c.statusMu.Unlock()
		if !same {
			c.publishStatus(inv, payload)
		}
	}
}

// onMQTTConnect republishes inverter availability and, when enabled, the Home
// Assistant discovery configs.
func (c *client) onMQTTConnect() {
	now := c.now()
	for _, id := range c.order {
		inv := c.inverters[id]
		c.publishStatus(inv, statusPayload(c.inverterOnline(inv, now)))
	}
	if !c.config.MQTTConfig.HomeAssistant.Enabled {
		return
	}
	for _, id := range c.order {
		inv := c.inverters[id]
		if inv.effectiveModel() == registers.ModelAuto {
			logrus.Debugf("inverter %s: model not detected yet, deferring Home Assistant discovery", inv.config.DataLog)
			continue
		}
		c.reportHomeAssistant(inv)
	}
}

// MetricsHandler implements Client.
func (c *client) MetricsHandler() http.Handler {
	if c.prom == nil {
		return http.NotFoundHandler()
	}
	return c.prom
}

func (c *client) HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, body := c.Health()
		logrus.Debugf("Got Healthcheck request sending status %v", status)
		if status != http.StatusNoContent {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(status)
			_, _ = fmt.Fprintln(w, body)
			return
		}
		w.WriteHeader(status)
	})
}

// lastActivity is the last valid frame, or process start if none yet.
func (c *client) lastActivity(inv *inverter) time.Time {
	last := inv.client.LastFrame()
	if last.Before(c.started) {
		return c.started
	}
	return last
}

// Health is healthy iff every enabled inverter has produced a valid frame
// within its health check duration (5m by default). MQTT is not considered.
func (c *client) Health() (int, string) {
	now := c.now()
	for _, id := range c.order {
		inv := c.inverters[id]
		age := now.Sub(c.lastActivity(inv))
		if age > inv.config.HealthCheckDuration {
			return http.StatusServiceUnavailable, fmt.Sprintf("inverter %s: no valid frame for %s (limit %s)",
				id, age.Round(time.Second), inv.config.HealthCheckDuration)
		}
	}
	return http.StatusNoContent, ""
}

func (c *client) metricName(suffix string) string {
	return fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, suffix)
}

func (c *client) reportHealthMetrics(ctx context.Context) {
	ticker := time.NewTicker(healthMetricsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.dropOfflineSeries()
			c.emitHealthGauges()
			c.updateInverterStatus()
		}
	}
}

// dropOfflineSeries removes the Prometheus gauges of every offline inverter
// (same rule as its availability topic), so a dead inverter's last readings
// are not scraped as current. Its health gauges are re-added straight after by
// emitHealthGauges. Holding values, read only every hold_read, stay exposed
// while the inverter is online.
func (c *client) dropOfflineSeries() {
	if c.prom == nil {
		return
	}
	now := c.now()
	for _, id := range c.order {
		if !c.inverterOnline(c.inverters[id], now) {
			c.prom.DropTag(metrics.FormatTag("deploymentID", id))
		}
	}
}

func (c *client) emitHealthGauges() {
	now := c.now()
	for _, id := range c.order {
		inv := c.inverters[id]
		tags := []string{metrics.FormatTag("deploymentID", id)}
		connected := 0.0
		if inv.client.Connected() {
			connected = 1
		}
		metrics.Gauge(c.metrics, c.metricName("bridge_connected"), tags, connected)
		metrics.Gauge(c.metrics, c.metricName("bridge_last_frame_age_s"), tags, now.Sub(c.lastActivity(inv)).Seconds())
	}
}

// observer reports modbus client events as metrics.
type observer struct {
	c    *client
	tags []string
}

func (o *observer) Reconnect() {
	metrics.Count(o.c.metrics, o.c.metricName("bridge_reconnects"), o.tags, 1)
}
func (o *observer) CommandDropped() {
	metrics.Count(o.c.metrics, o.c.metricName("bridge_commands_dropped"), o.tags, 1)
}
func (o *observer) Frame(result string) {
	metrics.Count(o.c.metrics, o.c.metricName("bridge_frames"), append(append([]string(nil), o.tags...), metrics.FormatTag("result", result)), 1)
}

func (c *client) ReportValue(register registers.Register, value int32, dataLogger string) {
	if inv, ok := c.inverters[dataLogger]; ok {
		var have bool
		if register, have = inv.effectiveModel().Resolve(register); !have {
			return
		}
	}
	topic := fmt.Sprintf("%s/%s/%s/%s", c.config.MQTTConfig.NameSpace, dataLogger, strings.ToLower(register.RegisterType.String()), strings.ToLower(register.ShortName))
	if register.TimeOfDay {
		// A time of day is a label, not a measurement: no gauge.
		text, ok := registers.FormatTimeOfDay(uint16(value))
		if !ok {
			logrus.Debugf("Ignoring invalid time %#04x for %s on %s", uint16(value), register.ShortName, dataLogger)
			return
		}
		_ = c.mqtt.SendMessage(topic, text, true)
		return
	}
	floatVal := float32(value)
	floatVal = floatVal * register.Multiplier
	_ = c.mqtt.SendMessage(topic, fmt.Sprintf("%v", floatVal), true)
	metrics.Gauge(c.metrics, fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, strings.ToLower(register.ShortName)), []string{metrics.FormatTag("deploymentID", dataLogger)}, float64(floatVal))
}

func (c *client) ReportPowerOpenEVSE(soc, batteryPower int32) {
	if !c.config.OpenEVSE.Enabled || !c.config.MQTTConfig.Enabled {
		return
	}
	exportPower := -batteryPower
	if soc < c.config.OpenEVSE.SOCChargeStart {
		exportPower = c.config.OpenEVSE.ChargeStopValue
	}
	if soc >= c.config.OpenEVSE.SOCChargeMax {
		exportPower = c.config.OpenEVSE.ChargeMaxValue
	}
	_ = c.mqtt.SendMessage(c.config.OpenEVSE.Topic, fmt.Sprintf("%v", exportPower), true)
}

// controllable reports whether a hold register is exposed to Home Assistant as
// a settable entity (Writeable with a command topic, and not a sensor/text).
func controllable(r registers.Register) bool {
	if !r.Writeable {
		return false
	}
	switch r.HomeAssistantType {
	case registers.Number, registers.Switch:
		return true
	case registers.Text:
		return r.TimeOfDay
	}
	return false
}

// registerRange returns the allowed raw value range for a register.
func registerRange(r registers.Register) (min, max int) {
	if r.Min == 0 && r.Max == 0 {
		return 0, maxRawRegisterValue
	}
	return r.Min, r.Max
}

// HandleCommand processes <ns>/cmd/<datalog>/set/hold/<register> messages.
// Only single-register, Home Assistant controllable hold registers are
// written, and only with an in-range integer value.
func (c *client) HandleCommand(topic string, message []byte) {
	if c.config.ReadOnly {
		// Commands are not subscribed to in read-only mode; this is a
		// defensive stop in case one arrives anyway.
		logrus.WithFields(logrus.Fields{"topic": topic}).Warn("Rejected register write: read-only mode")
		return
	}
	segments := strings.Split(topic, "/")
	if len(segments) < 6 {
		return
	}
	if segments[0] != c.config.MQTTConfig.NameSpace || segments[1] != "cmd" || segments[3] != "set" || segments[4] != "hold" {
		return
	}
	reject := func(reason string, args ...interface{}) {
		logrus.WithFields(logrus.Fields{"topic": topic, "payload": string(message)}).
			Warnf("Rejected register write: %s", fmt.Sprintf(reason, args...))
	}
	inv, ok := c.inverters[segments[2]]
	if !ok {
		reject("unknown datalog %q", segments[2])
		return
	}
	registerID, err := strconv.Atoi(segments[5])
	if err != nil {
		reject("invalid register id %q", segments[5])
		return
	}
	var register registers.Register
	switch len(segments) {
	case 6:
		holdRegister, ok := registers.HoldIDToRegisterMap[registerID]
		if !ok {
			reject("unknown hold register %d", registerID)
			return
		}
		register, ok = registers.HoldRegisters[holdRegister]
		if !ok {
			reject("unknown hold register %d", registerID)
			return
		}
	case 7:
		register, ok = holdField(registerID, segments[6])
		if !ok {
			reject("unknown field %q of hold register %d", segments[6], registerID)
			return
		}
	default:
		reject("unexpected topic")
		return
	}
	model := inv.effectiveModel()
	if model == registers.ModelAuto {
		reject("model not detected yet")
		return
	}
	register, ok = model.Resolve(register)
	if !ok {
		reject("register %d (%s) is not available on model %s", registerID, register.ShortName, model)
		return
	}
	if !controllable(register) {
		reject("register %d (%s) is not writable from Home Assistant", registerID, register.ShortName)
		return
	}
	if register.RegisterLength != 0 {
		reject("register %d (%s) is not a single register", registerID, register.ShortName)
		return
	}
	payload := strings.TrimSpace(string(message))
	if register.HomeAssistantType == registers.Switch {
		switch strings.ToUpper(payload) {
		case "ON", "TRUE":
			payload = "1"
		case "OFF", "FALSE":
			payload = "0"
		}
	}
	var value int64
	if register.TimeOfDay {
		// The raw value is hour | minute<<8, so the numeric range check
		// does not apply; ParseTimeOfDay validates hour and minute.
		raw, err := registers.ParseTimeOfDay(payload)
		if err != nil {
			reject("%v", err)
			return
		}
		value = int64(raw)
	} else {
		var err error
		value, err = rawValue(register, payload)
		if err != nil {
			reject("%v", err)
			return
		}
		min, max := registerRange(register)
		if value < int64(min) || value > int64(max) {
			reject("value %d outside allowed range %d..%d for register %d (%s)", value, min, max, registerID, register.ShortName)
			return
		}
	}
	if register.BitMask != 0 {
		c.queueFieldWrite(inv, uint16(registerID), register, uint16(value))
		return
	}
	logrus.Infof("Accepted register write: datalog=%s register=%d (%s) value=%d", inv.config.DataLog, registerID, register.ShortName, value)
	bitValue := modbus.GetBytesForUInt16(uint16(value))
	command := modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 1, 0x6, uint16(registerID), bitValue...)
	c.sendWrite(inv, command, fmt.Sprintf("register %d", registerID))
}

// sendWrite is the single choke point for everything that changes inverter
// state (function 6 and 0x10 frames). In read-only mode it refuses, whatever
// the caller. Reads go straight to inv.client.SendCommand.
func (c *client) sendWrite(inv *inverter, frame []byte, what string) {
	if c.config.ReadOnly {
		logrus.Warnf("Read-only mode: refusing to write %s on %s", what, inv.config.DataLog)
		return
	}
	inv.client.SendCommand(frame)
}

// holdField finds the bit field register named name (case-insensitive) in
// hold register id.
func holdField(id int, name string) (registers.Register, bool) {
	for hr, addr := range registers.HoldRegisterToIDMap {
		if addr != id {
			continue
		}
		r, ok := registers.HoldRegisters[hr]
		if ok && r.BitMask != 0 && strings.EqualFold(r.ShortName, name) {
			return r, true
		}
	}
	return registers.Register{}, false
}

// queueFieldWrite records a bit field change and reads the register; the
// change is written once the fresh value arrives (see ReportHolding).
func (c *client) queueFieldWrite(inv *inverter, addr uint16, register registers.Register, value uint16) {
	if c.config.ReadOnly {
		return
	}
	inv.mu.Lock()
	fw := inv.pending[addr]
	if fw == nil {
		fw = &fieldWrite{}
		inv.pending[addr] = fw
	}
	fw.mask |= register.BitMask
	fw.value = register.SetField(fw.value, value)
	now := c.now()
	fw.deadline = now.Add(fieldWriteTimeout)
	fw.sentAt = now
	fw.attempts = 1
	inv.mu.Unlock()
	logrus.Infof("Accepted field write: datalog=%s register=%d (%s) value=%d, reading register first", inv.config.DataLog, addr, register.ShortName, value)
	inv.client.SendCommand(modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 0, 0x3, addr, modbus.GetBytesForUInt16(1)...))
}

// checkFieldWrites re-reads registers whose field write has had no reply and
// drops writes that ran out of time. It is called once a second.
func (c *client) checkFieldWrites(inv *inverter, now time.Time) {
	if c.config.ReadOnly {
		return
	}
	var resend []uint16
	var dropped []string
	inv.mu.Lock()
	for addr, fw := range inv.pending {
		if now.After(fw.deadline) {
			delete(inv.pending, addr)
			dropped = append(dropped, fmt.Sprintf("Dropped field write to register %d on %s: no read reply after %d tries in %s", addr, inv.config.DataLog, fw.attempts, fieldWriteTimeout))
			continue
		}
		if fw.attempts >= fieldWriteMaxTries || fw.attempts > len(fieldWriteRetryDelays) || !inv.client.Connected() {
			continue
		}
		if now.Sub(fw.sentAt) >= fieldWriteRetryDelays[fw.attempts-1] {
			fw.attempts++
			fw.sentAt = now
			resend = append(resend, addr)
			logrus.Infof("Retrying read of register %d on %s for field write (try %d)", addr, inv.config.DataLog, fw.attempts)
		}
	}
	inv.mu.Unlock()
	tags := []string{metrics.FormatTag("deploymentID", inv.config.DataLog)}
	for _, msg := range dropped {
		logrus.Warn(msg)
		metrics.Count(c.metrics, c.metricName("bridge_field_writes_dropped"), tags, 1)
	}
	for _, addr := range resend {
		inv.client.SendCommand(modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 0, 0x3, addr, modbus.GetBytesForUInt16(1)...))
	}
}

// deviceTypeReg is the holding register with the device type code.
const deviceTypeReg = 19

// logModelHint logs the device type code once per inverter. For a model that
// was configured explicitly it warns when the code points at another layout.
func logModelHint(inv *inverter, code uint16) {
	dl := inv.config.DataLog
	detected := registers.DetectModel(code)
	switch {
	case inv.model == registers.ModelAuto:
		logrus.Infof("inverter %s: device type code %d, using model %s (auto-detected)", dl, code, detected)
	case inv.model != detected:
		logrus.Warnf("inverter %s is configured as model %s but reports device type code %d (looks like %s); the configured model overrides the detected one", dl, inv.model, code, detected)
	default:
		logrus.Infof("inverter %s device type code %d", dl, code)
	}
}

// detectModel records the model for an auto inverter the first time the
// device type code is seen. It reports whether it did.
func (inv *inverter) detectModel(code uint16) bool {
	if inv.model != registers.ModelAuto {
		return false
	}
	inv.detMu.Lock()
	defer inv.detMu.Unlock()
	if inv.detected != "" {
		return false
	}
	inv.detected = registers.DetectModel(code)
	return true
}

// RawFrame is the payload of a <namespace>/<datalog>/raw/<function>/<register> message: when the frame arrived
// (Unix ms) and the frame itself, hex encoded.
type RawFrame struct {
	Time  int64  `json:"t"`
	Frame string `json:"frame"`
}

// RawTopic is where a read reply's raw frame is published.
func RawTopic(namespace, dataLogger string, function byte, register uint16) string {
	return fmt.Sprintf("%s/%s/raw/%d/%d", namespace, dataLogger, function, register)
}

// ReportRawFrame publishes a valid read reply as received (mqtt.publish_raw_frames), retained, for a mirror.
func (c *client) ReportRawFrame(dataLogger string, function byte, register uint16, frame []byte) {
	if !c.config.MQTTConfig.PublishRawFrames || !c.mqtt.IsEnabled() {
		return
	}
	if _, ok := c.inverters[dataLogger]; !ok {
		return
	}
	payload, err := json.Marshal(RawFrame{Time: c.now().UnixMilli(), Frame: hex.EncodeToString(frame)})
	if err != nil {
		return
	}
	_ = c.mqtt.SendMessage(RawTopic(c.config.MQTTConfig.NameSpace, dataLogger, function, register), payload, true)
}

// ReportHolding receives raw holding register values from the inverter. It
// caches them and writes any pending bit field changes on top of the fresh
// values.
func (c *client) ReportHolding(dataLogger string, start uint16, values []uint16) {
	inv, ok := c.inverters[dataLogger]
	if !ok {
		return
	}
	now := c.now()
	announce := false
	// Registered first so it runs after inv.mu is released.
	defer func() {
		if announce && c.config.MQTTConfig.HomeAssistant.Enabled {
			c.reportHomeAssistant(inv)
		}
	}()
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if start <= deviceTypeReg && int(start)+len(values) > deviceTypeReg {
		code := values[deviceTypeReg-int(start)]
		announce = inv.detectModel(code)
		if !inv.modelHintLogged {
			inv.modelHintLogged = true
			logModelHint(inv, code)
		}
	}
	for i, v := range values {
		addr := start + uint16(i)
		inv.hold[addr] = v
		fw, ok := inv.pending[addr]
		if !ok {
			continue
		}
		delete(inv.pending, addr)
		if now.After(fw.deadline) {
			logrus.Warnf("Dropped field write to register %d on %s: no read reply after %d tries in %s", addr, dataLogger, fw.attempts, fieldWriteTimeout)
			metrics.Count(c.metrics, c.metricName("bridge_field_writes_dropped"), []string{metrics.FormatTag("deploymentID", dataLogger)}, 1)
			continue
		}
		next := v&^fw.mask | fw.value&fw.mask
		if next == v {
			logrus.Infof("Register %d on %s already has the requested value %d", addr, dataLogger, v)
			continue
		}
		logrus.Infof("Writing register %d on %s: %d -> %d", addr, dataLogger, v, next)
		c.sendWrite(inv, modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 1, 0x6, addr, modbus.GetBytesForUInt16(next)...), fmt.Sprintf("register %d", addr))
	}
	if start <= clockRegStart && int(start)+len(values) > clockRegStart+clockRegCount-1 {
		off := clockRegStart - int(start)
		c.checkClock(inv, now, values[off:off+clockRegCount])
	}
}

// rawValue converts a command payload to the register's raw value. Home
// Assistant sends values in display units (the discovery config scales min,
// max and step by the register multiplier), e.g. "48.5" V for a 0.1 V register.
func rawValue(register registers.Register, payload string) (int64, error) {
	display, err := strconv.ParseFloat(payload, 64)
	if err != nil || math.IsNaN(display) || math.IsInf(display, 0) {
		return 0, fmt.Errorf("value %q is not a number", payload)
	}
	// Multipliers are float32; go through the shortest decimal form so 0.1
	// becomes 0.1 rather than 0.10000000149.
	multiplier, _ := strconv.ParseFloat(strconv.FormatFloat(float64(register.Multiplier), 'g', -1, 32), 64)
	if multiplier == 0 {
		multiplier = 1
	}
	raw := display / multiplier
	rounded := math.Round(raw)
	if math.Abs(raw-rounded) > 1e-6 {
		return 0, fmt.Errorf("value %q is not a multiple of the register step %g", payload, multiplier)
	}
	if rounded < math.MinInt32 || rounded > math.MaxInt32 {
		return 0, fmt.Errorf("value %q is out of range", payload)
	}
	return int64(rounded), nil
}

func (c *client) reportHomeAssistant(inv *inverter) {
	config := inv.config
	model := inv.effectiveModel()
	device := registers.Device{
		Manufacturer: "LuxPower",
		Name:         fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, config.DataLog),
		Identifiers:  []string{fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, config.DataLog)},
	}
	prefix := strings.ToLower(c.config.MQTTConfig.HomeAssistant.Prefix)
	configTopic := func(t registers.HomeAssistantType, register registers.Register) string {
		return fmt.Sprintf("%s/%s/%s/%s/config", prefix, strings.ToLower(t.String()), device.Name, strings.ToLower(register.ShortName))
	}
	clear := func(topic string) {
		if err := c.mqtt.SendMessage(topic, "", true); err != nil {
			logrus.Errorf("Error clearing stale entity from homeassistant %s", err.Error())
		}
	}
	publish := func(id int, base registers.Register) {
		register, ok := model.Resolve(base)
		var stale []string // topics an earlier run may have left retained
		addBase := func() {
			stale = append(stale, configTopic(base.DiscoveryType(), base))
			if base.Writeable {
				readOnlyBase := base.AsReadOnly()
				stale = append(stale, configTopic(readOnlyBase.DiscoveryType(), base))
			}
		}
		if !ok {
			// The model does not have this register: remove any entity an
			// install made before the model was set.
			addBase()
			for _, t := range stale {
				clear(t)
			}
			return
		}
		if register.Writeable {
			// A writable register is announced as a control normally and as a
			// sensor in read-only mode. Clear whichever variant an earlier run
			// with the other setting left retained, so Home Assistant removes
			// it instead of showing the register twice.
			readOnly := register.AsReadOnly()
			other := readOnly
			if c.config.ReadOnly {
				other = register
				register = readOnly
			}
			stale = append(stale, configTopic(other.DiscoveryType(), other))
		}
		if register.DiscoveryType() != base.DiscoveryType() {
			addBase()
		}
		target := configTopic(register.DiscoveryType(), register)
		done := map[string]bool{target: true}
		for _, t := range stale {
			if !done[t] {
				done[t] = true
				clear(t)
			}
		}
		registerString, _ := json.Marshal(register.ToJson(id, device, c.config.MQTTConfig.NameSpace, config.DataLog))
		if err := c.mqtt.SendMessage(target, registerString, true); err != nil {
			logrus.Errorf("Error sending sensor config to homeassistant %s", err.Error())
		}
	}
	for id, register := range registers.InputRegisters {
		publish(int(id), register)
	}
	for id, register := range registers.HoldRegisters {
		publish(int(id), register)
	}
}

func (c *client) sendHoldingPolls(inv *inverter) {
	n := modbus.GetBytesForUInt16(40)
	for _, start := range []uint16{0, 40, 80, 120, 160, 200} {
		inv.client.SendCommand(modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 0, 0x3, start, n...))
	}
}

// deviceTypeRequestInterval is how often holding 19 is requested while an
// auto model inverter has not been detected.
const deviceTypeRequestInterval = 30 * time.Second

// requestDeviceType reads holding 19 (the device type code) on its own. It is
// a read, so it is also sent in read-only mode.
func (c *client) requestDeviceType(inv *inverter) {
	inv.client.SendCommand(modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 0, 0x3, deviceTypeReg, modbus.GetBytesForUInt16(1)...))
}

// inputBlockLength is how many registers to read from start. Block 160 is
// read through 204 to include the per-leg grid and generator values.
func inputBlockLength(start uint16) uint16 {
	if start == 160 {
		return 45
	}
	return 40
}

func (c *client) setupPolling(ctx context.Context, inv *inverter) {
	cfg := inv.config
	var short, long, hold <-chan time.Time
	if cfg.ShortRead != -1*time.Second {
		t := time.NewTicker(cfg.ShortRead)
		defer t.Stop()
		short = t.C
	}
	if cfg.LongRead != -1*time.Second {
		t := time.NewTicker(cfg.LongRead)
		defer t.Stop()
		long = t.C
	}
	holdEnabled := cfg.HoldRead != -1*time.Second
	if holdEnabled {
		t := time.NewTicker(cfg.HoldRead)
		defer t.Stop()
		hold = t.C
	}
	lwt := time.NewTicker(20 * time.Second)
	defer lwt.Stop()
	connCheck := time.NewTicker(time.Second)
	defer connCheck.Stop()

	wasConnected := false
	var lastTypeReq time.Time
	poll := func(function byte, starts ...uint16) {
		if !inv.client.Connected() {
			logrus.Debugf("Not connected to %s, skipping poll", cfg.DataLog)
			return
		}
		for _, s := range starts {
			inv.client.SendCommand(modbus.BuildPacket(cfg.DataLog, cfg.Serial, 0, function, s, modbus.GetBytesForUInt16(inputBlockLength(s))...))
		}
	}
	for {
		select {
		case <-short:
			logrus.Debug("Polling for inputs 1, 2, 4, and 5")
			poll(0x4, 0, 40, 120, 160)
		case <-long:
			logrus.Debug("Polling for input 3 (BMS)")
			poll(0x4, 80)
		case <-hold:
			if inv.client.Connected() {
				logrus.Debug("Polling holding registers")
				c.sendHoldingPolls(inv)
			}
		case <-lwt.C:
			_ = c.mqtt.SendMessage(fmt.Sprintf("%s/LWT", c.config.MQTTConfig.NameSpace), "Online", true)
		case <-connCheck.C:
			c.checkFieldWrites(inv, c.now())
			// Read the holding registers each time the inverter (re)connects.
			connected := inv.client.Connected()
			sentHolds := false
			if connected && !wasConnected && (cfg.PublishHoldingsOnConnect || holdEnabled) {
				logrus.Debugf("Requesting holding registers from %s", cfg.DataLog)
				c.sendHoldingPolls(inv)
				sentHolds = true
				lastTypeReq = c.now()
			}
			if connected && inv.effectiveModel() == registers.ModelAuto {
				// The layout is unknown until holding 19 has been read.
				if !sentHolds && (!wasConnected || c.now().Sub(lastTypeReq) >= deviceTypeRequestInterval) {
					logrus.Debugf("Requesting device type from %s", cfg.DataLog)
					c.requestDeviceType(inv)
					lastTypeReq = c.now()
				}
			}
			wasConnected = connected
		case <-ctx.Done():
			return
		}
	}
}
