[![Last release](https://img.shields.io/github/v/release/lucasmenendez/apihandler?color=purple)](https://github.com/lucasmenendez/apihandler/releases/latest)
[![GoDoc](https://godoc.org/github.com/lucasmenendez/apihandler?status.svg)](https://godoc.org/github.com/lucasmenendez/apihandler) 
[![Go Report Card](https://goreportcard.com/badge/github.com/lucasmenendez/apihandler)](https://goreportcard.com/report/github.com/lucasmenendez/apihandler)
[![test](https://github.com/lucasmenendez/apihandler/workflows/test/badge.svg)](https://github.com/lucasmenendez/apihandler/actions?query=workflow%3Atest)
[![license](https://img.shields.io/github/license/lucasmenendez/apihandler)](LICENSE)


# APIHandler

apihandler package provides a simple `http.Handler` implementation with a REST friendly API syntax. Provides simple methods to assign handlers to a path by HTTP method.

It also supports the configuration of a rate limit by request origin to prevent abuse.

### Installation
```sh
go get github.com/lucasmenendez/apihandler
```

## Basic example

Check out the [full example here](example_test.go).

```go 
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// create and register a new GET handler with cors enabled and a rate limit
// of one request per second
handler := NewHandler(true, RateLimiter(ctx, 1, 1, time.Minute
err := handler.Get("/service/{service_name}/resource/{resource_name}",
    func(w http.ResponseWriter, r *http.Request) {
        // get router arguments from Header
        status := map[string]string{
            "service":  apihandler.URIParam(r.Context(), "service_name"),
            "resource": apihandler.URIParam(r.Context(), "resource_name"),
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
    })
if err != nil {
    log.Printf("ERR: error listening for requests: %s\n", err)
}
// run http server with created handler
if err := http.ListenAndServe(":8090", handler); err != nil {
    log.Printf("ERR: error listening for requests: %s\n", err)
    return
}
```
