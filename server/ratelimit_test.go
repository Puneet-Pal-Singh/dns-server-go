package server

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenBucketRateLimiter(t *testing.T) {
	rl := NewTokenBucketRateLimiter(3, 2*time.Second)

	// Test burst capacity
	for i := 0; i < 3; i++ {
		if !rl.AllowQuery("192.168.1.1") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Fourth request should fail
	if rl.AllowQuery("192.168.1.1") {
		t.Error("Fourth request should be blocked")
	}

	// Test refill after 2 seconds
	time.Sleep(2100 * time.Millisecond)
	if !rl.AllowQuery("192.168.1.1") {
		t.Error("Should allow after refill period")
	}
}

func TestMultipleIPs(t *testing.T) {
	rl := NewTokenBucketRateLimiter(2, 1*time.Second)

	// IP 1 burst
	assert.True(t, rl.AllowQuery("192.168.1.1"))
	assert.True(t, rl.AllowQuery("192.168.1.1"))
	assert.False(t, rl.AllowQuery("192.168.1.1"))

	// IP 2 should have separate bucket
	assert.True(t, rl.AllowQuery("192.168.1.2"))
}

func TestRefillLogic(t *testing.T) {
	rl := NewTokenBucketRateLimiter(3, 2*time.Second)

	// Exhaust tokens
	rl.AllowQuery("10.0.0.1")
	rl.AllowQuery("10.0.0.1")
	rl.AllowQuery("10.0.0.1")

	// Wait 4 seconds (2 refill intervals)
	time.Sleep(4 * time.Second)

	// Should have 2 tokens (min(3, 0 + 2))
	assert.True(t, rl.AllowQuery("10.0.0.1"))
	assert.True(t, rl.AllowQuery("10.0.0.1"))
	assert.False(t, rl.AllowQuery("10.0.0.1"))
}

func TestCleanup(t *testing.T) {
	rl := NewTokenBucketRateLimiter(5, 1*time.Minute)
	rl.Cleanup(1 * time.Second)

	rl.AllowQuery("10.0.0.5")
	time.Sleep(24*time.Hour + 1*time.Second) // Wait past cleanup threshold

	rl.tb.mu.Lock()
	defer rl.tb.mu.Unlock()
	if _, exists := rl.tb.buckets["10.0.0.5"]; exists {
		t.Error("IP should have been cleaned up")
	}
}

func TestConcurrentAccess(t *testing.T) {
	rl := NewTokenBucketRateLimiter(1000, 1*time.Millisecond)
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.AllowQuery("192.168.0.1")
		}()
	}
	wg.Wait()

	// Verify final count
	rl.tb.mu.Lock()
	defer rl.tb.mu.Unlock()
	if rl.tb.buckets["192.168.0.1"] != 0 {
		t.Errorf("Expected 0 tokens, got %d", rl.tb.buckets["192.168.0.1"])
	}
}
