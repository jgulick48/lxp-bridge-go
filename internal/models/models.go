package models

import (
	"time"
)

type Config struct {
	LogLevel string `yaml:"loglevel"`
	// ReadOnly makes the bridge monitoring-only: nothing is ever written to
	// any inverter (MQTT commands, bit-field writes and clock sync are off)
	// and Home Assistant is offered sensors only.
	ReadOnly     bool             `yaml:"read_only"`
	MQTTConfig   MQTTConfig       `yaml:"mqtt"`
	StatsDConfig StatsDConfig     `yaml:"statsd"`
	Prometheus   PrometheusConfig `yaml:"prometheus"`
	Inverters    []LXPConfig      `yaml:"inverters"`
	OpenEVSE     OpenEVSEConfig   `yaml:"openEVSE"`
	// Mirror, when enabled, runs a read-only dongle mirror instead of the bridge (see internal/mirror).
	Mirror MirrorConfig `yaml:"mirror"`
}

// MirrorConfig serves one dongle's frames, taken from MQTT, to LuxPower clients on Listen.
type MirrorConfig struct {
	Enabled bool `yaml:"enabled"`
	// DataLog is the datalogger serial to mirror.
	DataLog string `yaml:"datalog"`
	// MQTT is the broker the source bridge publishes raw frames to (host, port, tls, namespace, clientName).
	MQTT MQTTConfig `yaml:"mqtt"`
	// Listen is where clients connect, as to a dongle (default 0.0.0.0:8000).
	Listen string `yaml:"listen"`
	// InputMaxAge and HoldMaxAge: older frames aren't served (defaults 2m and 24h).
	InputMaxAge time.Duration `yaml:"input_max_age"`
	HoldMaxAge  time.Duration `yaml:"hold_max_age"`
	// MaxClients connected at once (default 4).
	MaxClients int `yaml:"max_clients"`
}

type MQTTConfig struct {
	Enabled       bool                `yaml:"enabled"`
	Host          string              `yaml:"host"`
	Port          int                 `yaml:"port"`
	ClientName    string              `yaml:"clientName"`
	Username      string              `yaml:"username"`
	Password      string              `yaml:"password"`
	NameSpace     string              `yaml:"namespace"`
	HomeAssistant HomeAssistantConfig `yaml:"homeassistant"`
	TLS           MQTTTLSConfig       `yaml:"tls"`
	// PublishRawFrames publishes every valid read reply from the dongle, as received, to
	// <namespace>/<datalog>/raw/<function>/<start register> (retained), for a mirror to serve.
	PublishRawFrames bool `yaml:"publish_raw_frames"`
}

// MQTTTLSConfig configures TLS (and mutual TLS) for the broker connection.
type MQTTTLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CAFile             string `yaml:"ca_file"`
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
	ServerName         string `yaml:"server_name"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

type OpenEVSEConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Topic           string `yaml:"topic"`
	SOCChargeStart  int32  `yaml:"SOCChargeStart"`
	SOCChargeMax    int32  `yaml:"SOCChargeMax"`
	ChargeStopValue int32  `yaml:"chargeStopValue"`
	ChargeMaxValue  int32  `yaml:"chargeMaxValue"`
}

type StatsDConfig struct {
	StatsServer string `yaml:"server"`
}

// PrometheusConfig enables the /metrics endpoint on the health HTTP server.
type PrometheusConfig struct {
	Enabled bool `yaml:"enabled"`
	// StaleAfter hides gauges not updated for this long. 0 (default) disables
	// it; an offline inverter's gauges are dropped regardless.
	StaleAfter time.Duration `yaml:"stale_after"`
}

type HomeAssistantConfig struct {
	Enabled bool   `yaml:"enabled"`
	Prefix  string `yaml:"prefix"`
}

type LXPConfig struct {
	Enabled                  bool          `yaml:"enabled"`
	Host                     string        `yaml:"host"`
	Port                     string        `yaml:"port"`
	Serial                   string        `yaml:"serial"`
	DataLog                  string        `yaml:"datalog"`
	Heartbeats               bool          `yaml:"heartbeats"`
	PublishHoldingsOnConnect bool          `yaml:"publish_holdings_on_connect"`
	ReadTimeout              time.Duration `yaml:"read_timeout"`
	ShortRead                time.Duration `yaml:"short_read"`
	LongRead                 time.Duration `yaml:"long_read"`
	// HoldRead is how often the holding (settings) registers are read. -1s disables.
	HoldRead time.Duration `yaml:"hold_read"`
	// Model selects the register mapping, case-insensitively. Empty is the
	// default (18kPV-style) mapping; "6000xp" is the EG4 6000XP.
	Model               string        `yaml:"model"`
	HealthCheckDuration time.Duration `yaml:"health_check_duration"`
	// CRCMode is "log" (default) or "drop"; see README.
	CRCMode string `yaml:"crc_mode"`
	// ClockSync keeps the inverter clock (holding registers 12-14) in sync with the host.
	ClockSync ClockSyncConfig `yaml:"clock_sync"`
}

// ClockSyncConfig configures inverter clock synchronisation.
type ClockSyncConfig struct {
	// Enabled turns on writing the clock. Off by default.
	Enabled bool `yaml:"enabled"`
	// Timezone is the IANA zone the inverter clock is kept in. Required when
	// enabled; also enables the drift gauge when set.
	Timezone string `yaml:"timezone"`
	// MaxDrift is the allowed difference before a sync; 60s when 0.
	MaxDrift time.Duration `yaml:"max_drift"`
}

type MessageJson map[string]interface{}
