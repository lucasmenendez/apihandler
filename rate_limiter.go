package apihandler

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// client struct represents a client making requests to the API. It holds
// the number of tokens consumed (the number of requests that has made) by
// the client and the time when the client was included in the rate limiter.
type client struct {
	tokens int
	age    time.Time
}

// RateLimiter is a simple rate limiter that allows a maximum number of
// requests from a client within a specified time interval. It uses an IP
// address or hostname to identify clients and tracks the number of tokens
// available for each client. Each request consumes a token, and if no tokens
// are available, the request is denied. If a client has not made a request
// within the specified interval, their tokens are reset to 1, allowing them
// to make a new request. The rate limiter runs a cleanup goroutine that
// periodically removes clients that have not made requests within the
// specified interval, freeing up memory and ensuring that the rate limiter
// does not grow indefinitely.
type RateLimiter struct {
	ctx       context.Context
	cancel    context.CancelFunc
	clients   map[string]*client
	mtx       sync.Mutex
	maxTokens int
	interval  time.Duration
}

// NewRateLimiter creates a new RateLimiter instance with the specified
// maximum number of tokens and the interval for token refresh. It initializes
// the internal context and starts a cleanup goroutine to remove old clients
// that have not made requests within the specified interval.
func NewRateLimiter(ctx context.Context, maxTokens int, interval time.Duration) *RateLimiter {
	innerCtx, cancel := context.WithCancel(ctx)
	rt := &RateLimiter{
		ctx:       innerCtx,
		cancel:    cancel,
		clients:   make(map[string]*client),
		maxTokens: maxTokens,
		interval:  interval,
	}
	go rt.cleanup()
	return rt
}

// Allow method checks if the request is allowed based on the rate limit.
// It returns true if the request is allowed, false otherwise. A request
// is allowed if the client has tokens available, or if the client is older
// than the interval, in which case the tokens are reset to 1 and the age
// is updated.
func (rl *RateLimiter) Allow(r *http.Request) bool {
	ip := getIPOrHostname(r)
	rl.mtx.Lock()
	defer rl.mtx.Unlock()
	// get the client by IP address
	cl, exists := rl.clients[ip]
	// if the client does not exist, create a new one with 1 token and return
	// true
	if !exists {
		rl.clients[ip] = &client{tokens: 1, age: time.Now()}
		return true
	}
	// if the client exists, check if it has tokens available, if it does,
	// increment the token count and return true
	if cl.tokens < rl.maxTokens {
		cl.tokens++
		return true
	}
	// if the client has no tokens available, check if it is older than the
	// interval, if it is, reset the tokens to 1 and update the age and return
	// true
	if time.Since(cl.age) > rl.interval {
		cl.tokens = 1
		cl.age = time.Now()
		return true
	}
	// if the client has no tokens available and is not older than the interval,
	// return false
	return false
}

// Middleware method wraps a HandlerFunc to apply rate limiting to it, by
// returning a new HandlerFunc that checks the rate limit before calling the
// original handler.
func (rl *RateLimiter) Middleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(r) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// cleanup method runs until the context is done, periodically checking
// the clients included in the rate limiter. If a client was included for
// longer than the specified interval, it is removed from the clients map
// to reset the rate limiter for that client, but also to free up memory
// and ensure that the rate limiter does not grow indefinitely.
func (rl *RateLimiter) cleanup() {
	tiker := time.NewTicker(rl.interval)
	for {
		select {
		case <-rl.ctx.Done():
			tiker.Stop()
			return
		case <-tiker.C:
			rl.mtx.Lock()
			for ip, cl := range rl.clients {
				if time.Since(cl.age) > rl.interval {
					delete(rl.clients, ip)
				}
			}
			rl.mtx.Unlock()
		}
	}
}

// getIPOrHostname extracts the IP address or the hostname from the request.
// It checks the "X-Forwarded-For" header first, which is commonly used
// in reverse proxy setups to forward the original client's IP address.
func getIPOrHostname(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		ip := strings.TrimSpace(parts[0])
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
