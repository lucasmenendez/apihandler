// apihandler package provides a simple http.Handler implementation with a REST
// friendly API syntax. Provides simple methods to assign handlers to a path by
// HTTP method. It supports the following HTTP methods: GET, HEAD, POST, PUT,
// PATCH, DELETE, CONNECT, OPTIONS & TRACE.
package apihandler

import (
	"fmt"
	"net/http"
	"sync"
)

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

// Handler struct cotains the list of assigned routes and also an error channel
// to listen to raised errors using `Handler.Error(error)`.
type Handler struct {
	Errors chan error
	mtx    *sync.Mutex
	routes map[string]func(http.ResponseWriter, *http.Request)
}

// New function returns a Handler initialized and read-to-use.
func New() *Handler {
	return &Handler{
		Errors: make(chan error),
		mtx:    &sync.Mutex{},
		routes: map[string]func(http.ResponseWriter, *http.Request){},
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
	if handler := m.find(req.Method, req.RequestURI); handler != nil {
		handler(res, req)
		return
	}
	res.WriteHeader(http.StatusMethodNotAllowed)
	if _, err := res.Write([]byte("405 method not allowed")); err != nil {
		m.Error(fmt.Errorf("{%s} %s %w", req.Method, req.RequestURI, err))
	}
}

// HandleFunc method assign the provided handler for requests sent to the
// desired method and path. It checks if the method provided is already
// supported before assign it.
func (m *Handler) HandleFunc(method, path string, handler func(http.ResponseWriter, *http.Request)) {
	for _, supported := range supportedMethods {
		if supported == method {
			m.mtx.Lock()
			defer m.mtx.Unlock()
			m.routes[routeKey(method, path)] = handler
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

// find method search for a registered handler for the method and path provided,
// calculating the routeKey with that arguments. If the route is not registered,
// it returns nil.
func (m *Handler) find(method, path string) func(http.ResponseWriter, *http.Request) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	handler, found := m.routes[routeKey(method, path)]
	if !found {
		return nil
	}
	return handler
}

// routeKey function returns the key of a route with the method and path
// provided as argument. Both arguments will be joinned by '~'.
func routeKey(method, path string) string {
	return fmt.Sprintf("%s~%s", method, path)
}
