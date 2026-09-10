package retry

import (
	"context"
	"time"
)

const (
	Delay    int = 2
	Attempts int = 4
)

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
		case <-time.After(time.Duration(1+(Delay*i)) * time.Second):
		}
	}
	return err
}
