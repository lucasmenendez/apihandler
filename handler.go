// apihandler package provides a simple http.Handler implementation with a REST
// friendly API syntax. Provides simple methods to assign handlers to a path by
// HTTP method. It supports basic HTTP methods and route path arguments.
package apihandler

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

var (
	// supportedMethods variable contains the list of HTTP suppoted methods
	supportedMethods = []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}
)

// Config struct contains the configuration parameters to initialize a new
// Handler instance. It contains the CORS flag to enable CORS headers in the
// responses, the rate to limit the requests per second, and the limit of
// requests allowed per second. If the rate or the limit are set to 0, the
// rate limiter will be disabled.
type Config struct {
	CORS  bool
	Rate  float64
	Limit int
}

// Handler struct cotains the list of assigned routes and also an error channel
// to listen to raised errors using `Handler.Error(error)`.
type Handler struct {
	mtx         *sync.Mutex
	routes      []*route
	rateLimiter *rateLimiter
	cors        bool
}

// URIParam function returns the value of the named argument from the request
// context. It is used to access the named arguments from the request URI in the
// handler function.
func URIParam(ctx context.Context, key string) string {
	return ctx.Value(argName(key)).(string)
}

// NewHandler function returns a Handler initialized and read-to-use.
func NewHandler(cfg *Config) *Handler {
	if cfg == nil {
		cfg = &Config{}
	}
	var rl *rateLimiter
	if cfg.Rate > 0 && cfg.Limit > 0 {
		rl = &rateLimiter{
			r: rate.Limit(cfg.Rate),
			b: cfg.Limit,
		}
	}
	return &Handler{
		mtx:         &sync.Mutex{},
		routes:      []*route{},
		rateLimiter: rl,
		cors:        cfg.CORS,
	}
}

// ServerHTTP method implements `http.Handler` interface. This funcion is
// executed when a request is received. It checks if the handler has a route
// assigned with the request method and path to execute the route handler. If
// it is not registered yet, the function sends a response with a 405 HTTP
// error. It also stores the URL parameters, if they exist, in the request
// context to allow the handler to access them.
func (m *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// check if rate limiter is enabled and if the request is allowed
	if m.rateLimiter != nil {
		if !m.rateLimiter.Allowed(req.RemoteAddr) {
			http.Error(res, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}
	}
	// check if CORS is enabled and set headers
	if m.cors {
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Headers", "*")
		res.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, CONNECT, TRACE")
		if req.Method == http.MethodOptions {
			res.WriteHeader(http.StatusOK)
			return
		}
	}
	// find route and execute handler
	if route, exist := m.find(req.Method, req.URL.Path); exist {
		if args, ok := route.decodeArgs(req.URL.Path); ok {
			ctx := req.Context()
			for key, val := range args {
				ctx = context.WithValue(ctx, argName(key), val)
			}
			route.handler(res, req.WithContext(ctx))
			return
		}
	}
	// if no route is found, return 405 Method Not Allowed
	http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

// HandleFunc method assign the provided handler for requests sent to the
// desired method and path. It checks if the method provided is already
// supported before assign it. It also transform the provided path into a regex
// and assign it to the created route. If already exists a route with the same
// method and path, it will be overwritten.
func (m *Handler) HandleFunc(method, path string, handler func(http.ResponseWriter, *http.Request)) error {
	for _, supported := range supportedMethods {
		if supported == method {
			m.mtx.Lock()
			defer m.mtx.Unlock()
			// create route and calculate regex
			newRoute := &route{
				method:  method,
				path:    path,
				handler: handler,
			}
			if err := newRoute.parse(); err != nil {
				return fmt.Errorf("error registering route '%s': %w", path, err)
			}
			// try to overwrite if already exist a registered handler for it
			for i, r := range m.routes {
				if r.method == method && r.path == path {
					m.routes[i] = newRoute
					return nil
				}
			}
			// if it does not exists, create it
			m.routes = append(m.routes, newRoute)
			return nil
		}
	}
	return fmt.Errorf("method not allowed")
}

// Get method wraps `Handler.HandleFunc` for HTTP method 'GET'.
func (m *Handler) Get(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodGet, p, h)
}

// Head method wraps `Handler.HandleFunc` for HTTP method 'HEAD'.
func (m *Handler) Head(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodHead, p, h)
}

// Post method wraps `Handler.HandleFunc` for HTTP method 'POST'.
func (m *Handler) Post(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodPost, p, h)
}

// Put method wraps `Handler.HandleFunc` for HTTP method 'PUT'.
func (m *Handler) Put(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodPut, p, h)
}

// Patch method wraps `Handler.HandleFunc` for HTTP method 'PATCH'.
func (m *Handler) Patch(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodPatch, p, h)
}

// Delete method wraps `Handler.HandleFunc` for HTTP method 'DELETE'.
func (m *Handler) Delete(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodDelete, p, h)
}

// Connect method wraps `Handler.HandleFunc` for HTTP method 'CONNECT'.
func (m *Handler) Connect(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodConnect, p, h)
}

// Options method wraps `Handler.HandleFunc` for HTTP method 'OPTIONS'.
func (m *Handler) Options(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodOptions, p, h)
}

// Trace method wraps `Handler.HandleFunc` for HTTP method 'TRACE'.
func (m *Handler) Trace(p string, h func(http.ResponseWriter, *http.Request)) error {
	return m.HandleFunc(http.MethodTrace, p, h)
}

// find method search for a registered handler for the method and request URI
// provided, matching the routes regex with the URI provided. If the route is
// not registered, it returns also false.
func (m *Handler) find(method, requestURI string) (*route, bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for _, r := range m.routes {
		if r.method == method && r.match(requestURI) {
			return r, true
		}
	}
	return nil, false
}
