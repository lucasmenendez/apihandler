// apihandler package provides a simple http.Handler implementation with a REST
// friendly API syntax. Provides simple methods to assign handlers to a path by
// HTTP method. It supports basic HTTP methods and route path arguments.
package apihandler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// uriSeparator contains a string with the backslash character to split the
// URI for sanity checks
const uriSeparator = "/"

// argsToRgxSub constant contains the regex pattern to match a named argument
// in a request URI, includes the interpolation of the name of the argument.
const argsToRgxSub = "(?P<$arg_name>.+)"

// argsToRgx variable is a regex that allows to detect named arguments from a
// route path, helping to build a regex to match requests URIs with the route
// supporting named args.
var argsToRgx = regexp.MustCompile(`(?U)\{(?P<arg_name>.+)\}`)

// supportedMethods variable contains the list of HTTP suppoted methods
var supportedMethods = []string{
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

// route struct contains the parameters of a valid route, which contains the
// method, the path, a regex to match request URIs with paths that use named
// arguments, and the route handler.
type route struct {
	method  string
	path    string
	rgx     *regexp.Regexp
	handler func(http.ResponseWriter, *http.Request)
}

// parse function transforms the provided path into a regex to match with
// the URI of incoming requests. The resulting regex will be stored into current
// route and will be used to match named arguments from a request URI.
func (r *route) parse() error {
	rgx := argsToRgx.ReplaceAllString(r.path, argsToRgxSub)
	escapedRgx := strings.ReplaceAll(rgx, "/", "\\/")
	var err error
	if r.rgx, err = regexp.Compile(fmt.Sprintf("%s$", escapedRgx)); err != nil {
		return fmt.Errorf("error parsing path: %w", err)
	}
	return nil
}

// match function returns if the requestURI provided matches with the current
// route regex. It also checks if both arguments have the same number of
// URI parts to ensure that is the same level of depth.
func (r *route) match(requestURI string) bool {
	uri, _ := strings.CutSuffix(requestURI, uriSeparator)
	lenURI := strings.Count(uri, uriSeparator)
	lenRgx := strings.Count(r.rgx.String(), uriSeparator)
	return lenURI == lenRgx && r.rgx.MatchString(requestURI)
}

// decodeArgs function returns if the request URI matches with the route regex
// provided and the named arguments that the URI could contain.
func (r *route) decodeArgs(requestURI string) (map[string]string, bool) {
	// check if matches
	if !r.match(requestURI) {
		return nil, false
	}
	// find named arguments
	args := make(map[string]string)
	uri, _ := strings.CutSuffix(requestURI, uriSeparator)
	matches := r.rgx.FindStringSubmatch(uri)
	if len(matches) < 1 {
		return nil, false
	}
	for i, name := range r.rgx.SubexpNames()[0:] {
		args[name] = matches[i]
	}
	return args, true
}

type RateLimitConfig struct {
	Rate  float64
	Limit int
}

type Config struct {
	CORS bool
	*RateLimitConfig
}

// Handler struct cotains the list of assigned routes and also an error channel
// to listen to raised errors using `Handler.Error(error)`.
type Handler struct {
	mtx         *sync.Mutex
	routes      []*route
	rateLimiter *rateLimiter
	cors        bool
}

// NewHandler function returns a Handler initialized and read-to-use.
func NewHandler(cfg *Config) *Handler {
	if cfg == nil {
		cfg = &Config{}
	}

	var rl *rateLimiter
	if cfg.RateLimitConfig != nil {
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
// error.
func (m *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// check if rate limiter is enabled and if the request is allowed
	if m.rateLimiter != nil {
		limiter := m.rateLimiter.Get(req.RemoteAddr)
		if !limiter.Allow() {
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
			for key, val := range args {
				req.Header.Set(key, val)
			}
			route.handler(res, req)
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
