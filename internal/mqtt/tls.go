package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/jgulick48/lxp-bridge-go/internal/models"
)

const (
	defaultPort    = 1883
	defaultTLSPort = 8883
)

// brokerURL returns the paho broker URL. A zero port defaults to 8883 with
// TLS and 1883 without.
func brokerURL(c models.MQTTConfig) string {
	port := c.Port
	scheme := "tcp"
	if c.TLS.Enabled {
		scheme = "ssl"
	}
	if port == 0 {
		port = defaultPort
		if c.TLS.Enabled {
			port = defaultTLSPort
		}
	}
	return scheme + "://" + net.JoinHostPort(c.Host, strconv.Itoa(port))
}

// buildTLSConfig turns the mqtt.tls settings into a *tls.Config for host.
func buildTLSConfig(c models.MQTTTLSConfig, host string) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         host,
		InsecureSkipVerify: c.InsecureSkipVerify, //nolint:gosec // explicit opt-in
	}
	if c.ServerName != "" {
		cfg.ServerName = c.ServerName
	}
	if c.CAFile != "" {
		pem, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("mqtt.tls: ca_file %s: %w", c.CAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("mqtt.tls: ca_file %s: no certificates found", c.CAFile)
		}
		cfg.RootCAs = pool
	}
	switch {
	case c.CertFile != "" && c.KeyFile != "":
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("mqtt.tls: client certificate %s / %s: %w", c.CertFile, c.KeyFile, err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	case c.CertFile != "":
		return nil, errors.New("mqtt.tls: cert_file is set but key_file is not")
	case c.KeyFile != "":
		return nil, errors.New("mqtt.tls: key_file is set but cert_file is not")
	}
	return cfg, nil
}

// plaintextCredentialsRisk reports whether credentials would cross a
// non-local network unencrypted. Hostnames other than "localhost" are not
// resolved and never trigger it.
func plaintextCredentialsRisk(c models.MQTTConfig) bool {
	if c.TLS.Enabled || (c.Username == "" && c.Password == "") {
		return false
	}
	ip := net.ParseIP(c.Host)
	if ip == nil {
		return false
	}
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}
