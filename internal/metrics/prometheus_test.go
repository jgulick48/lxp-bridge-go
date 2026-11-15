package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scrape(p *Prometheus) (string, string) {
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	return rec.Body.String(), rec.Header().Get("Content-Type")
}

func TestPrometheusExposition(t *testing.T) {
	p := NewPrometheus(time.Minute)
	tags := []string{"deploymentID:BA1"}
	Gauge(p, "lxp_p_pv", tags, 2500)
	Gauge(p, "lxp_p_battery", tags, -1185)
	Gauge(p, "lxp_p_battery", []string{"deploymentID:BA0"}, 1.5)
	ok := []string{"deploymentID:BA1", "result:ok"}
	Count(p, "lxp_bridge_frames", ok, 1)
	Count(p, "lxp_bridge_frames", ok, 2)
	Count(p, "lxp_bridge_frames", []string{"deploymentID:BA1", "result:crc_error"}, 1)

	body, ct := scrape(p)
	assert.Equal(t, "text/plain; version=0.0.4; charset=utf-8", ct)
	assert.Equal(t, `# TYPE lxp_p_battery gauge
lxp_p_battery{deploymentID="BA0"} 1.5
lxp_p_battery{deploymentID="BA1"} -1185
# TYPE lxp_p_pv gauge
lxp_p_pv{deploymentID="BA1"} 2500
# TYPE lxp_bridge_frames_total counter
lxp_bridge_frames_total{deploymentID="BA1",result="crc_error"} 1
lxp_bridge_frames_total{deploymentID="BA1",result="ok"} 3
`, body)
}

func TestPrometheusSanitizeAndEscape(t *testing.T) {
	p := NewPrometheus(0)
	require.NoError(t, p.Gauge("9bad-name.x", 1, []string{"a-b:va\"l\\ue\nx", "notag", "k:v:w"}, 1))
	body, _ := scrape(p)
	assert.Equal(t, "# TYPE _bad_name_x gauge\n_bad_name_x{a_b=\"va\\\"l\\\\ue\\nx\",k=\"v:w\"} 1\n", body)
}

func TestPrometheusStale(t *testing.T) {
	p := NewPrometheus(5 * time.Minute)
	now := time.Unix(1000, 0)
	p.now = func() time.Time { return now }
	Gauge(p, "g", nil, 1)
	Count(p, "c", nil, 1)
	now = now.Add(4 * time.Minute)
	body, _ := scrape(p)
	assert.Contains(t, body, "g 1\n")
	now = now.Add(2 * time.Minute)
	body, _ = scrape(p)
	assert.NotContains(t, body, "gauge")
	assert.Contains(t, body, "c_total 1\n")
	Gauge(p, "g", nil, 2)
	body, _ = scrape(p)
	assert.Contains(t, body, "g 2\n")
}

func TestPrometheusConcurrent(t *testing.T) {
	p := NewPrometheus(time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				Gauge(p, "g", []string{fmt.Sprintf("id:%d", i)}, float64(j))
				Count(p, "c", nil, 1)
				scrape(p)
			}
		}(i)
	}
	wg.Wait()
	body, _ := scrape(p)
	assert.Contains(t, body, "c_total 1600\n")
}

type errSink struct {
	Nop
	err   error
	calls int
}

func (e *errSink) Gauge(string, float64, []string, float64) error { e.calls++; return e.err }
func (e *errSink) Count(string, int64, []string, float64) error   { e.calls++; return e.err }
func (e *errSink) Close() error                                   { e.calls++; return e.err }

func TestMultiFansOutAndReturnsFirstError(t *testing.T) {
	e1, e2 := errors.New("one"), errors.New("two")
	a, b, c := &errSink{err: e1}, &errSink{err: e2}, &errSink{}
	m := Multi{a, b, c}
	assert.Equal(t, e1, m.Gauge("g", 1, nil, 1))
	assert.Equal(t, e1, m.Count("c", 1, nil, 1))
	assert.Equal(t, e1, m.Close())
	assert.Equal(t, 3, a.calls)
	assert.Equal(t, 3, b.calls)
	assert.Equal(t, 3, c.calls)
	assert.NoError(t, Multi{c}.Gauge("g", 1, nil, 1))
}

func TestPrometheusDropTag(t *testing.T) {
	p := NewPrometheus(0)
	_ = p.Gauge("a", 1, []string{"deploymentID:X"}, 1)
	_ = p.Gauge("a", 2, []string{"deploymentID:Y"}, 1)
	_ = p.Count("c", 1, []string{"deploymentID:X"}, 1)
	p.DropTag("deploymentID:X")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	assert.NotContains(t, body, `a{deploymentID="X"}`)
	assert.Contains(t, body, `a{deploymentID="Y"} 2`)
	assert.Contains(t, body, `c_total{deploymentID="X"} 1`)
}
