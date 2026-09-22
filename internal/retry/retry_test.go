package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errRetriable = errors.New("retriable")
var errFatal = errors.New("fatal")

func TestDoSuccess(t *testing.T) {

	calls := 0
	ctx := context.Background()
	err := Do(ctx, isRetriable, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDoNonRetriableStops(t *testing.T) {
	calls := 0
	ctx := context.Background()
	err := Do(ctx, isRetriable, func() error {
		calls++
		return errFatal
	})
	if err == nil {
		t.Fatalf("Do not failed: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDoRetriableEventuallySucceeds(t *testing.T) {
	calls := 0
	old := delays
	delays = []time.Duration{0, 0, 0}
	defer func() { delays = old }()
	ctx := context.Background()
	err := Do(ctx, isRetriable, func() error {
		calls++
		if calls < 3 {
			return errRetriable
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 call, got %d", calls)
	}
}

func isRetriable(err error) bool {
	return errors.Is(err, errRetriable)
}

func TestDoExhaustsRetries(t *testing.T) {
	old := delays
	delays = []time.Duration{0, 0, 0}
	defer func() { delays = old }()
	calls := 0
	ctx := context.Background()
	err := Do(ctx, isRetriable, func() error {
		calls++
		return errRetriable
	})
	if !errors.Is(err, errRetriable) {
		t.Fatalf("expected errRetriable, got %v", err)
	}
	if calls != 4 {
		t.Errorf("expected 4 calls, got %d", calls)
	}
}
