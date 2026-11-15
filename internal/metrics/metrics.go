// Package metrics wraps DogStatsD behind a small injectable interface.
package metrics

import (
	"fmt"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/sirupsen/logrus"
)

// Sink is the subset of the DogStatsD client the bridge uses.
type Sink interface {
	Gauge(name string, value float64, tags []string, rate float64) error
	Count(name string, value int64, tags []string, rate float64) error
	Close() error
}

// Nop discards everything; used when statsd is not configured.
type Nop struct{}

func (Nop) Gauge(string, float64, []string, float64) error { return nil }
func (Nop) Count(string, int64, []string, float64) error   { return nil }
func (Nop) Close() error                                   { return nil }

// New returns a DogStatsD sink for addr, or Nop when addr is empty.
//
// Client-side aggregation is disabled: it keeps only the last gauge value per
// context per flush, which would swallow the zero reset that the statsd sink
// sends ahead of every negative gauge value.
func New(addr string) (Sink, error) {
	if addr == "" {
		return Nop{}, nil
	}
	c, err := statsd.New(addr, statsd.WithoutClientSideAggregation())
	if err != nil {
		return nil, err
	}
	return statsdSink{Sink: c}, nil
}

// statsdSink wraps a statsd client with the signed-gauge workaround.
//
// In the statsd protocol a signed gauge value ("name:-5|g") is a relative
// change, not an absolute value, so a negative reading is first preceded by a
// reset to 0. Without it, servers such as Telegraf subtract the value from the
// current gauge, which doubles the result when the same reading arrives twice
// within one flush. This only applies to statsd, so it lives here and not in
// the generic Gauge helper (a Prometheus scrape must never see the 0).
type statsdSink struct{ Sink }

func (s statsdSink) Gauge(name string, value float64, tags []string, rate float64) error {
	if value < 0 {
		if err := s.Sink.Gauge(name, 0, tags, rate); err != nil {
			return err
		}
	}
	return s.Sink.Gauge(name, value, tags, rate)
}

// Multi fans every call out to all sinks, returning the first error.
type Multi []Sink

func (m Multi) Gauge(name string, value float64, tags []string, rate float64) error {
	var first error
	for _, s := range m {
		if err := s.Gauge(name, value, tags, rate); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (m Multi) Count(name string, value int64, tags []string, rate float64) error {
	var first error
	for _, s := range m {
		if err := s.Count(name, value, tags, rate); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (m Multi) Close() error {
	var first error
	for _, s := range m {
		if err := s.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// FormatTag builds a key:value tag.
func FormatTag(key, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}

// Gauge sends a gauge, logging (at debug) rather than failing on errors.
func Gauge(s Sink, name string, tags []string, value float64) {
	if err := s.Gauge(name, value, tags, 1); err != nil {
		logrus.Debugf("Error sending metric %s: %s", name, err)
	}
}

// Count sends a counter increment, logging (at debug) rather than failing on errors.
func Count(s Sink, name string, tags []string, value int64) {
	if err := s.Count(name, value, tags, 1); err != nil {
		logrus.Debugf("Error sending metric %s: %s", name, err)
	}
}
