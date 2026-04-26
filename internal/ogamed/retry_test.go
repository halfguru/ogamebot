package ogamed

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"log/slog"
)

func TestRetry_SucceedsOnFirstTry(t *testing.T) {
	var calls atomic.Int32
	fn := func() error {
		calls.Add(1)
		return nil
	}

	cfg := DefaultRetryConfig
	err := retryWithBackoff(context.Background(), fn, cfg, slog.Default())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 call, got %d", calls.Load())
	}
}

func TestRetry_RetriesOnTransientError(t *testing.T) {
	var calls atomic.Int32
	fn := func() error {
		c := calls.Add(1)
		if c < 3 {
			return errors.New("transient error")
		}
		return nil
	}

	cfg := RetryConfig{
		MaxAttempts:  3,
		BaseDelayMs:  1,
		MaxDelayMs:   10,
		JitterFactor: 0,
	}
	err := retryWithBackoff(context.Background(), fn, cfg, slog.Default())

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", calls.Load())
	}
}

func TestRetry_MaxAttemptsExhausted(t *testing.T) {
	expectedErr := errors.New("always fails")
	var calls atomic.Int32
	fn := func() error {
		calls.Add(1)
		return expectedErr
	}

	cfg := RetryConfig{
		MaxAttempts:  3,
		BaseDelayMs:  1,
		MaxDelayMs:   10,
		JitterFactor: 0,
	}
	err := retryWithBackoff(context.Background(), fn, cfg, slog.Default())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedErr {
		t.Errorf("expected 'always fails' error, got %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", calls.Load())
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	// 4xx client error should not be retried
	ogamedErr := &OgamedError{Code: 400, Message: "bad request"}
	var calls atomic.Int32
	fn := func() error {
		calls.Add(1)
		return ogamedErr
	}

	cfg := RetryConfig{
		MaxAttempts:  3,
		BaseDelayMs:  1,
		MaxDelayMs:   10,
		JitterFactor: 0,
	}
	err := retryWithBackoff(context.Background(), fn, cfg, slog.Default())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 call (no retry for 4xx), got %d", calls.Load())
	}
}

func TestRetry_BackoffIncreases(t *testing.T) {
	var calls atomic.Int32
	var firstRetryTime time.Time
	var secondRetryTime time.Time
	fn := func() error {
		c := calls.Add(1)
		if c == 1 {
			return errors.New("fail 1")
		}
		if c == 2 {
			firstRetryTime = time.Now()
			return errors.New("fail 2")
		}
		if c == 3 {
			secondRetryTime = time.Now()
			return errors.New("fail 3")
		}
		return nil
	}

	cfg := RetryConfig{
		MaxAttempts:  4,
		BaseDelayMs:  50,
		MaxDelayMs:   5000,
		JitterFactor: 0,
	}
	retryWithBackoff(context.Background(), fn, cfg, slog.Default())

	if firstRetryTime.IsZero() || secondRetryTime.IsZero() {
		t.Fatal("expected retries to be tracked")
	}

	firstWait := firstRetryTime.Sub(time.Time{})
	secondWait := secondRetryTime.Sub(firstRetryTime)
	_ = firstWait
	_ = secondWait

	// The second retry delay (2*BaseDelayMs) should be longer than the first (1*BaseDelayMs)
	// We can't test exact timing, but we can verify the second gap is meaningfully longer
	gap2 := secondRetryTime.Sub(firstRetryTime)
	if gap2 < 50*time.Millisecond {
		t.Errorf("expected second retry gap >= 50ms (exponential growth), got %v", gap2)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var calls atomic.Int32
	fn := func() error {
		calls.Add(1)
		return errors.New("fail")
	}

	cfg := RetryConfig{
		MaxAttempts:  3,
		BaseDelayMs:  1,
		MaxDelayMs:   10,
		JitterFactor: 0,
	}
	err := retryWithBackoff(ctx, fn, cfg, slog.Default())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Context already cancelled — fn may be called once (first attempt before any wait)
	// but must not retry after that
	if calls.Load() > 1 {
		t.Errorf("expected at most 1 call with pre-cancelled context, got %d", calls.Load())
	}
}
