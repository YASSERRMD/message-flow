package providers

import (
	"context"
	"errors"
	"time"
)

type Retrier struct {
	Attempts int
	Delay    time.Duration
}

func (r Retrier) Do(ctx context.Context, fn func() error) error {
	attempts := r.Attempts
	if attempts <= 0 {
		attempts = 1
	}
	delay := r.Delay
	if delay <= 0 {
		delay = 300 * time.Millisecond
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("retry failed")
	}
	return lastErr
}
