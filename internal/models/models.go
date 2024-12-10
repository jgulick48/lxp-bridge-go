package models

type Config struct {
	MQTTConfig   MQTTConfig   `yaml:"mqtt_config"`
	StatsDConfig StatsDConfig `yaml:"statsd_config"`
	Inverters    []LXPConfig  `yaml:"inverters"`
}

type MQTTConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type StatsDConfig struct {
	StatsServer string `yaml:"statsServer"`
}

type LXPConfig struct {
	Enabled                  bool   `yaml:"enabled"`
	Host                     string `yaml:"host"`
	Port                     int    `yaml:"port"`
	Serial                   string `yaml:"serial"`
	DataLog                  string `yaml:"dataLog"`
	Heartbeats               bool   `yaml:"heartbeats"`
	PublishHoldingsOnConnect bool   `yaml:"publish_holdings_on_connect"`
	ReadTimeout              int    `yaml:"read_timeout"`
	ShortRead                int    `yaml:"short_read"`
	LongRead                 int    `yaml:"long_read"`
}

type MessageJson map[string]interface{}
