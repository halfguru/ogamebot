package ogamed

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/user/ogame-bot/internal/config"
)

// RateLimiter enforces configurable random delays between API calls
// to prevent detection by OGame's anti-bot systems.
// It is thread-safe; a single instance should be shared across all Client calls.
type RateLimiter struct {
	mu       sync.Mutex
	lastCall time.Time
	config   config.RateLimitConfig
}

// NewRateLimiter creates a rate limiter with the given configuration.
func NewRateLimiter(cfg config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{config: cfg}
}

// Wait blocks until the minimum delay since the last API call has elapsed.
// The delay includes random jitter within [minDelay, maxDelay] for anti-detection.
// Per-endpoint overrides take precedence when available.
func (r *RateLimiter) Wait(ctx context.Context, endpoint string) error {
	r.mu.Lock()

	override, hasOverride := r.config.EndpointOverrides[endpoint]
	minDelay := r.config.DefaultMinDelayMs
	maxDelay := r.config.DefaultMaxDelayMs
	if hasOverride {
		minDelay = override.MinMs
		maxDelay = override.MaxMs
	}

	// Random delay within [minDelay, maxDelay]
	spread := maxDelay - minDelay + 1
	jitteredDelay := time.Duration(minDelay+rand.Intn(spread)) * time.Millisecond
	elapsed := time.Since(r.lastCall)
	waitTime := jitteredDelay - elapsed

	r.mu.Unlock()

	if waitTime > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
		}
	}

	r.mu.Lock()
	r.lastCall = time.Now()
	r.mu.Unlock()
	return nil
}
