# v2fasthttp

v2fasthttp is a small, fast HTTP toolkit for Go built on top of `net/http` and `quic-go`.

It provides:

- A high‑performance client with HTTP/1.1, HTTP/2 and optional HTTP/3 support.
- A fasthttp‑style Request/Response API.
- A lightweight HTTP server with a simple router (params and wildcards).
- Multiple examples that cover common usage patterns.

> Note: this project is standalone and does not depend on the original `fasthttp` module.

## Installation

Use it as a normal Go module:

```bash
go get github.com/fpes/v2fasthttp
```

## Packages

- `v2fasthttp` (root):
  - `ClientConfig`, `Client`
  - `Request`, `Response` + helpers (`Do`, `Get`, `Post`, `Delete`, …)
  - `Session` (base URL + headers + auth)
  - `Server`, `Router`, `RequestCtx`
- `v2fasthttp/client`:
  - Low‑level client built on `net/http` with optional HTTP/3.
- `v2fasthttp/server`:
  - Lightweight HTTP server + router.

## Client – basic usage

```go
import (
    "context"
    "log"
    "time"

    "v2fasthttp"
)

func main() {
    cfg := v2fasthttp.DefaultClientConfig()

    // Fine‑tune the configuration.
    cfg.MaxConnsPerHost = 1024
    cfg.MaxIdleConns = 4096
    cfg.MaxIdleConnsPerHost = 1024
    cfg.DialTimeout = 3 * time.Second
    cfg.IdleConnTimeout = 60 * time.Second

    // Protocols.
    cfg.DisableHTTP2 = false // HTTP/1.1 + HTTP/2
    cfg.EnableHTTP3 = false  // enable only if you have an HTTP/3 server

    c, err := v2fasthttp.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    body, status, err := c.GetBytes(ctx, "http://localhost:8080/")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("status=%d body=%s", status, string(body))
}
```

## Client – fasthttp‑style Request/Response API

```go
req := v2fasthttp.AcquireRequest()
resp := v2fasthttp.AcquireResponse()
defer v2fasthttp.ReleaseRequest(req)
defer v2fasthttp.ReleaseResponse(resp)

req.SetMethod("GET")
req.SetRequestURI("http://localhost:8080/hello?name=world")

if err := v2fasthttp.Get("http://localhost:8080/hello?name=world", resp); err != nil {
    log.Fatal(err)
}
log.Printf("status=%d body=%s", resp.StatusCode, resp.Body)
```

### Configuring the default client

You can configure a single global client used by the helpers (`Do`, `Get`, `Post`, …):

```go
cfg := v2fasthttp.DefaultClientConfig()
cfg.MaxConnsPerHost = 1000
cfg.DisableHTTP2 = false
cfg.EnableHTTP3 = false

if err := v2fasthttp.SetDefaultClientConfig(cfg); err != nil {
    log.Fatal(err)
}

resp := v2fasthttp.AcquireResponse()
defer v2fasthttp.ReleaseResponse(resp)

if err := v2fasthttp.Get("http://localhost:8080/", resp); err != nil {
    log.Fatal(err)
}
```

Or provide a fully constructed client:

```go
c, _ := v2fasthttp.NewClient(cfg)
v2fasthttp.SetDefaultClient(c)
```

## Session API

`Session` makes API clients easier (base URL + headers + auth in one place):

```go
sess := v2fasthttp.NewSession(c).
    WithBaseURL("https://api.example.com/v1").
    WithBearer("TOKEN").
    WithHeader("X-App", "my-service")

resp, err := sess.Get(context.Background(), "/users")
if err != nil {
    log.Fatal(err)
}
```

## Server + Router

Example from `examples/server`:

```go
router := server.NewRouter()

router.GET("/", func(ctx *server.RequestCtx) {
    ctx.SetContentType("text/plain; charset=utf-8")
    ctx.WriteString("hello from v2fasthttp\n")
})

router.GET("/user/:id", func(ctx *server.RequestCtx) {
    id := ctx.UserValue("id")
    ctx.WriteString("user id = " + id)
})

router.POST("/echo", func(ctx *server.RequestCtx) {
    body, _ := io.ReadAll(ctx.Request().Body)
    ctx.Write(body)
})

s := server.NewFast(router.Handler, server.DefaultConfig())
if err := s.ListenAndServe(); err != nil {
    log.Fatal(err)
}
```

## Examples

- `examples/server`  
  Basic HTTP server using the router.

- `examples/client`  
  Simple client using `v2fasthttp/client` directly.

- `examples/requests`  
  Demonstrates multiple clients (HTTP/1 only, HTTP/1+2, HTTP/1+2+3) and
  GET/POST/DELETE using the Request/Response API and helpers.

- `examples/defaultclient`  
  Shows how to configure the global default client (`SetDefaultClientConfig`)
  and use the package‑level `Do/Get/Post` helpers.

- `examples/h2h3`  
  Server and client that support HTTP/2 and HTTP/3 (via `quic-go`).

## Running the examples

From the project root:

```bash
# HTTP server on :8080
go run ./examples/server

# Simple client
go run ./examples/client

# Multiple clients + GET/POST/DELETE
go run ./examples/requests

# Configure default client and use Do/Get/Post
go run ./examples/defaultclient

# h2/h3 server (requires TLS certs)
go run ./examples/h2h3/server

# h2/h3 client
go run ./examples/h2h3/client
```
## License

Licensed under the MIT License.  
See `LICENSE` for details.
