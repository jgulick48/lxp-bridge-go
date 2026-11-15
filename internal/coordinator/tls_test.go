package coordinator

import (
	"testing"

	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/require"
)

func TestBuildFailsOnBadMQTTTLSConfig(t *testing.T) {
	_, err := build(models.Config{
		MQTTConfig: models.MQTTConfig{Host: "broker", TLS: models.MQTTTLSConfig{Enabled: true, CAFile: "/nonexistent/ca.pem"}},
	}, deps{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "mqtt.tls: ca_file /nonexistent/ca.pem")
}
