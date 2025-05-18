package apihandler

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	// create a new rate limiter with a maximum of 1 tokens each 2 seconds
	rl := NewRateLimiter(context.Background(), 2, 2*time.Second)
	time.Sleep(time.Second) // wait to desynchronize the cleanup goroutine

	testID := "test-client"
	if allowed := rl.Allow(testID); !allowed {
		t.Errorf("Expected Allow to return true, got false")
	}
	if allowed := rl.Allow(testID); !allowed {
		t.Errorf("Expected Allow to return true, got false")
	}
	if allowed := rl.Allow(testID); allowed {
		t.Errorf("Expected Allow to return false, got true")
	}
	// wait for 2 seconds to allow the token to refresh
	time.Sleep(2 * time.Second)
	if allowed := rl.Allow(testID); !allowed {
		t.Errorf("Expected Allow to return true after waiting, got false")
	}
}

func TestMiddleware(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// create a new rate limiter with a maximum of 1 tokens each 2 seconds
	rl := NewRateLimiter(ctx, 1, 2*time.Second)
	time.Sleep(time.Second) // wait to desynchronize the cleanup goroutine

	// create a http server with the rate limiter middleware
	testServer := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(
			rl.Middleware(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Ok"))
			}),
		),
	}
	// start the server in a goroutine
	go func() {
		_ = testServer.ListenAndServe()
	}()

	// make a request to the server
	resp, err := http.Get("http://localhost:8080")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	// make another request that should not be allowed
	resp, err = http.Get("http://localhost:8080")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("Expected status code %d, got %d", http.StatusTooManyRequests, resp.StatusCode)
	}
}

func TestCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// create a new rate limiter with a maximum of 1 tokens each 2 seconds
	rl := NewRateLimiter(ctx, 2, 2*time.Second)
	time.Sleep(time.Second) // wait to desynchronize the cleanup goroutine

	testID := "test-client"
	if allowed := rl.Allow(testID); !allowed {
		t.Errorf("Expected Allow to return true, got false")
	}
	// wait for 3 seconds to allow the client to be cleaned up
	time.Sleep(3 * time.Second)
	if allowed := rl.Allow(testID); !allowed {
		t.Errorf("Expected Allow to return true after cleanup, got false")
	}
	cancel() // cancel the context to stop the cleanup goroutine
	// wait for the cleanup goroutine to finish
	time.Sleep(1 * time.Second)
}

func TestRequestHostname(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	req.RemoteAddr = "example.com"
	if hostname := requestHostname(req); hostname != "example.com" {
		t.Errorf("Expected hostname to be 'example.com', got '%s'", hostname)
	}
	req.RemoteAddr = "127.0.0.1:8080"
	if hostname := requestHostname(req); hostname != "127.0.0.1" {
		t.Errorf("Expected hostname to be '127.0.0.1', got '%s'", hostname)
	}
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	if hostname := requestHostname(req); hostname != "192.168.1.1" {
		t.Errorf("Expected hostname to be '192.168.1.1', got '%s'", hostname)
	}
}
