package espresso

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketRateLimiter_Allow(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, 10) // 10 req/sec, burst 10
	key := "test-key"

	// Should allow first 10 requests (burst)
	for i := 0; i < 10; i++ {
		if !limiter.Allow(key) {
			t.Fatalf("Request %d should be allowed (burst)", i+1)
		}
	}

	// 11th request should be denied (no more tokens)
	if limiter.Allow(key) {
		t.Fatal("Request 11 should be denied")
	}
}

func TestTokenBucketRateLimiter_Refill(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, 5) // 10 req/sec, burst 5
	key := "test-key"

	// Use all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow(key)
	}

	// Should be denied
	if limiter.Allow(key) {
		t.Fatal("Should be denied before refill")
	}

	// Wait for refill (100ms = 1 token at 10/sec)
	time.Sleep(150 * time.Millisecond)

	// Should allow at least one request after refill
	if !limiter.Allow(key) {
		t.Fatal("Should allow request after refill")
	}
}

func TestTokenBucketRateLimiter_Wait(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(100, 1) // 100 req/sec, burst 1
	key := "test-key"

	// Use the only token
	limiter.Allow(key)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx, key)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	// Should have waited at least ~10ms (1/100 sec)
	if duration < 5*time.Millisecond {
		t.Fatalf("Wait duration too short: %v", duration)
	}
}

func TestTokenBucketRateLimiter_Reset(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, 5)
	key := "test-key"

	// Use all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow(key)
	}

	// Should be denied
	if limiter.Allow(key) {
		t.Fatal("Should be denied before reset")
	}

	// Reset
	limiter.Reset(key)

	// Should allow requests again (new bucket with full tokens)
	if !limiter.Allow(key) {
		t.Fatal("Should allow request after reset")
	}
}

func TestSlidingWindowRateLimiter_Allow(t *testing.T) {
	limiter := NewSlidingWindowRateLimiter(5, 1*time.Second)
	key := "test-key"

	// Should allow 5 requests within window
	for i := 0; i < 5; i++ {
		if !limiter.Allow(key) {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if limiter.Allow(key) {
		t.Fatal("Request 6 should be denied")
	}

	// Wait for window to slide
	time.Sleep(1100 * time.Millisecond)

	// Should allow requests again
	if !limiter.Allow(key) {
		t.Fatal("Should allow request after window slide")
	}
}

func TestFixedWindowRateLimiter_Allow(t *testing.T) {
	limiter := NewFixedWindowRateLimiter(3, 500*time.Millisecond)
	key := "test-key"

	// Should allow 3 requests in current window
	for i := 0; i < 3; i++ {
		if !limiter.Allow(key) {
			t.Fatalf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if limiter.Allow(key) {
		t.Fatal("Request 4 should be denied in current window")
	}

	// Wait for window reset
	time.Sleep(550 * time.Millisecond)

	// Should allow requests in new window
	if !limiter.Allow(key) {
		t.Fatal("Should allow request in new window")
	}
}

func TestFixedWindowRateLimiter_Wait(t *testing.T) {
	limiter := NewFixedWindowRateLimiter(1, 200*time.Millisecond)
	key := "test-key"

	// Use the only token in window
	limiter.Allow(key)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx, key)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	// Should have waited for window reset (~200ms)
	if duration < 150*time.Millisecond {
		t.Fatalf("Wait duration too short: %v (expected ~200ms)", duration)
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(1, 1)
	key := "test-key"

	// Use token
	limiter.Allow(key)

	// Create context that will be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctx, key)

	if err == nil {
		t.Fatal("Wait should return error on context cancellation")
	}

	if err != context.DeadlineExceeded {
		t.Fatalf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestRateLimiter_MultipleKeys(t *testing.T) {
	limiter := NewTokenBucketRateLimiter(10, 2)

	key1 := "user-1"
	key2 := "user-2"

	// Use all tokens for key1
	limiter.Allow(key1)
	limiter.Allow(key1)

	// key1 should be denied
	if limiter.Allow(key1) {
		t.Fatal("key1 should be denied")
	}

	// key2 should still be allowed (separate bucket)
	if !limiter.Allow(key2) {
		t.Fatal("key2 should be allowed (separate bucket)")
	}
}
