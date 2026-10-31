package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetLogLevel(t *testing.T) {
	defer logrus.SetLevel(logrus.InfoLevel)
	for in, want := range map[string]logrus.Level{"": logrus.InfoLevel, "debug": logrus.DebugLevel, "WARN": logrus.WarnLevel, "error": logrus.ErrorLevel} {
		require.NoError(t, setLogLevel(in))
		assert.Equal(t, want, logrus.GetLevel(), in)
	}
	assert.Error(t, setLogLevel("loud"))
	assert.Equal(t, logrus.InfoLevel, logrus.GetLevel())
}

func TestLoadClientConfigProductionKeys(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yml")
	require.NoError(t, os.WriteFile(p, []byte(`loglevel: debug
inverters:
  - enabled: true
    host: 10.0.2.10
    port: 8000
    datalog: BA31101550
    serial: 3192670065
    read_timeout: 300s
    short_read: 30s
    long_read: -1s
    health_check_duration: 3m
    publish_holdings_on_connect: true
    crc_mode: drop
mqtt:
  host: ""
  namespace: lxp
`), 0o600))
	c, err := LoadClientConfig(p)
	require.NoError(t, err)
	assert.Equal(t, "debug", c.LogLevel)
	require.Len(t, c.Inverters, 1)
	inv := c.Inverters[0]
	assert.Equal(t, 3*time.Minute, inv.HealthCheckDuration)
	assert.Equal(t, -time.Second, inv.LongRead)
	assert.True(t, inv.PublishHoldingsOnConnect)
	assert.Equal(t, "drop", inv.CRCMode)

	_, err = LoadClientConfig(filepath.Join(t.TempDir(), "missing.yml"))
	assert.Error(t, err)
}
