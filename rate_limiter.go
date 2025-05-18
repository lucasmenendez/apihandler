package apihandler

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type client struct {
	tokens int
	age    time.Time
}

type RateLimiter struct {
	ctx       context.Context
	cancel    context.CancelFunc
	clients   map[string]*client
	mtx       sync.Mutex
	maxTokens int
	interval  time.Duration
}

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

func (rl *RateLimiter) Allow(r *http.Request) bool {
	ip := getIP(r)
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

func (rl *RateLimiter) Middleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(r) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

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

func getIP(r *http.Request) string {
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
