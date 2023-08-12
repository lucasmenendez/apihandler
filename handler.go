// apihandler package provides a simple http.Handler implementation with a REST
// friendly API syntax. Provides simple methods to assign handlers to a path by
// HTTP method. It supports the following HTTP methods: GET, HEAD, POST, PUT,
// PATCH, DELETE, CONNECT, OPTIONS & TRACE.
package apihandler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

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

// Handler struct cotains the list of assigned routes and also an error channel
// to listen to raised errors using `Handler.Error(error)`.
type Handler struct {
	Errors chan error
	mtx    *sync.Mutex
	routes []*route
}

// New function returns a Handler initialized and read-to-use.
func New() *Handler {
	return &Handler{
		Errors: make(chan error),
		mtx:    &sync.Mutex{},
		routes: []*route{},
	}
}

// Error method writes the provided error into the Handler error channel.
func (m *Handler) Error(err error) {
	m.Errors <- err
}

// ServerHTTP method implements `http.Handler` interface. This funcion is
// executed when a request is received. It checks if the handler has a route
// assigned with the request method and path to execute the route handler. If
// it is not registered yet, the function sends a response with a 405 HTTP
// error.
func (m *Handler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	if route, exist := m.find(req.Method, req.RequestURI); exist {
		args, _ := parseArgs(req.RequestURI, route.rgx)
		for key, val := range args {
			req.Header.Set(key, val)
		}

		route.handler(res, req)
		return
	}
	res.WriteHeader(http.StatusMethodNotAllowed)
	if _, err := res.Write([]byte("405 method not allowed")); err != nil {
		m.Error(fmt.Errorf("{%s} %s %w", req.Method, req.RequestURI, err))
	}
}

// HandleFunc method assign the provided handler for requests sent to the
// desired method and path. It checks if the method provided is already
// supported before assign it. It also transform the provided path into a regex
// and assign it to the created route. If already exists a route with the same
// method and path, it will be overwritten.
func (m *Handler) HandleFunc(method, path string, handler func(http.ResponseWriter, *http.Request)) {
	for _, supported := range supportedMethods {
		if supported == method {
			m.mtx.Lock()
			defer m.mtx.Unlock()
			rgx, err := pathToRegex(path)
			if err != nil {
				m.Error(fmt.Errorf("error parsing route '%s': %w", path, err))
				return
			}
			// try to overwrite if already exist a registered handler for it
			for i, r := range m.routes {
				if r.method == method && r.path == path {
					m.routes[i] = &route{
						method:  method,
						path:    path,
						rgx:     rgx,
						handler: handler,
					}
					return
				}
			}
			// if it does not exists, create it
			m.routes = append(m.routes, &route{
				method:  method,
				path:    path,
				rgx:     rgx,
				handler: handler,
			})
			return
		}
	}
}

// Get method wraps `Handler.HandleFunc` for HTTP method 'GET'.
func (m *Handler) Get(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodGet, p, h)
}

// Head method wraps `Handler.HandleFunc` for HTTP method 'HEAD'.
func (m *Handler) Head(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodHead, p, h)
}

// Post method wraps `Handler.HandleFunc` for HTTP method 'POST'.
func (m *Handler) Post(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodPost, p, h)
}

// Put method wraps `Handler.HandleFunc` for HTTP method 'PUT'.
func (m *Handler) Put(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodPut, p, h)
}

// Patch method wraps `Handler.HandleFunc` for HTTP method 'PATCH'.
func (m *Handler) Patch(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodPatch, p, h)
}

// Delete method wraps `Handler.HandleFunc` for HTTP method 'DELETE'.
func (m *Handler) Delete(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodDelete, p, h)
}

// Connect method wraps `Handler.HandleFunc` for HTTP method 'CONNECT'.
func (m *Handler) Connect(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodConnect, p, h)
}

// Options method wraps `Handler.HandleFunc` for HTTP method 'OPTIONS'.
func (m *Handler) Options(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodOptions, p, h)
}

// Trace method wraps `Handler.HandleFunc` for HTTP method 'TRACE'.
func (m *Handler) Trace(p string, h func(http.ResponseWriter, *http.Request)) {
	m.HandleFunc(http.MethodTrace, p, h)
}

// find method search for a registered handler for the method and request URI
// provided, matching the routes regex with the URI provided. If the route is
// not registered, it returns also false.
func (m *Handler) find(method, requestURI string) (*route, bool) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	for _, r := range m.routes {
		if r.method == method && r.rgx.MatchString(requestURI) {
			return r, true
		}
	}
	return nil, false
}

// pathToRegex function transforms the provided path into a regex to match with
// the URI of incoming requests. The resulting regex will be able to match
// named arguments from a request URI.
func pathToRegex(path string) (*regexp.Regexp, error) {
	rgx := argsToRgx.ReplaceAllString(path, argsToRgxSub)
	escapedRgx := strings.ReplaceAll(rgx, "/", "\\/")
	return regexp.Compile(escapedRgx)
}

// parseArgs function returns if the request URI matches with the route regex
// provided and the named arguments that the URI could contain.
func parseArgs(requestURI string, routeRgx *regexp.Regexp) (map[string]string, bool) {
	// check if matches
	if !routeRgx.MatchString(requestURI) {
		return nil, false
	}
	// find named arguments
	args := make(map[string]string)
	matches := routeRgx.FindStringSubmatch(requestURI)
	for i, name := range routeRgx.SubexpNames()[0:] {
		args[name] = matches[i]
	}
	return args, true
}
