package retle

import (
	"context"
	"time"
)

// Default ExpTimer option
const (
	DefaultInitialInterval = 500 * time.Millisecond
	DefaultMultiplier      = 1.5
)

// RetryFunc is a type of retry function
// first return value is bool that represent whether to retry
// When first return value is false, second return value is used
type RetryFunc func() (bool, error)

// ExpTimer is a type to retry using exponential backoff algorithm
type ExpTimer struct {
	interval   time.Duration
	multiplier float64
}

// NewExpTimer return a ExpTimer instance
func NewExpTimer(interval time.Duration, multiplier float64) *ExpTimer {
	return &ExpTimer{
		interval:   interval,
		multiplier: multiplier,
	}
}

// DefaultExpTimer return a ExpTimer instance to use default option
func DefaultExpTimer() *ExpTimer {
	return NewExpTimer(DefaultInitialInterval, DefaultMultiplier)
}

// NextDuration return a next backoff duration
func (e *ExpTimer) NextDuration() time.Duration {
	beforeInterval := e.interval
	e.interval = time.Duration(float64(e.interval) * e.multiplier)
	return beforeInterval
}

// Sleep will sleep during NextDuration
func (e *ExpTimer) Sleep() {
	time.Sleep(e.NextDuration())
}

// Retry calls retryFunc repeatedly according to exponential backoff algorithm
func (e *ExpTimer) Retry(ctx context.Context, retryFunc RetryFunc) error {
	for {
		isRetry, err := retryFunc()
		if !isRetry {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			e.Sleep()
		}
	}
}

// Retry calls retryFunc repeatedly according to exponential backoff algorithm
// using DefaultExpTimer
func Retry(ctx context.Context, retryFunc RetryFunc) error {
	e := DefaultExpTimer()
	return e.Retry(ctx, retryFunc)
}
