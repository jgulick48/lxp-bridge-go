package registers

import (
	"fmt"
	"strconv"
	"strings"
)

// TimeOfDayPattern is the Home Assistant text entity pattern for "HH:MM".
const TimeOfDayPattern = `^([01]?[0-9]|2[0-3]):[0-5][0-9]$`

// DecodeTimeOfDay splits a raw time register: hour in the low byte, minute in
// the high byte (the opposite of the clock registers 12-14). ok is false when
// the hour is above 23 or the minute above 59.
func DecodeTimeOfDay(raw uint16) (hour, minute int, ok bool) {
	hour, minute = int(raw&0xff), int(raw>>8)
	return hour, minute, hour <= 23 && minute <= 59
}

// EncodeTimeOfDay packs hour and minute into a raw time register.
func EncodeTimeOfDay(hour, minute int) (uint16, error) {
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("time %d:%d is not a valid time of day", hour, minute)
	}
	return uint16(hour) | uint16(minute)<<8, nil
}

// FormatTimeOfDay renders a raw time register as zero-padded "HH:MM".
func FormatTimeOfDay(raw uint16) (string, bool) {
	h, m, ok := DecodeTimeOfDay(raw)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%02d:%02d", h, m), true
}

// ParseTimeOfDay parses "HH:MM" or "H:MM" (surrounding spaces ignored) into a
// raw time register value.
func ParseTimeOfDay(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	hs, ms, found := strings.Cut(s, ":")
	if !found || len(hs) < 1 || len(hs) > 2 || len(ms) != 2 || !isDigits(hs) || !isDigits(ms) {
		return 0, fmt.Errorf("time %q is not HH:MM", s)
	}
	h, _ := strconv.Atoi(hs)
	m, _ := strconv.Atoi(ms)
	raw, err := EncodeTimeOfDay(h, m)
	if err != nil {
		return 0, fmt.Errorf("time %q is not a valid time of day", s)
	}
	return raw, nil
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}
