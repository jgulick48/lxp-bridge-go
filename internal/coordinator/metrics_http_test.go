package coordinator

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
