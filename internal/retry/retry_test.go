package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

var retriableErr = errors.New("transient")
var fatalErr = errors.New("fatal")

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
		return fatalErr
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
			return retriableErr
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
	return errors.Is(err, retriableErr)
}

func TestDoExhaustsRetries(t *testing.T) {
	old := delays
	delays = []time.Duration{0, 0, 0}
	defer func() { delays = old }()
	calls := 0
	ctx := context.Background()
	err := Do(ctx, isRetriable, func() error {
		calls++
		return retriableErr
	})
	if !errors.Is(err, retriableErr) {
		t.Fatalf("expected retriableErr, got %v", err)
	}
	if calls != 4 {
		t.Errorf("expected 4 calls, got %d", calls)
	}
}
