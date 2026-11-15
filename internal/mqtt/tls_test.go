package mqtt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jgulick48/lxp-bridge-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPKI struct {
	dir     string
	caFile  string
	ca      *x509.Certificate
	caKey   *ecdsa.PrivateKey
	counter int64
}

func newPKI(t *testing.T) *testPKI {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	ca, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	p := &testPKI{dir: t.TempDir(), ca: ca, caKey: key, counter: 1}
	p.caFile = filepath.Join(p.dir, "ca.pem")
	require.NoError(t, os.WriteFile(p.caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	return p
}

// issue creates a CA-signed leaf, returning the tls.Certificate and the
// PEM files it was written to.
func (p *testPKI) issue(t *testing.T, name string, server bool) (tls.Certificate, string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	p.counter++
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(p.counter),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if server {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.DNSNames = []string{"localhost"}
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, p.ca, &key.PublicKey, p.caKey)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	certFile := filepath.Join(p.dir, name+".crt")
	keyFile := filepath.Join(p.dir, name+".key")
	require.NoError(t, os.WriteFile(certFile, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0o600))
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	return pair, certFile, keyFile
}

func TestBuildTLSConfig(t *testing.T) {
	p := newPKI(t)
	_, certFile, keyFile := p.issue(t, "client", false)

	t.Run("defaults", func(t *testing.T) {
		cfg, err := buildTLSConfig(models.MQTTTLSConfig{Enabled: true}, "broker.example")
		require.NoError(t, err)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Equal(t, "broker.example", cfg.ServerName)
		assert.Nil(t, cfg.RootCAs)
		assert.False(t, cfg.InsecureSkipVerify)
		assert.Empty(t, cfg.Certificates)
	})
	t.Run("ca file loads", func(t *testing.T) {
		cfg, err := buildTLSConfig(models.MQTTTLSConfig{CAFile: p.caFile}, "h")
		require.NoError(t, err)
		assert.NotNil(t, cfg.RootCAs)
	})
	t.Run("bad ca file", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "bad.pem")
		require.NoError(t, os.WriteFile(bad, []byte("not a cert"), 0o600))
		_, err := buildTLSConfig(models.MQTTTLSConfig{CAFile: bad}, "h")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no certificates found")
		assert.Contains(t, err.Error(), bad)
	})
	t.Run("missing ca file", func(t *testing.T) {
		_, err := buildTLSConfig(models.MQTTTLSConfig{CAFile: "/nonexistent/ca.pem"}, "h")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ca_file")
	})
	t.Run("cert without key", func(t *testing.T) {
		_, err := buildTLSConfig(models.MQTTTLSConfig{CertFile: certFile}, "h")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key_file")
	})
	t.Run("key without cert", func(t *testing.T) {
		_, err := buildTLSConfig(models.MQTTTLSConfig{KeyFile: keyFile}, "h")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cert_file")
	})
	t.Run("client pair loads", func(t *testing.T) {
		cfg, err := buildTLSConfig(models.MQTTTLSConfig{CertFile: certFile, KeyFile: keyFile}, "h")
		require.NoError(t, err)
		assert.Len(t, cfg.Certificates, 1)
	})
	t.Run("mismatched pair errors", func(t *testing.T) {
		_, otherCert, _ := p.issue(t, "other", false)
		_, err := buildTLSConfig(models.MQTTTLSConfig{CertFile: otherCert, KeyFile: keyFile}, "h")
		require.Error(t, err)
	})
	t.Run("server name override", func(t *testing.T) {
		cfg, err := buildTLSConfig(models.MQTTTLSConfig{ServerName: "other.name"}, "10.0.0.1")
		require.NoError(t, err)
		assert.Equal(t, "other.name", cfg.ServerName)
	})
	t.Run("insecure skip verify", func(t *testing.T) {
		cfg, err := buildTLSConfig(models.MQTTTLSConfig{InsecureSkipVerify: true}, "h")
		require.NoError(t, err)
		assert.True(t, cfg.InsecureSkipVerify)
	})
}

func TestBrokerURL(t *testing.T) {
	assert.Equal(t, "tcp://h:1883", brokerURL(models.MQTTConfig{Host: "h"}))
	assert.Equal(t, "ssl://h:8883", brokerURL(models.MQTTConfig{Host: "h", TLS: models.MQTTTLSConfig{Enabled: true}}))
	assert.Equal(t, "tcp://h:1884", brokerURL(models.MQTTConfig{Host: "h", Port: 1884}))
	assert.Equal(t, "ssl://h:8884", brokerURL(models.MQTTConfig{Host: "h", Port: 8884, TLS: models.MQTTTLSConfig{Enabled: true}}))
	assert.Equal(t, "tcp://[::1]:1883", brokerURL(models.MQTTConfig{Host: "::1"}))
}

func TestPlaintextCredentialsRisk(t *testing.T) {
	risk := func(host, user string, tlsOn bool) bool {
		return plaintextCredentialsRisk(models.MQTTConfig{Host: host, Username: user, TLS: models.MQTTTLSConfig{Enabled: tlsOn}})
	}
	assert.True(t, risk("8.8.8.8", "u", false))
	assert.False(t, risk("8.8.8.8", "", false))
	assert.False(t, risk("8.8.8.8", "u", true))
	assert.False(t, risk("192.168.1.10", "u", false))
	assert.False(t, risk("127.0.0.1", "u", false))
	assert.False(t, risk("localhost", "u", false))
	assert.False(t, risk("broker.example.com", "u", false))
}

func TestNewClientRejectsBadTLS(t *testing.T) {
	_, err := NewClient(models.MQTTConfig{Host: "h", TLS: models.MQTTTLSConfig{Enabled: true, CAFile: "/nonexistent"}}, false, &BaseMessageCallback{}, nil)
	require.Error(t, err)
	// Ignored when TLS is off or MQTT is disabled.
	_, err = NewClient(models.MQTTConfig{Host: "h", TLS: models.MQTTTLSConfig{CAFile: "/nonexistent"}}, false, &BaseMessageCallback{}, nil)
	require.NoError(t, err)
	_, err = NewClient(models.MQTTConfig{TLS: models.MQTTTLSConfig{Enabled: true, CAFile: "/nonexistent"}}, false, &BaseMessageCallback{}, nil)
	require.NoError(t, err)
}

type handshakeResult struct {
	peers []*x509.Certificate
	err   error
}

// tlsBroker accepts one TLS connection requiring a client certificate and
// reports the handshake outcome.
func tlsBroker(t *testing.T, p *testPKI) (port int, result chan handshakeResult) {
	t.Helper()
	serverCert, _, _ := p.issue(t, "server", true)
	pool := x509.NewCertPool()
	pool.AddCert(p.ca)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS12,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	result = make(chan handshakeResult, 4)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				tc := conn.(*tls.Conn)
				_ = tc.SetDeadline(time.Now().Add(3 * time.Second))
				if err := tc.Handshake(); err != nil {
					result <- handshakeResult{err: err}
					return
				}
				result <- handshakeResult{peers: tc.ConnectionState().PeerCertificates}
				buf := make([]byte, 64)
				_, _ = tc.Read(buf) // MQTT CONNECT, ignored
			}()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, result
}

func TestTLSHandshakeWithClientCert(t *testing.T) {
	p := newPKI(t)
	_, certFile, keyFile := p.issue(t, "client", false)
	port, result := tlsBroker(t, p)

	c, err := NewClient(models.MQTTConfig{
		Host: "127.0.0.1", Port: port, ClientName: "t",
		TLS: models.MQTTTLSConfig{Enabled: true, CAFile: p.caFile, CertFile: certFile, KeyFile: keyFile},
	}, false, &BaseMessageCallback{}, nil)
	require.NoError(t, err)
	c.Connect()
	defer c.Close()

	select {
	case r := <-result:
		require.NoError(t, r.err)
		require.NotEmpty(t, r.peers)
		assert.Equal(t, "client", r.peers[0].Subject.CommonName)
	case <-time.After(4 * time.Second):
		t.Fatal("no TLS handshake seen")
	}
}

func TestTLSHandshakeFailsWithoutCA(t *testing.T) {
	p := newPKI(t)
	_, certFile, keyFile := p.issue(t, "client", false)
	port, result := tlsBroker(t, p)

	c, err := NewClient(models.MQTTConfig{
		Host: "127.0.0.1", Port: port, ClientName: "t",
		// No ca_file: the test CA is not in the system roots.
		TLS: models.MQTTTLSConfig{Enabled: true, CertFile: certFile, KeyFile: keyFile},
	}, false, &BaseMessageCallback{}, nil)
	require.NoError(t, err)
	c.Connect()
	defer c.Close()

	select {
	case r := <-result:
		require.Error(t, r.err, "handshake must not complete when the broker cert is untrusted")
	case <-time.After(4 * time.Second):
		t.Fatal("broker saw no connection attempt")
	}
}
