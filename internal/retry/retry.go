package retry

import (
	"context"
	"time"
)

const (
	Attempts int = 4
)

var delays = []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

func Do(ctx context.Context, isRetriable func(error) bool, fn func() error) error {
	var err error
	for i := range Attempts {
		err = fn()
		if err != nil && !isRetriable(err) {
			return err
		}
		if err != nil && i == Attempts-1 {
			return err
		}
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delays[i]):
		}
	}
	return err
}
