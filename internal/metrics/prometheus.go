package metrics

import (
	"bufio"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Prometheus is a Sink that keeps the latest values in memory and serves them
// in the Prometheus text exposition format (v0.0.4).
type Prometheus struct {
	staleAfter time.Duration
	now        func() time.Time

	mu       sync.Mutex
	gauges   map[string]*promSeries // keyed by name + label string
	counters map[string]*promSeries
}

type promSeries struct {
	name   string
	labels string // rendered {k="v",...} or ""
	value  float64
	set    time.Time
}

// NewPrometheus returns a sink whose gauges disappear from the output when
// they have not been updated for staleAfter (0 disables expiry).
func NewPrometheus(staleAfter time.Duration) *Prometheus {
	return &Prometheus{
		staleAfter: staleAfter,
		now:        time.Now,
		gauges:     make(map[string]*promSeries),
		counters:   make(map[string]*promSeries),
	}
}

// Gauge records the latest value.
func (p *Prometheus) Gauge(name string, value float64, tags []string, _ float64) error {
	name, labels := sanitizeName(name), renderLabels(tags)
	p.mu.Lock()
	defer p.mu.Unlock()
	key := name + labels
	s := p.gauges[key]
	if s == nil {
		s = &promSeries{name: name, labels: labels}
		p.gauges[key] = s
	}
	s.value = value
	s.set = p.now()
	return nil
}

// Count adds value to a running total exposed as <name>_total.
func (p *Prometheus) Count(name string, value int64, tags []string, _ float64) error {
	name, labels := sanitizeName(name)+"_total", renderLabels(tags)
	p.mu.Lock()
	defer p.mu.Unlock()
	key := name + labels
	s := p.counters[key]
	if s == nil {
		s = &promSeries{name: name, labels: labels}
		p.counters[key] = s
	}
	s.value += float64(value)
	return nil
}

// Close is a no-op.
func (p *Prometheus) Close() error { return nil }

// ServeHTTP writes the exposition output.
func (p *Prometheus) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	p.mu.Lock()
	now := p.now()
	var gauges, counters []promSeries
	for k, s := range p.gauges {
		if p.staleAfter > 0 && now.Sub(s.set) >= p.staleAfter {
			delete(p.gauges, k)
			continue
		}
		gauges = append(gauges, *s)
	}
	for _, s := range p.counters {
		counters = append(counters, *s)
	}
	p.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	bw := bufio.NewWriter(w)
	writeSeries(bw, gauges, "gauge")
	writeSeries(bw, counters, "counter")
	_ = bw.Flush()
}

func writeSeries(w *bufio.Writer, series []promSeries, typ string) {
	sort.Slice(series, func(i, j int) bool {
		if series[i].name != series[j].name {
			return series[i].name < series[j].name
		}
		return series[i].labels < series[j].labels
	})
	last := ""
	for _, s := range series {
		if s.name != last {
			_, _ = w.WriteString("# TYPE " + s.name + " " + typ + "\n")
			last = s.name
		}
		_, _ = w.WriteString(s.name + s.labels + " " + strconv.FormatFloat(s.value, 'g', -1, 64) + "\n")
	}
}

// sanitizeName replaces characters invalid in a metric name with '_'.
func sanitizeName(s string) string {
	if s == "" {
		return "_"
	}
	b := []byte(s)
	for i, c := range b {
		ok := c == '_' || c == ':' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (i > 0 && c >= '0' && c <= '9')
		if !ok {
			b[i] = '_'
		}
	}
	return string(b)
}

var labelEscaper = strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n")

// renderLabels turns "k:v" tags into a sorted {k="v",...} string; tags without
// a colon are ignored. Label names cannot contain ':' so it becomes '_'.
func renderLabels(tags []string) string {
	type kv struct{ k, v string }
	var kvs []kv
	for _, t := range tags {
		k, v, ok := strings.Cut(t, ":")
		if !ok {
			continue
		}
		kvs = append(kvs, kv{strings.ReplaceAll(sanitizeName(k), ":", "_"), v})
	}
	if len(kvs) == 0 {
		return ""
	}
	sort.SliceStable(kvs, func(i, j int) bool { return kvs[i].k < kvs[j].k })
	var b strings.Builder
	b.WriteByte('{')
	for i, e := range kvs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(e.k + `="` + labelEscaper.Replace(e.v) + `"`)
	}
	b.WriteByte('}')
	return b.String()
}
