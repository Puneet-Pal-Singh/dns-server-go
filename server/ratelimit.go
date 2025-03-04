package server

import (
	"sync"
	"time"
)

// RateLimiter interface for SOLID compliance
type RateLimiter interface {
	AllowQuery(ip string) bool
}

// TokenBucket implementation
type tokenBucket struct {
	capacity     int
	refillRate   time.Duration
	mu           sync.Mutex
	buckets      map[string]int
	lastRefilled map[string]time.Time
}

type TokenBucketRateLimiter struct {
	tb *tokenBucket
}

func NewTokenBucketRateLimiter(capacity int, refillRate time.Duration) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		tb: &tokenBucket{
			capacity:     capacity,
			refillRate:   refillRate,
			buckets:      make(map[string]int),
			lastRefilled: make(map[string]time.Time),
		},
	}
}

func (rl *TokenBucketRateLimiter) AllowQuery(ip string) bool {
	rl.tb.mu.Lock()
	defer rl.tb.mu.Unlock()

	now := time.Now()

	// Initialize if first request
	if _, exists := rl.tb.buckets[ip]; !exists {
		rl.tb.buckets[ip] = rl.tb.capacity - 1
		rl.tb.lastRefilled[ip] = now
		return true
	}

	// Calculate tokens to add/refill
	elapsed := now.Sub(rl.tb.lastRefilled[ip])
	refills := int(elapsed / rl.tb.refillRate)

	// Add tokens if refill period passed
	if refills > 0 {
		rl.tb.lastRefilled[ip] = rl.tb.lastRefilled[ip].Add(time.Duration(refills) * rl.tb.refillRate)
		rl.tb.buckets[ip] = min(rl.tb.capacity, rl.tb.buckets[ip]+refills)
	}

	// Check available tokens FIRST before consuming
	if rl.tb.buckets[ip] <= 0 {
		return false
	}

	rl.tb.buckets[ip]--
	return true
}

func (rl *TokenBucketRateLimiter) Cleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			rl.tb.mu.Lock()
			for ip := range rl.tb.buckets {
				if time.Since(rl.tb.lastRefilled[ip]) > 24*time.Hour {
					delete(rl.tb.buckets, ip)
					delete(rl.tb.lastRefilled, ip)
				}
			}
			rl.tb.mu.Unlock()
		}
	}()
}

// Modified cleanup function with observability hooks
// func (rl *TokenBucketRateLimiter) Cleanup(interval time.Duration, onCleanup func(ips []string)) {
//     go func() {
//         ticker := time.NewTicker(interval)
//         for range ticker.C {
//             var cleanedIPs []string
            
//             rl.tb.mu.Lock()
//             for ip := range rl.tb.buckets {
//                 if time.Since(rl.tb.lastRefilled[ip]) > 24*time.Hour {
//                     cleanedIPs = append(cleanedIPs, ip)
//                     delete(rl.tb.buckets, ip)
//                     delete(rl.tb.lastRefilled, ip)
//                 }
//             }
//             rl.tb.mu.Unlock()
            
//             if onCleanup != nil && len(cleanedIPs) > 0 {
//                 onCleanup(cleanedIPs)
//             }
//         }
//     }()
// }
