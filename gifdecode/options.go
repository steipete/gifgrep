package gifdecode

import "time"

const (
	defaultMaxFrames = 60
	defaultMaxPixels = 40_000_000
	defaultMaxBytes  = int64(20 << 20)
)

const (
	defaultDelay = 80 * time.Millisecond
	minDelay     = 10 * time.Millisecond
	maxDelay     = 1 * time.Second
)

type Options struct {
	// MaxFrames limits returned frames; zero uses the default, negative disables the limit.
	MaxFrames    int
	MaxPixels    int
	MaxBytes     int64
	DefaultDelay time.Duration
	MinDelay     time.Duration
	MaxDelay     time.Duration
	StrictGIF    bool
}

func (o Options) withDefaults() Options {
	if o.MaxFrames == 0 {
		o.MaxFrames = defaultMaxFrames
	}
	if o.MaxPixels == 0 {
		o.MaxPixels = defaultMaxPixels
	}
	if o.MaxBytes == 0 {
		o.MaxBytes = defaultMaxBytes
	}
	if o.DefaultDelay == 0 {
		o.DefaultDelay = defaultDelay
	}
	if o.MinDelay == 0 {
		o.MinDelay = minDelay
	}
	if o.MaxDelay == 0 {
		o.MaxDelay = maxDelay
	}
	if o.MaxDelay < o.MinDelay {
		o.MaxDelay = o.MinDelay
	}
	return o
}

func DefaultOptions() Options {
	return Options{
		MaxFrames:    defaultMaxFrames,
		MaxPixels:    defaultMaxPixels,
		MaxBytes:     defaultMaxBytes,
		DefaultDelay: defaultDelay,
		MinDelay:     minDelay,
		MaxDelay:     maxDelay,
	}
}
