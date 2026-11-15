package coordinator

import (
	"context"
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
)

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
}

type inverter struct {
	client modbus.Client
	config models.LXPConfig

	// mu guards hold and pending, which are written by the inverter read
	// loop and by MQTT commands.
	mu      sync.Mutex
	hold    map[uint16]uint16      // last raw value of each holding register
	pending map[uint16]*fieldWrite // bit field changes waiting for a fresh read
}

// fieldWrite is a change to part of a holding register. It is applied to the
// next value read from the inverter so other bits are never overwritten with
// stale data.
type fieldWrite struct {
	mask     uint16
	value    uint16
	deadline time.Time
}

type client struct {
	config  models.Config
	mqtt    mqtt.Client
	metrics metrics.Sink
	now     func() time.Time
	started time.Time

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
	return build(config, deps{
		metrics: sink,
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
		now:       d.now,
		mqtt:      d.mqtt,
		inverters: make(map[string]*inverter),
	}
	if c.now == nil {
		c.now = time.Now
	}
	if c.metrics == nil {
		c.metrics = metrics.Nop{}
	}
	c.started = c.now()
	if c.mqtt == nil {
		c.mqtt = mqtt.NewClient(config.MQTTConfig, false, c, c.onMQTTConnect)
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
		mc, err := d.newModbus(modConfig, c, &observer{c: c, tags: []string{metrics.FormatTag("deploymentID", modConfig.DataLog)}})
		if err != nil {
			return nil, fmt.Errorf("inverter %s: %w", modConfig.DataLog, err)
		}
		c.inverters[modConfig.DataLog] = &inverter{client: mc, config: modConfig, hold: map[uint16]uint16{}, pending: map[uint16]*fieldWrite{}}
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
	c.mqtt.SubMultiple(subTopics)
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
		c.mqtt.Close()
		_ = c.metrics.Close()
	})
}

// onMQTTConnect republishes the Home Assistant discovery configs.
func (c *client) onMQTTConnect() {
	if !c.config.MQTTConfig.HomeAssistant.Enabled {
		return
	}
	for _, id := range c.order {
		c.reportHomeAssistant(c.inverters[id].config)
	}
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
			c.emitHealthGauges()
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
	topic := fmt.Sprintf("%s/%s/%s/%s", c.config.MQTTConfig.NameSpace, dataLogger, strings.ToLower(register.RegisterType.String()), strings.ToLower(register.ShortName))
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
	value, err := rawValue(register, payload)
	if err != nil {
		reject("%v", err)
		return
	}
	min, max := registerRange(register)
	if value < int64(min) || value > int64(max) {
		reject("value %d outside allowed range %d..%d for register %d (%s)", value, min, max, registerID, register.ShortName)
		return
	}
	if register.BitMask != 0 {
		c.queueFieldWrite(inv, uint16(registerID), register, uint16(value))
		return
	}
	logrus.Infof("Accepted register write: datalog=%s register=%d (%s) value=%d", inv.config.DataLog, registerID, register.ShortName, value)
	bitValue := modbus.GetBytesForUInt16(uint16(value))
	command := modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 1, 0x6, uint16(registerID), bitValue...)
	inv.client.SendCommand(command)
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
	inv.mu.Lock()
	fw := inv.pending[addr]
	if fw == nil {
		fw = &fieldWrite{}
		inv.pending[addr] = fw
	}
	fw.mask |= register.BitMask
	fw.value = register.SetField(fw.value, value)
	fw.deadline = c.now().Add(fieldWriteTimeout)
	inv.mu.Unlock()
	logrus.Infof("Accepted field write: datalog=%s register=%d (%s) value=%d, reading register first", inv.config.DataLog, addr, register.ShortName, value)
	inv.client.SendCommand(modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 0, 0x3, addr, modbus.GetBytesForUInt16(1)...))
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
	inv.mu.Lock()
	defer inv.mu.Unlock()
	for i, v := range values {
		addr := start + uint16(i)
		inv.hold[addr] = v
		fw, ok := inv.pending[addr]
		if !ok {
			continue
		}
		delete(inv.pending, addr)
		if now.After(fw.deadline) {
			logrus.Warnf("Dropped field write to register %d on %s: no read reply in %s", addr, dataLogger, fieldWriteTimeout)
			continue
		}
		next := v&^fw.mask | fw.value&fw.mask
		if next == v {
			logrus.Infof("Register %d on %s already has the requested value %d", addr, dataLogger, v)
			continue
		}
		logrus.Infof("Writing register %d on %s: %d -> %d", addr, dataLogger, v, next)
		inv.client.SendCommand(modbus.BuildPacket(inv.config.DataLog, inv.config.Serial, 1, 0x6, addr, modbus.GetBytesForUInt16(next)...))
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

func (c *client) reportHomeAssistant(config models.LXPConfig) {
	device := registers.Device{
		Manufacturer: "LuxPower",
		Name:         fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, config.DataLog),
		Identifiers:  []string{fmt.Sprintf("%s_%s", c.config.MQTTConfig.NameSpace, config.DataLog)},
	}
	publish := func(id int, register registers.Register) {
		registerString, _ := json.Marshal(register.ToJson(id, device, c.config.MQTTConfig.NameSpace, config.DataLog))
		topic := fmt.Sprintf("%s/%s/%s/%s/config", strings.ToLower(c.config.MQTTConfig.HomeAssistant.Prefix), strings.ToLower(register.DiscoveryType().String()), device.Name, strings.ToLower(register.ShortName))
		if err := c.mqtt.SendMessage(topic, registerString, true); err != nil {
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
			// Read the holding registers each time the inverter (re)connects.
			connected := inv.client.Connected()
			if connected && !wasConnected && (cfg.PublishHoldingsOnConnect || holdEnabled) {
				logrus.Debugf("Requesting holding registers from %s", cfg.DataLog)
				c.sendHoldingPolls(inv)
			}
			wasConnected = connected
		case <-ctx.Done():
			return
		}
	}
}
