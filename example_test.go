package apihandler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func Example() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// limit the number of requests to 20 per minute
	rateLimiter := NewRateLimiter(ctx, 20, time.Minute)
	// create and register a new GET handler
	handler := NewHandler(true)
	err := handler.Get("/service/{service_name}/resource/{resource_name}",
		rateLimiter.Middleware(func(w http.ResponseWriter, r *http.Request) {
			// get router arguments from Header
			status := map[string]string{
				"service":  r.Header.Get("service_name"),
				"resource": r.Header.Get("resource_name"),
				"status":   "ok",
			}
			// encoding response
			body, err := json.Marshal(status)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(fmt.Sprintf("error encoding status: %s", err)))
				return
			}
			// writing response
			_, _ = w.Write(body)
		}))
	if err != nil {
		log.Printf("ERR: error listening for requests: %s\n", err)
	}
	// run http server with created handler
	if err := http.ListenAndServe(":8090", handler); err != nil {
		log.Printf("ERR: error listening for requests: %s\n", err)
		return
	}
}
