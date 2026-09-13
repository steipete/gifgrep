package app

import (
	"encoding"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
)

type DurationValue time.Duration

var _ encoding.TextUnmarshaler = (*DurationValue)(nil)

func (d *DurationValue) UnmarshalText(text []byte) error {
	raw := strings.TrimSpace(string(text))
	if raw == "" {
		return errors.New("empty duration")
	}
	if parsed, err := time.ParseDuration(raw); err == nil {
		if parsed < 0 {
			return errors.New("negative duration")
		}
		*d = DurationValue(parsed)
		return nil
	}
	if secs, err := strconv.ParseFloat(raw, 64); err == nil {
		if math.IsNaN(secs) || math.IsInf(secs, 0) || secs >= float64(math.MaxInt64)/float64(time.Second) {
			return errors.New("duration out of range")
		}
		if secs < 0 {
			return errors.New("negative duration")
		}
		*d = DurationValue(time.Duration(secs * float64(time.Second)))
		return nil
	}
	return errors.New("invalid duration")
}
