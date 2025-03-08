package apihandler

import (
	"context"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestRateLimiter_Add(t *testing.T) {
	ctx := context.Background()
	rl := NewRateLimiter(ctx, 1, 5, time.Minute)
	invalidIP := "invalid"
	limiter := rl.Add(invalidIP)
	if limiter != nil {
		t.Fatalf("expected rate limiter to not be created, got %v", limiter)
	}

	ip := "192.168.1.1"
	limiter = rl.Add(ip)
	if limiter == nil {
		t.Fatalf("expected rate limiter to be created, got nil")
	}

	loadedLimiter, ok := rl.rateLimiters.Load(ip)
	if !ok {
		t.Fatalf("expected rate limiter to be stored, but it was not found")
	}

	entry := loadedLimiter.(*rateLimiterEntry)
	if entry.limiter != limiter {
		t.Fatalf("expected stored rate limiter to match created rate limiter")
	}
}

func TestRateLimiter_Get(t *testing.T) {
	ctx := context.Background()
	rl := NewRateLimiter(ctx, 1, 5, time.Minute)
	ip := "192.168.1.1"

	limiter := rl.Get(ip)
	if limiter == nil {
		t.Fatalf("expected rate limiter to be created, got nil")
	}

	loadedLimiter, ok := rl.rateLimiters.Load(ip)
	if !ok {
		t.Fatalf("expected rate limiter to be stored, but it was not found")
	}

	entry := loadedLimiter.(*rateLimiterEntry)
	if entry.limiter != limiter {
		t.Fatalf("expected stored rate limiter to match created rate limiter")
	}
}

func TestRateLimiter_Remove(t *testing.T) {
	ctx := context.Background()
	rl := NewRateLimiter(ctx, 1, 5, time.Minute)
	ip := "192.168.1.1"

	rl.Add(ip)
	rl.Remove(ip)

	_, ok := rl.rateLimiters.Load(ip)
	if ok {
		t.Fatalf("expected rate limiter to be removed, but it was found")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	ctx := context.Background()
	rl := NewRateLimiter(ctx, 1, 5, time.Second)
	ip := "192.168.1.1"

	rl.Add(ip)
	time.Sleep(2 * time.Second)
	rl.Cleanup()

	_, ok := rl.rateLimiters.Load(ip)
	if ok {
		t.Fatalf("expected rate limiter to be cleaned up, but it was found")
	}
}

func TestNewRateLimiter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rl := NewRateLimiter(ctx, 1, 5, time.Minute)
	if rl == nil {
		t.Fatalf("expected rate limiter to be created, got nil")
	}

	if rl.r != rate.Limit(1) {
		t.Fatalf("expected rate limit to be 1, got %v", rl.r)
	}

	if rl.b != 5 {
		t.Fatalf("expected burst to be 5, got %v", rl.b)
	}

	if rl.ttl != time.Minute {
		t.Fatalf("expected TTL to be 1 minute, got %v", rl.ttl)
	}
}

func TestRateLimiter_AlreadyLimited(t *testing.T) {
	ctx := context.Background()
	rl := NewRateLimiter(ctx, 1, 1, time.Minute)
	ip := "192.168.1.1"

	limiter := rl.Get(ip)
	if limiter == nil {
		t.Fatalf("expected rate limiter to be created, got nil")
	}
	if !limiter.Allow() {
		t.Fatalf("expected first request to be allowed")
	}

	if limiter.Allow() {
		t.Fatalf("expected second request to be denied due to rate limiting")
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	rl := NewRateLimiter(ctx, 1, 1, time.Minute) // Adjusted rate limit and burst values
	ip := "192.168.1.1"

	var wg sync.WaitGroup
	var allowedCount, disallowedCount int
	var mu sync.Mutex
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limiter := rl.Get(ip)
			mu.Lock()
			if limiter.Allow() {
				allowedCount++
			} else {
				disallowedCount++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if allowedCount != 1 {
		t.Errorf("expected 1 request to be allowed, got %d", allowedCount)
	}
	if disallowedCount != 19 {
		t.Errorf("expected 19 requests to be denied, got %d", disallowedCount)
	}
}

func TestRateLimiter_CleanupWithMultipleIPs(t *testing.T) {
	ctx := context.Background()
	rl := NewRateLimiter(ctx, 1, 5, time.Second)
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	for _, ip := range ips {
		rl.Add(ip)
	}

	time.Sleep(2 * time.Second)
	rl.Cleanup()
	for _, ip := range ips {
		_, ok := rl.rateLimiters.Load(ip)
		if ok {
			t.Fatalf("expected rate limiter for IP %s to be cleaned up, but it was found", ip)
		}
	}
}

func TestHostnameFromAddr(t *testing.T) {
	tests := []struct {
		addr     string
		expected string
		ok       bool
	}{
		{"http://example.com", "example.com", true},
		{"https://example.com", "example.com", true},
		{"http://example.com:8080", "example.com", true},
		{"https://example.com:8080", "example.com", true},
		{"example.com", "example.com", true},
		{"example.com:8080", "example.com", true},
		{"http://192.168.1.1", "192.168.1.1", true},
		{"https://192.168.1.1", "192.168.1.1", true},
		{"http://192.168.1.1:8080", "192.168.1.1", true},
		{"https://192.168.1.1:8080", "192.168.1.1", true},
		{"192.168.1.1", "192.168.1.1", true},
		{"192.168.1.1:8080", "192.168.1.1", true},
		{"", "", false},
		{"http://", "", false},
		{"https://", "", false},
		{"invalid", "", false},
	}

	for _, test := range tests {
		hostname, ok := hostnameFromAddr(test.addr)
		if hostname != test.expected || ok != test.ok {
			t.Errorf("hostnameFromAddr(%q) = %q, %v; want %q, %v", test.addr, hostname, ok, test.expected, test.ok)
		}
	}
}
func TestRateLimiter_Allowed(t *testing.T) {
	ctx := context.Background()
	rl := NewRateLimiter(ctx, 1, 1, time.Minute)
	ip := "192.168.1.1"

	// Test when IP is not in the map of rate limiters
	if rl.Allowed("invalid") {
		t.Fatalf("expected request to be denied for invalid IP")
	}

	// Test when IP is in the map of rate limiters and request is allowed
	if !rl.Allowed(ip) {
		t.Fatalf("expected first request to be allowed")
	}

	// Test when IP is in the map of rate limiters and request is denied
	if rl.Allowed(ip) {
		t.Fatalf("expected second request to be denied due to rate limiting")
	}
}

