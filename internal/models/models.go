package models

import (
	"time"
)

type Config struct {
	MQTTConfig   MQTTConfig     `yaml:"mqtt"`
	StatsDConfig StatsDConfig   `yaml:"statsd"`
	Inverters    []LXPConfig    `yaml:"inverters"`
	OpenEVSE     OpenEVSEConfig `yaml:"openEVSE"`
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
}

type MessageJson map[string]interface{}
