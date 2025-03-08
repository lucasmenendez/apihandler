package apihandler

import (
	"context"
	"regexp"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimiter struct contains the context, a map of rate limiters, the rate,
// burst, and TTL for the rate limiters. This struct is used to manage rate
// limiters for IP addresses and to clean up rate limiters that have not been
// used for a specified duration.
type rateLimiter struct {
	ctx          context.Context
	rateLimiters sync.Map
	r            rate.Limit
	b            int
	ttl          time.Duration
}

// rateLimiterEntry struct contains the rate limiter and the last accessed time
// for an IP address. This is used to track the last time the rate limiter was
// accessed and to determine if it should be removed from the map of rate
// limiters.
type rateLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

var hostnameRgx = regexp.MustCompile(`^(?:(?:https?://)((?:(?:[a-zA-Z0-9][a-zA-Z0-9-]*\.)+[a-zA-Z]{2,}|(?:\d{1,3}(?:\.\d{1,3}){3})))(?::\d+)?|((?:(?:[a-zA-Z0-9][a-zA-Z0-9-]*\.)+[a-zA-Z]{2,}|(?:\d{1,3}(?:\.\d{1,3}){3})))(?::\d+)?)$`)

func hostnameFromAddr(addr string) (string, bool) {
	matches := hostnameRgx.FindStringSubmatch(addr)
	if len(matches) < 3 {
		return "", false
	}
	if matches[1] != "" {
		return matches[1], true
	}
	return matches[2], true
}

// Add method creates a new rate limiter for the provided IP address and stores
// it in the map of rate limiters. If a rate limiter already exists for the IP
// address, it will be overwritten. This method is useful for initializing rate
// limiters for new IP addresses.
func (rl *rateLimiter) Add(remoteAddr string) *rate.Limiter {
	hostname, ok := hostnameFromAddr(remoteAddr)
	if !ok {
		return nil
	}
	entry := &rateLimiterEntry{
		limiter:    rate.NewLimiter(rl.r, rl.b),
		lastAccess: time.Now(),
	}
	rl.rateLimiters.Store(hostname, entry)
	return entry.limiter
}

// Allowed method checks if the provided IP address is allowed to make a
// request. If the IP address is not in the map of rate limiters, false is
// returned. If the IP address is in the map of rate limiters, the rate limiter
// is checked to see if the request is allowed.
func (rl *rateLimiter) Allowed(remoteAddr string) bool {
	limiter := rl.Get(remoteAddr)
	if limiter == nil {
		return false
	}
	return limiter.Allow()
}

// Get method returns the rate limiter for the provided IP address if it
// exists in the map of rate limiters. If no rate limiter exists for the IP
// address, a new rate limiter is created, stored in the map, and then
// returned. This ensures that every IP address has a rate limiter and allows
// for dynamic addition of new IP addresses without needing to explicitly
// call Add.
func (rl *rateLimiter) Get(remoteAddr string) *rate.Limiter {
	hostname, ok := hostnameFromAddr(remoteAddr)
	if !ok {
		return nil
	}
	actual, _ := rl.rateLimiters.LoadOrStore(hostname, &rateLimiterEntry{
		limiter:    rate.NewLimiter(rl.r, rl.b),
		lastAccess: time.Now(),
	})
	entry := actual.(*rateLimiterEntry)
	entry.lastAccess = time.Now()
	return entry.limiter
}

// Remove method deletes the rate limiter for the provided IP address from the
// map of rate limiters. This is useful for cleaning up rate limiters that are
// no longer needed.
func (rl *rateLimiter) Remove(remoteAddr string) {
	if hostname, ok := hostnameFromAddr(remoteAddr); ok {
		rl.rateLimiters.Delete(hostname)
	}
}

// Cleanup method removes rate limiters that have not been used for a specified
// duration.
func (rl *rateLimiter) Cleanup() {
	now := time.Now()
	rl.rateLimiters.Range(func(key, value any) bool {
		entry := value.(*rateLimiterEntry)
		if now.Sub(entry.lastAccess) > rl.ttl {
			rl.rateLimiters.Delete(key)
		}
		return true
	})
}

// NewRateLimiter creates a new rateLimiter with the specified rate, burst,
// and TTL.
func NewRateLimiter(ctx context.Context, r float64, b int, ttl time.Duration) *rateLimiter {
	rl := &rateLimiter{
		ctx: ctx,
		r:   rate.Limit(r),
		b:   b,
		ttl: ttl,
	}
	// start a goroutine to clean up rate limiters
	go func() {
		// create a ticker to trigger cleanup at regular intervals
		ticker := time.NewTicker(ttl)
		defer ticker.Stop()
		// loop until the context is canceled or the ticker stops
		for {
			select {
			case <-rl.ctx.Done():
				return
			case <-ticker.C:
				rl.Cleanup()
			}
		}
	}()
	return rl
}
