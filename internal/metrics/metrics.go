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
func New(addr string) (Sink, error) {
	if addr == "" {
		return Nop{}, nil
	}
	return statsd.New(addr)
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
