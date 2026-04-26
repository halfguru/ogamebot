package ogamed

import (
	"context"
	"testing"
	"time"

	"github.com/user/ogame-bot/internal/config"
)

func TestRateLimiter_EnforcesMinDelay(t *testing.T) {
	cfg := config.RateLimitConfig{
		DefaultMinDelayMs: 50,
		DefaultMaxDelayMs: 50,
	}
	limiter := NewRateLimiter(cfg)

	start := time.Now()
	if err := limiter.Wait(context.Background(), "/test"); err != nil {
		t.Fatalf("first Wait failed: %v", err)
	}
	if err := limiter.Wait(context.Background(), "/test"); err != nil {
		t.Fatalf("second Wait failed: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < 50*time.Millisecond {
		t.Errorf("expected at least 50ms between two Wait calls, got %v", elapsed)
	}
}

func TestRateLimiter_PerEndpointOverride(t *testing.T) {
	cfg := config.RateLimitConfig{
		DefaultMinDelayMs: 50,
		DefaultMaxDelayMs: 50,
		EndpointOverrides: map[string]config.EndpointDelayConfig{
			"/slow": {MinMs: 100, MaxMs: 100},
		},
	}
	limiter := NewRateLimiter(cfg)

	start := time.Now()
	if err := limiter.Wait(context.Background(), "/slow"); err != nil {
		t.Fatalf("first Wait failed: %v", err)
	}
	if err := limiter.Wait(context.Background(), "/slow"); err != nil {
		t.Fatalf("second Wait failed: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Errorf("expected at least 100ms for /slow endpoint override, got %v", elapsed)
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	cfg := config.RateLimitConfig{
		DefaultMinDelayMs: 5000, // long delay so context cancel kicks in
		DefaultMaxDelayMs: 5000,
	}
	limiter := NewRateLimiter(cfg)

	// First call sets lastCall, second should try to wait but context cancelled
	ctx, cancel := context.WithCancel(context.Background())
	// Call once to set lastCall
	if err := limiter.Wait(ctx, "/test"); err != nil {
		t.Fatalf("first Wait failed: %v", err)
	}

	// Cancel context before second call
	cancel()
	err := limiter.Wait(ctx, "/test")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
