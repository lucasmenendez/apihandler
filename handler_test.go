package apihandler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testMethod = http.MethodGet
const testPath = "/test/{name}"
const testURI = "/test/args"

var testHandler = func(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "test_%s", req.Header.Get("name"))
}

func TestHandleFunc(t *testing.T) {
	handler := New()

	handler.HandleFunc("wrongmethod", testPath, testHandler)
	if _, exist := handler.find("wrongmethod", testPath); exist {
		t.Fatalf("expected no handler for [%s] %s", "wrongmethod", testPath)
	}

	handler.HandleFunc(testMethod, testPath, testHandler)
	if _, exist := handler.find(testMethod, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", testMethod, testPath)
	}

	go func() {
		time.Sleep(time.Millisecond * 300)
		handler.HandleFunc(testMethod, `^\/(?!\/)(.*?)`, testHandler)
		close(handler.Errors)
	}()

	for err := range handler.Errors {
		if err != nil && !strings.Contains(err.Error(), "error parsing route") {
			t.Fatalf("expected 'error parsing route' error got %s", err)
		}
	}
	if _, exist := handler.find(testMethod, `^\/(?!\/)(.*?)`); exist {
		t.Fatalf("expected no handler for [%s] %s", testMethod, testPath)
	}
}

func TestServerHTTP(t *testing.T) {
	handler := New()
	handler.HandleFunc(http.MethodGet, testPath, testHandler)
	go func() {
		if err := http.ListenAndServe(":8080", handler); err != nil {
			t.Log(err)
		}
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

	resp, err = http.Get("http://localhost:8080" + testURI)
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
}

func TestHTTPMethods(t *testing.T) {
	handler := New()

	handler.Get(testPath, testHandler)
	if _, exist := handler.find(http.MethodGet, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodGet, testPath)
	}
	// try overwrite
	handler.Get(testPath, testHandler)
	if _, exist := handler.find(http.MethodGet, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodGet, testPath)
	}

	handler.Head(testPath, testHandler)
	if _, exist := handler.find(http.MethodHead, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodHead, testPath)
	}

	handler.Post(testPath, testHandler)
	if _, exist := handler.find(http.MethodPost, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPost, testPath)
	}

	handler.Put(testPath, testHandler)
	if _, exist := handler.find(http.MethodPut, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPut, testPath)
	}

	handler.Patch(testPath, testHandler)
	if _, exist := handler.find(http.MethodPatch, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodPatch, testPath)
	}

	handler.Delete(testPath, testHandler)
	if _, exist := handler.find(http.MethodDelete, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodDelete, testPath)
	}

	handler.Connect(testPath, testHandler)
	if _, exist := handler.find(http.MethodConnect, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodConnect, testPath)
	}

	handler.Options(testPath, testHandler)
	if _, exist := handler.find(http.MethodOptions, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodOptions, testPath)
	}

	handler.Trace(testPath, testHandler)
	if _, exist := handler.find(http.MethodTrace, testPath); !exist {
		t.Fatalf("expected handler for [%s] %s", http.MethodTrace, testPath)
	}
}

func Test_pathToRegex(t *testing.T) {
	routePath := "/api/{version}/user/{id}"
	routeRgx, err := pathToRegex(routePath)
	if err != nil {
		t.Fatalf("expected nil, got %s", err)
	}

	wrongRequestURI := "/api/v2"
	if _, match := parseArgs(wrongRequestURI, routeRgx); match {
		t.Fatal("expected false, got true")
	}
	requestURI := "/api/v2/user/0xffffff"
	args, match := parseArgs(requestURI, routeRgx)
	if !match {
		t.Fatal("expected true, got false")
	}
	if value, ok := args["version"]; !ok || value != "v2" {
		t.Fatalf("expected 'v2', got '%s'", value)
	}
	if value, ok := args["id"]; !ok || value != "0xffffff" {
		t.Fatalf("expected '0xffffff', got '%s'", value)
	}
}
