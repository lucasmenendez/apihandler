[![GoDoc](https://godoc.org/github.com/lucasmenendez/apihandler?status.svg)](https://godoc.org/github.com/lucasmenendez/apihandler) 
[![Go Report Card](https://goreportcard.com/badge/github.com/lucasmenendez/apihandler)](https://goreportcard.com/report/github.com/lucasmenendez/apihandler)
[![test](https://github.com/lucasmenendez/apihandler/workflows/test/badge.svg)](https://github.com/lucasmenendez/apihandler/actions?query=workflow%3Atest)
[![license](https://img.shields.io/github/license/lucasmenendez/apihandler)](LICENSE)


# APIHandler

apihandler package provides a simple `http.Handler` implementation with a REST friendly API syntax. Provides simple methods to assign handlers to a path by HTTP method.

### Installation
```sh
go get github.com/lucasmenendez/apihandler
```

## Basic example
```go 
package main

import (
    "fmt"
    "net/http"

    "github.com/lucasmenendez/apihandler"
)

func main() {
    // create handler and register a GET handler function on '/count' path
    handler := apihandler.New()
    handler.Get("/count", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("{\"count\": 100}"))
    })
    // run a goroutine to handle internal handler errors
    go func() {
        for err := range handler.Errors {
            fmt.Printf("ERR: internal error: %s\n",err)
        }
    }()
    // run http server with created handler
    if err := http.ListenAndServe(":8080", handler); err != nil {
        fmt.Printf("ERR: error listening for requests: %s\n", err)
        return
    }
}

```
