package models

import (
	"time"
)

type Config struct {
	LogLevel     string           `yaml:"loglevel"`
	MQTTConfig   MQTTConfig       `yaml:"mqtt"`
	StatsDConfig StatsDConfig     `yaml:"statsd"`
	Prometheus   PrometheusConfig `yaml:"prometheus"`
	Inverters    []LXPConfig      `yaml:"inverters"`
	OpenEVSE     OpenEVSEConfig   `yaml:"openEVSE"`
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
	HoldRead            time.Duration `yaml:"hold_read"`
	HealthCheckDuration time.Duration `yaml:"health_check_duration"`
	// CRCMode is "log" (default) or "drop"; see README.
	CRCMode string `yaml:"crc_mode"`
}

type MessageJson map[string]interface{}
