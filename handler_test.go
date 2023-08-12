package apihandler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

const testMethod = http.MethodGet
const testPath = "/test"

var testHandler = func(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "test")
}

func TestHandleFunc(t *testing.T) {
	handler := New()

	handler.HandleFunc("wrongmethod", testPath, testHandler)
	if _, exist := handler.routes[routeKey("wrongmethod", testPath)]; exist {
		t.Fatalf("expected no handler for [%s] %s", "wrongmethod", testPath)
	}

	handler.HandleFunc(testMethod, testPath, testHandler)
	if _, exist := handler.routes[routeKey(testMethod, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", testMethod, testPath)
	}
}

func TestServerHTTP(t *testing.T) {
	handler := New()
	handler.HandleFunc(http.MethodGet, testPath, testHandler)
	go func() {
		http.ListenAndServe(":8080", handler)
	}()

	resp, err := http.Get("http://localhost:8080")
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}
	if err := string(body); !strings.Contains(err, "405") {
		t.Fatalf("expected 405 error, got %s", err)
	}

	resp, err = http.Get("http://localhost:8080" + testPath)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("expected nil, got error: %s", err)
	}

	if string(body) != "test" {
		t.Fatalf("expected 'test', got %s", string(body))
	}
}

func TestHTTPMethods(t *testing.T) {
	handler := New()

	handler.Get(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodGet, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodGet, testPath)
	}

	handler.Head(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodHead, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodHead, testPath)
	}

	handler.Post(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodPost, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPost, testPath)
	}

	handler.Put(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodPut, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPut, testPath)
	}

	handler.Patch(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodPatch, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPatch, testPath)
	}

	handler.Delete(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodDelete, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodDelete, testPath)
	}

	handler.Connect(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodConnect, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodConnect, testPath)
	}

	handler.Options(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodOptions, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodOptions, testPath)
	}

	handler.Trace(testPath, testHandler)
	if _, exist := handler.routes[routeKey(http.MethodTrace, testPath)]; !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodTrace, testPath)
	}
}
