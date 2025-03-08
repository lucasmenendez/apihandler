package apihandler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testMethod = http.MethodGet
	testPath   = "/test/{name}"
	testURI    = "/test/args"
)

var testHandler = func(w http.ResponseWriter, req *http.Request) {
	name := URIParam(req.Context(), "name")
	fmt.Fprintf(w, "test_%s", name)
}

func TestHandleFunc(t *testing.T) {
	handler := NewHandler(nil)

	if err := handler.HandleFunc("wrongmethod", testPath, testHandler); err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := handler.HandleFunc(testMethod, testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(testMethod, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", testMethod, testPath)
	}

	if err := handler.HandleFunc(testMethod, `^\/(?!\/)(.*?)`, testHandler); err == nil {
		t.Fatal("expected error, got nil")
	} else if !strings.Contains(err.Error(), "error registering route") {
		t.Fatalf("expected 'error registering route' error got %s", err)
	}
	if _, exist := handler.find(testMethod, `^\/(?!\/)(.*?)`); exist {
		t.Fatalf("expected no handler for [%s] %s", testMethod, testPath)
	}
}

func TestServerHTTP(t *testing.T) {
	handler := NewHandler(&Config{CORS: false})
	_ = handler.HandleFunc(http.MethodGet, testPath, testHandler)

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	if err := string(body); !strings.Contains(err, http.StatusText(http.StatusMethodNotAllowed)) {
		t.Fatalf("expected 405 error, got %s", err)
	}

	resp, err = http.Get(server.URL + testURI)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}

	if string(body) != "test_args" {
		t.Fatalf("expected 'test_args', got %s", string(body))
	}
	resp, err = http.Get(server.URL + "/invalid")
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 error, got %d", resp.StatusCode)
	}
}

func TestHTTPMethods(t *testing.T) {
	handler := NewHandler(&Config{CORS: false})

	if err := handler.Get(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodGet, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodGet, testPath)
	}
	// try overwrite
	if err := handler.Get(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodGet, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodGet, testPath)
	}

	if err := handler.Head(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodHead, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodHead, testPath)
	}

	if err := handler.Post(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodPost, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPost, testPath)
	}

	if err := handler.Put(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodPut, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPut, testPath)
	}

	if err := handler.Patch(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodPatch, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPatch, testPath)
	}

	if err := handler.Delete(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodDelete, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodDelete, testPath)
	}

	if err := handler.Connect(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodConnect, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodConnect, testPath)
	}

	if err := handler.Options(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodOptions, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodOptions, testPath)
	}

	if err := handler.Trace(testPath, testHandler); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}
	if _, exist := handler.find(http.MethodTrace, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodTrace, testPath)
	}
}

func Test_parseAndDecodeArgs(t *testing.T) {
	routePath := "/api/{version}/user/{id}"
	testRoute := &route{
		path: routePath,
	}
	if err := testRoute.parse(); err != nil {
		t.Fatalf("expected nil, got %s", err)
	}

	wrongRequestURI := "/api/v2"
	if _, match := testRoute.decodeArgs(wrongRequestURI); match {
		t.Fatal("expected false, got true")
	}
	wrongRequestURI = "/api/v2/user/0xffffff/age"
	if _, match := testRoute.decodeArgs(wrongRequestURI); match {
		t.Fatal("expected false, got true")
	}
	wrongRequestURI = "/api/v2/user//"
	if _, match := testRoute.decodeArgs(wrongRequestURI); match {
		t.Fatal("expected false, got true")
	}
	requestURI := "/api/v2/user/0xffffff"
	args, match := testRoute.decodeArgs(requestURI)
	if !match {
		t.Fatal("expected true, got false")
	}
	if value, ok := args["version"]; !ok || value != "v2" {
		t.Fatalf("expected 'v2', got '%s'", value)
	}
	if value, ok := args["id"]; !ok || value != "0xffffff" {
		t.Fatalf("expected '0xffffff', got '%s'", value)
	}
	requestURI = "/api/v3/user/0xffffff/"
	args, match = testRoute.decodeArgs(requestURI)
	if !match {
		t.Fatal("expected true, got false")
	}
	if value, ok := args["version"]; !ok || value != "v3" {
		t.Fatalf("expected 'v3', got '%s'", value)
	}
	if value, ok := args["id"]; !ok || value != "0xffffff" {
		t.Fatalf("expected '0xffffff', got '%s'", value)
	}
}

func TestCORSHeaders(t *testing.T) {
	handler := NewHandler(&Config{CORS: true})
	_ = handler.HandleFunc(http.MethodGet, testPath, testHandler)

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + testURI)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS headers, got none")
	}

	req, err := http.NewRequest(http.MethodOptions, server.URL+testURI, nil)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK, got %d", resp.StatusCode)
	}

	server.Close()
	handler = NewHandler(&Config{CORS: false})
	_ = handler.HandleFunc(http.MethodGet, testPath, testHandler)
	server = httptest.NewServer(handler)

	resp, err = http.Get(server.URL + testURI)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no CORS headers, got some")
	}

	req, err = http.NewRequest(http.MethodOptions, server.URL+testURI, nil)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status Method Not Allowed, got %d", resp.StatusCode)
	}
}

func TestHandlerWithRateLimiter(t *testing.T) {
	handler := NewHandler(&Config{
		CORS:  false,
		Rate:  1, // 1 request per second
		Limit: 1, // burst limit of 1
	})

	handler.Get(testPath, testHandler)

	server := httptest.NewServer(handler)
	defer server.Close()

	// Helper function to perform a request and check the status code
	doRequest := func() (int, error) {
		resp, err := http.Get(server.URL + testURI)
		if err != nil {
			return 0, err
		}
		return resp.StatusCode, nil
	}

	// First request should be allowed
	if status, err := doRequest(); err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	} else if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}

	// Wait for a short duration to ensure the rate limiter is in effect
	time.Sleep(500 * time.Millisecond)

	// Second request should be rate limited
	if status, err := doRequest(); err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	} else if status != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, status)
	}

	// Wait for rate limiter to reset
	time.Sleep(1 * time.Second)

	// Third request should be allowed again
	if status, err := doRequest(); err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	} else if status != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, status)
	}
}
