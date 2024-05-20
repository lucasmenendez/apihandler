package apihandler

import (
	"sync"

	"golang.org/x/time/rate"
)

// rateLimiter struct contains the list of IP addresses and their rate limiter
// to control the number of requests (b) per frequency defined (r).
type rateLimiter struct {
	ipList sync.Map
	r      rate.Limit
	b      int
}

// Add method creates a new rate limiter for the provided IP address and stores
// it in the list of rate limiters.
func (al *rateLimiter) Add(ip string) *rate.Limiter {
	limiter := rate.NewLimiter(al.r, al.b)
	al.ipList.Store(ip, limiter)
	return limiter
}

// Get method returns the rate limiter for the provided IP address if it exists
// in the list of rate limiters, otherwise creates a new rate limiter and stores
// it in the list.
func (al *rateLimiter) Get(ip string) *rate.Limiter {
	if limiter, ok := al.ipList.Load(ip); ok {
		return limiter.(*rate.Limiter)
	}
	return al.Add(ip)
}
