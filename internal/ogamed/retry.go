package ogamed

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"log/slog"
)

// RetryConfig controls the retry behavior for transient failures.
type RetryConfig struct {
	MaxAttempts  int     // default 3
	BaseDelayMs  int     // default 1000
	MaxDelayMs   int     // default 30000
	JitterFactor float64 // default 0.25 (±25% jitter)
}

// DefaultRetryConfig provides sensible defaults for retry behavior.
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	BaseDelayMs:  1000,
	MaxDelayMs:   30000,
	JitterFactor: 0.25,
}

// IsRetryable determines if an error should be retried.
// Non-retryable: 4xx client errors.
// Retryable: network errors, 5xx server errors.
var IsRetryable = func(err error) bool {
	if ogamedErr, ok := err.(*OgamedError); ok {
		return ogamedErr.Code >= 500 || ogamedErr.Code == 0
	}
	// Network errors and other transient failures are retryable
	return true
}

// retryWithBackoff executes fn with exponential backoff on transient failures.
// Non-retryable errors (e.g., 4xx client errors) are returned immediately.
func retryWithBackoff(ctx context.Context, fn func() error, cfg RetryConfig, log *slog.Logger) error {
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		if !IsRetryable(err) {
			return err
		}

		if attempt >= cfg.MaxAttempts {
			return err
		}

		// Exponential backoff with jitter
		baseDelay := math.Min(float64(cfg.BaseDelayMs)*math.Pow(2, float64(attempt-1)), float64(cfg.MaxDelayMs))
		jitter := baseDelay * cfg.JitterFactor * (rand.Float64()*2 - 1)
		delay := time.Duration(math.Max(0, baseDelay+jitter)) * time.Millisecond

		log.Warn("Request failed, retrying",
			"attempt", attempt,
			"maxAttempts", cfg.MaxAttempts,
			"delay_ms", delay.Milliseconds(),
			"error", err,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("retry loop exited unexpectedly")
}
