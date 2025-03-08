package apihandler

import (
	"context"
	"regexp"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DefaultTTL represents the default time-to-live duration for rate limiting.
// It is set to 5 minutes.
const DefaultTTL = 5 * time.Minute

// hostnameRgx is a regular expression that matches hostnames and IP addresses
// with optional port numbers. It is used to extract the hostname from an
// address string. The regular expression supports both hostnames and IP
// addresses with optional port numbers.
var hostnameRgx = regexp.MustCompile(`^(?:(?:https?://)((?:(?:[a-zA-Z0-9][a-zA-Z0-9-]*\.)+[a-zA-Z]{2,}|(?:\d{1,3}(?:\.\d{1,3}){3})))(?::\d+)?|((?:(?:[a-zA-Z0-9][a-zA-Z0-9-]*\.)+[a-zA-Z]{2,}|(?:\d{1,3}(?:\.\d{1,3}){3})))(?::\d+)?)(?:/.*)?$`)

// rateLimiter struct contains the context, a map of rate limiters, the rate,
// burst, and TTL for the rate limiters. This struct is used to manage rate
// limiters for IP addresses and to clean up rate limiters that have not been
// used for a specified duration.
type rateLimiter struct {
	ctx          context.Context
	rateLimiters map[string]*rateLimiterEntry
	mtx          sync.Mutex
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

// RateLimiter creates a new rateLimiter with the specified rate, burst,
// and TTL.
func RateLimiter(ctx context.Context, r float64, b int, ttl time.Duration) *rateLimiter {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	rl := &rateLimiter{
		ctx:          ctx,
		rateLimiters: make(map[string]*rateLimiterEntry),
		r:            rate.Limit(r),
		b:            b,
		ttl:          ttl,
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
				rl.cleanup()
			}
		}
	}()
	return rl
}

// includeAddr method creates a new rate limiter for the provided IP address
// and stores it in the map of rate limiters. If a rate limiter already exists
// for the IP address, it will be overwritten. This method is useful for
// initializing rate limiters for new IP addresses.
func (rl *rateLimiter) includeAddr(remoteAddr string) *rate.Limiter {
	hostname, ok := hostnameFromAddr(remoteAddr)
	if !ok {
		return nil
	}
	entry := &rateLimiterEntry{
		limiter:    rate.NewLimiter(rl.r, rl.b),
		lastAccess: time.Now(),
	}
	rl.mtx.Lock()
	defer rl.mtx.Unlock()
	rl.rateLimiters[hostname] = entry
	return entry.limiter
}

// isAllowed method checks if the provided IP address is allowed to make a
// request. If the IP address is not in the map of rate limiters, false is
// returned. If the IP address is in the map of rate limiters, the rate
// limiter is checked to see if the request is allowed.
func (rl *rateLimiter) isAllowed(remoteAddr string) bool {
	limiter, _ := rl.addrLimiter(remoteAddr)
	if limiter == nil {
		return false
	}
	return limiter.Allow()
}

// addrLimiter method returns the rate limiter for the provided IP address if
// it exists in the map of rate limiters. If no rate limiter exists for the IP
// address, a new rate limiter is created, stored in the map, and then
// returned. This ensures that every IP address has a rate limiter and allows
// for dynamic addition of new IP addresses without needing to explicitly
// call Add.
func (rl *rateLimiter) addrLimiter(remoteAddr string) (*rate.Limiter, bool) {
	hostname, ok := hostnameFromAddr(remoteAddr)
	if !ok {
		return nil, false
	}
	rl.mtx.Lock()
	defer rl.mtx.Unlock()
	actual, ok := rl.rateLimiters[hostname]
	if !ok {
		actual = &rateLimiterEntry{
			limiter:    rate.NewLimiter(rl.r, rl.b),
			lastAccess: time.Now(),
		}
		rl.rateLimiters[hostname] = actual
	}
	return actual.limiter, ok
}

// removeAddr method deletes the rate limiter for the provided IP address from
// the map of rate limiters. This is useful for cleaning up rate limiters that
// are no longer needed.
func (rl *rateLimiter) removeAddr(remoteAddr string) {
	if hostname, ok := hostnameFromAddr(remoteAddr); ok {
		rl.mtx.Lock()
		delete(rl.rateLimiters, hostname)
		rl.mtx.Unlock()
	}
}

// cleanup method removes rate limiters that have not been used for a specified
// duration.
func (rl *rateLimiter) cleanup() {
	now := time.Now()
	rl.mtx.Lock()
	defer rl.mtx.Unlock()
	for key, value := range rl.rateLimiters {
		if now.Sub(value.lastAccess) > rl.ttl {
			delete(rl.rateLimiters, key)
		}
	}
}

// hostnameFromAddr extracts the hostname from the given address string. It
// returns the hostname and a boolean indicating whether the extraction was
// successful. The function uses a regular expression to find the hostname
// in the address. If the address does not match the expected pattern, it
// returns an empty string and false. The function supports both IP addresses
// and hostnames with optional port numbers. For example, it can extract the
// hostname from "http://example.com/dashboard" or "192.168.1.12:8080",
// returning "example.com" and "192.168.1.12" respectively.
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
