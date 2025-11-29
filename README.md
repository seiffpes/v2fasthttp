# v2fasthttp

`v2fasthttp` is a high-performance HTTP client toolkit for Go built directly on top of [`github.com/valyala/fasthttp`](https://github.com/valyala/fasthttp).

The focus is:
 - Fast HTTP/1.1 requests using the fasthttp client.
 - Simple, fasthttp-style API: `Request`, `Response`, `Do`, `Get`, `Post`, etc.
 - Strong proxy support (HTTP and SOCKS5) with helpers.
 - Multi-client pools for very high QPS workloads.
 - Optional HTTP/2 and HTTP/3 clients built on top of `net/http` and `quic-go`, while keeping the fasthttp-style `Request` / `Response` API.

## Installation

```bash
go get github.com/seiffpes/v2fasthttp
```

## Core types

Package `v2fasthttp` exposes the main fasthttp types:

 - `type Client struct{ fasthttp.Client /* + HTTP2/3 client */ }`
 - `type Request = fasthttp.Request`
 - `type Response = fasthttp.Response`
 - `type RequestCtx = fasthttp.RequestCtx`
 - `type RequestHandler = fasthttp.RequestHandler`

There is also a global default client used by package-level helpers.

## Quick start

Simple GET using the default client:

```go
package main

import (
	"log"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	var req v2.Request
	var resp v2.Response

	req.SetRequestURI("https://httpbin.org/get")
	req.Header.SetMethod("GET")

	if err := v2.Do(&req, &resp); err != nil {
		log.Fatal(err)
	}

	log.Printf("status=%d body=%s", resp.StatusCode(), resp.Body())
}
```

Using the high-level helpers:

```go
body, status, err := v2.GetBytesURL("https://httpbin.org/get")
if err != nil {
	log.Fatal(err)
}
log.Printf("status=%d body=%s", status, string(body))
```

## High-performance client

You can fully control the underlying client via `ClientOptions`:

```go
opt := v2.ClientOptions{
	HTTPVersion:                  v2.HTTP1, // or v2.HTTP2 / v2.HTTP3
	MaxConnsPerHost:               100000,
	MaxIdleConnDuration:           100 * time.Millisecond,
	ReadBufferSize:                64 * 1024,
	WriteBufferSize:               64 * 1024,
	MaxIdemponentCallAttempts:     1,
	NoDefaultUserAgentHeader:      true,
	DisableHeaderNamesNormalizing: true,
	DisablePathNormalizing:        true,
	MaxConnWaitTimeout:            time.Second,
	TLSConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}

c := v2.NewClientWithOptions(opt)
```

For a ready-to-use aggressive configuration there is:

```go
c := v2.NewHighPerfClient("")
```

This is tuned for very high QPS and is what the benchmark example uses.

## HTTP/2 and HTTP/3

`ClientOptions` has an `HTTPVersion` field that lets you switch the transport behind the same fasthttp-style API:

```go
// HTTP/2 client with an HTTP proxy
opt := v2.ClientOptions{
	HTTPVersion: v2.HTTP2,
	ProxyHTTP:   "user:pass@127.0.0.1:8080",
	TLSConfig:   &tls.Config{InsecureSkipVerify: true},
}

c := v2.NewClientWithOptions(opt)

var req v2.Request
var resp v2.Response

req.SetRequestURI("https://example.com/")
req.Header.SetMethod("GET")

if err := c.Do(&req, &resp); err != nil {
	log.Fatal(err)
}
```

HTTP/1.1 (`HTTP1`) continues to use the native `fasthttp.Client`. HTTP/2 uses a tuned `net/http.Client` under the hood, and HTTP/3 uses `quic-go`'s HTTP/3 transport. Proxy helpers (`SetProxy`, `SetProxyHTTP`, `SetSOCKS5Proxy`, `SetProxyFromEnvironment`) work with HTTP/1.1 and HTTP/2.

If you set `HTTPVersion: HTTP3` together with `ProxyHTTP` or `SOCKS5Proxy`, the client will automatically fall back to `HTTP2`, since HTTP/3 over HTTP or SOCKS5 proxies is not supported in this package.

## Proxy support

### Per-client proxy

On any `*Client` you can set proxies:

```go
// HTTP proxy, with or without auth.
c.SetProxyHTTP("127.0.0.1:8080")
c.SetProxyHTTP("user:pass@127.0.0.1:8080")

// SOCKS5 proxy.
c.SetSOCKS5Proxy("socks5://127.0.0.1:9050")

// Auto-detect (socks5:// → SOCKS5, otherwise HTTP).
c.SetProxy("user:pass@127.0.0.1:8080")
c.SetProxy("socks5://127.0.0.1:9050")

// Use HTTP(S)_PROXY / NO_PROXY from the environment.
c.SetProxyFromEnvironment()
c.SetProxyFromEnvironmentTimeout(2 * time.Second)
```

You can also configure proxies through `ClientOptions` (`ProxyHTTP`, `SOCKS5Proxy`) and build the client with `NewClientWithOptions`.

### Multi-client pools and proxy lists

For high concurrency and proxy lists there is a small pool type:

```go
pool := v2.NewHighPerfClientPool(200, "user:pass@127.0.0.1:8080")

var req v2.Request
var resp v2.Response

req.SetRequestURI("https://httpbin.org/get")
req.Header.SetMethod("GET")

if err := pool.Do(&req, &resp); err != nil {
	log.Fatal(err)
}
```

To use multiple proxies:

```go
proxies := []string{
	"user:pass@127.0.0.1:8080",
	"127.0.0.1:8081",
	"socks5://127.0.0.1:9050",
}

pool := v2.NewProxyClientPool(proxies, 4)
```

Or from a string list (newline / comma / space separated):

```go
list := "user:pass@127.0.0.1:8080\n127.0.0.1:8081\nsocks5://127.0.0.1:9050"
pool := v2.NewProxyClientPoolFromString(list, 4)
```

The pool does round-robin over all clients and proxies.

## Byte and JSON helpers

Common helpers on `*Client`:

 - `DoBytes`, `DoBytesTimeout`
 - `GetBytes`, `GetBytesTimeout`
 - `PostBytes`, `PostBytesTimeout`
 - `PostJSON`, `PostJSONTimeout`
 - `GetString`, `GetStringTimeout`
 - `PostString`, `PostStringTimeout`

And global helpers using the default client:

 - `GetBytesURL`, `GetBytesTimeoutURL`
 - `PostBytesURL`, `PostBytesTimeoutURL`
 - `PostJSONURL`, `PostJSONTimeoutURL`
 - `GetStringURL`, `GetStringTimeoutURL`
 - `PostStringURL`, `PostStringTimeoutURL`

These keep the fasthttp style but make basic HTTP requests quick to write.

## Benchmark example

`examples/bench` is a small benchmark tool that can hit a URL with many concurrent clients (optionally through a proxy) and print the achieved requests per second.

From the project root:

```bash
go run ./examples/bench -url https://httpbin.org/get -total 200000 -concurrency 200

# With an HTTP proxy
go run ./examples/bench -url https://httpbin.org/get -total 200000 -concurrency 200 -proxy user:pass@127.0.0.1:8080
```

## License

Licensed under the MIT License.  
Copyright (c) 2025 fpes.
