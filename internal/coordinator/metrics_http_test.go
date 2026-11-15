package coordinator

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/metrics"
	"github.com/jgulick48/lxp-bridge-go/internal/modbus"
	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildProm(t *testing.T, prom *metrics.Prometheus) *client {
	t.Helper()
	var sink metrics.Sink = metrics.Nop{}
	if prom != nil {
		sink = metrics.Multi{sink, prom}
	}
	c, err := build(models.Config{MQTTConfig: models.MQTTConfig{NameSpace: "lxp"}}, deps{
		mqtt: fakeMQTT{}, metrics: sink, prom: prom,
		newModbus: func(models.LXPConfig, modbus.ParserCallback, modbus.Observer) (modbus.Client, error) {
			return &fakeModbus{}, nil
		},
	})
	require.NoError(t, err)
	return c
}

func TestMetricsEndpoint(t *testing.T) {
	c := buildProm(t, metrics.NewPrometheus(0))
	metrics.Gauge(c.metrics, "lxp_p_battery", []string{"deploymentID:BA1"}, -5)
	metrics.Count(c.metrics, "lxp_bridge_frames", []string{"deploymentID:BA1", "result:ok"}, 1)

	mux := http.NewServeMux()
	mux.Handle("/health", c.HealthHandler())
	mux.Handle("/metrics", c.MetricsHandler())

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "lxp_p_battery{deploymentID=\"BA1\"} -5\n")
	assert.Contains(t, rec.Body.String(), "lxp_bridge_frames_total{deploymentID=\"BA1\",result=\"ok\"} 1\n")

	// No inverters configured, so the probe is healthy; unchanged by metrics.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestMetricsEndpointDisabled(t *testing.T) {
	c := buildProm(t, nil)
	rec := httptest.NewRecorder()
	c.MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func scrape(t *testing.T, c *client) string {
	t.Helper()
	rec := httptest.NewRecorder()
	c.MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return rec.Body.String()
}

func TestOfflineInverterSeriesDropped(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	prom := metrics.NewPrometheus(0)
	mods := map[string]*fakeModbus{"UP": {connected: true}, "DOWN": {connected: true}}
	c, err := build(models.Config{
		MQTTConfig: models.MQTTConfig{NameSpace: "lxp"},
		Inverters:  []models.LXPConfig{{Enabled: true, DataLog: "UP"}, {Enabled: true, DataLog: "DOWN"}},
	}, deps{
		mqtt: fakeMQTT{}, metrics: metrics.Multi{metrics.Nop{}, prom}, prom: prom,
		now: func() time.Time { return now },
		newModbus: func(cfg models.LXPConfig, _ modbus.ParserCallback, _ modbus.Observer) (modbus.Client, error) {
			return mods[cfg.DataLog], nil
		},
	})
	require.NoError(t, err)
	now = now.Add(time.Minute)
	mods["UP"].last, mods["DOWN"].last = now, now
	for _, id := range []string{"UP", "DOWN"} {
		metrics.Gauge(c.metrics, "lxp_p_load", []string{"deploymentID:" + id}, 1000)
		metrics.Gauge(c.metrics, "lxp_hold_value", []string{"deploymentID:" + id}, 7) // set once, like a hold read
		metrics.Count(c.metrics, "lxp_bridge_frames", []string{"deploymentID:" + id}, 1)
	}

	// DOWN disconnects; ten minutes pass with no new readings at all.
	mods["DOWN"].connected = false
	now = now.Add(time.Minute)
	mods["UP"].last = now
	c.dropOfflineSeries()
	c.emitHealthGauges()

	body := scrape(t, c)
	assert.Contains(t, body, `lxp_p_load{deploymentID="UP"} 1000`)
	assert.Contains(t, body, `lxp_hold_value{deploymentID="UP"} 7`, "an online inverter keeps slow-changing values")
	assert.NotContains(t, body, `lxp_p_load{deploymentID="DOWN"}`)
	assert.NotContains(t, body, `lxp_hold_value{deploymentID="DOWN"}`)
	assert.Contains(t, body, `lxp_bridge_connected{deploymentID="DOWN"} 0`, "health gauges are re-added")
	assert.Contains(t, body, `lxp_bridge_frames_total{deploymentID="DOWN"} 1`, "counters are kept")
}
