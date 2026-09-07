package retry

import (
	"context"
	"time"
)

const (
	delay   int = 2
	retries int = 3
)

func Do(ctx context.Context, isRetriable func(error) bool, fn func() error) error {
	var err error
	for i := range retries {
		err = fn()
		if err != nil && !isRetriable(err) {
			return err
		}
		if err != nil && i == retries-1 {
			return err
		}
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(1+(delay*i)) * time.Second):
		}
	}
	return err
}
