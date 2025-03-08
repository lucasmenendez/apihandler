package apihandler

import (
	"net/http"
	"regexp"
	"testing"
)

func TestRouteParse(t *testing.T) {
	r := &route{
		method:  "GET",
		path:    "/users/{id}",
		handler: func(w http.ResponseWriter, r *http.Request) {},
	}
	err := r.parse()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expectedRgx := regexp.MustCompile(`\/users\/(?P<id>.+)$`)
	if r.rgx.String() != expectedRgx.String() {
		t.Errorf("expected %v, got %v", expectedRgx, r.rgx)
	}
}

func TestRouteMatch(t *testing.T) {
	r := &route{
		method:  "GET",
		path:    "/users/{id}",
		handler: func(w http.ResponseWriter, r *http.Request) {},
	}
	r.parse()
	tests := []struct {
		uri      string
		expected bool
	}{
		{"/users/123", true},
		{"/users/", false},
		{"/users/123/profile", false},
	}
	for _, test := range tests {
		if r.match(test.uri) != test.expected {
			t.Errorf("expected %v, got %v for uri %v", test.expected, !test.expected, test.uri)
		}
	}
}

func TestRouteDecodeArgs(t *testing.T) {
	r := &route{
		method:  "GET",
		path:    "/users/{id}",
		handler: func(w http.ResponseWriter, r *http.Request) {},
	}
	r.parse()
	args, ok := r.decodeArgs("/users/123")
	if !ok {
		t.Fatalf("expected true, got false")
	}
	expectedArgs := map[string]string{"id": "123"}
	for k, v := range expectedArgs {
		if args[k] != v {
			t.Errorf("expected %v, got %v for key %v", v, args[k], k)
		}
	}
}
