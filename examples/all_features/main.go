package main

import (
	"crypto/tls"
	"log"
	"time"

	"github.com/valyala/fasthttp"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	// -------- Default client (package-level helpers) --------

	// Low-level Do / DoTimeout with Request / Response
	{
		var req v2.Request
		var resp v2.Response

		req.SetRequestURI("https://httpbin.org/get")
		req.Header.SetMethod("GET")

		_ = v2.Do(&req, &resp)
		_ = v2.DoTimeout(&req, &resp, 2*time.Second)
	}

	// Get / GetTimeout / Post on the default client
	{
		_, _, _ = v2.Get(nil, "https://httpbin.org/get")
		_, _, _ = v2.GetTimeout(nil, "https://httpbin.org/get", 2*time.Second)

		args := fasthttp.AcquireArgs()
		args.Set("foo", "bar")
		_, _, _ = v2.Post(nil, "https://httpbin.org/post", args)
		fasthttp.ReleaseArgs(args)
	}

	// -------- Per-client usage (HTTP/1) --------

	client := &v2.Client{}

	// Proxy helpers on Client
	client.SetProxyHTTP("127.0.0.1:8080")
	client.SetSOCKS5Proxy("socks5://127.0.0.1:9050")
	client.SetProxy("127.0.0.1:8080")
	client.SetProxy("socks5://127.0.0.1:9050")
	client.SetProxyFromEnvironment()
	client.SetProxyFromEnvironmentTimeout(2 * time.Second)

	// Byte helpers on Client
	_, _, _ = client.DoBytes("GET", "https://httpbin.org/get", nil)
	_, _, _ = client.DoBytesTimeout("POST", "https://httpbin.org/post", []byte("body"), 2*time.Second)
	_, _, _ = client.GetBytes("https://httpbin.org/get")
	_, _, _ = client.GetBytesTimeout("https://httpbin.org/get", 2*time.Second)
	_, _, _ = client.PostBytes("https://httpbin.org/post", []byte("body"))
	_, _, _ = client.PostBytesTimeout("https://httpbin.org/post", []byte("body"), 2*time.Second)

	// JSON helpers on Client
	payload := map[string]any{"foo": "bar"}
	_, _, _ = client.PostJSON("https://httpbin.org/post", payload)
	_, _, _ = client.PostJSONTimeout("https://httpbin.org/post", payload, 2*time.Second)

	// String helpers on Client
	_, _, _ = client.GetString("https://httpbin.org/get")
	_, _, _ = client.GetStringTimeout("https://httpbin.org/get", 2*time.Second)
	_, _, _ = client.PostString("https://httpbin.org/post", []byte("body"))
	_, _, _ = client.PostStringTimeout("https://httpbin.org/post", []byte("body"), 2*time.Second)

	// -------- Package-level URL helpers --------

	_, _, _ = v2.GetBytesURL("https://httpbin.org/get")
	_, _, _ = v2.GetBytesTimeoutURL("https://httpbin.org/get", 2*time.Second)
	_, _, _ = v2.PostBytesURL("https://httpbin.org/post", []byte("body"))
	_, _, _ = v2.PostBytesTimeoutURL("https://httpbin.org/post", []byte("body"), 2*time.Second)
	_, _, _ = v2.PostJSONURL("https://httpbin.org/post", payload)
	_, _, _ = v2.PostJSONTimeoutURL("https://httpbin.org/post", payload, 2*time.Second)

	_, _, _ = v2.GetStringURL("https://httpbin.org/get")
	_, _, _ = v2.GetStringTimeoutURL("https://httpbin.org/get", 2*time.Second)
	_, _, _ = v2.PostStringURL("https://httpbin.org/post", []byte("body"))
	_, _, _ = v2.PostStringTimeoutURL("https://httpbin.org/post", []byte("body"), 2*time.Second)

	// -------- HTTP/2 / HTTP/3 clients via ClientOptions --------

	http2Client := v2.NewClientWithOptions(v2.ClientOptions{
		HTTPVersion:                   v2.HTTP2,
		MaxConnsPerHost:               1000,
		MaxIdleConnDuration:           30 * time.Second,
		ReadTimeout:                   5 * time.Second,
		WriteTimeout:                  5 * time.Second,
		ProxyHTTP:                     "127.0.0.1:8080",
		NoDefaultUserAgentHeader:      true,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	http3Client := v2.NewClientWithOptions(v2.ClientOptions{
		HTTPVersion: v2.HTTP3,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	_ = http2Client
	_ = http3Client

	// -------- High-performance client and pools --------

	highPerf := v2.NewHighPerfClient("")
	_ = highPerf

	// Generic client pool
	pool := v2.NewClientPool(5, func() *v2.Client {
		return v2.NewHighPerfClient("")
	})

	if pool != nil {
		var req v2.Request
		var resp v2.Response

		req.SetRequestURI("https://httpbin.org/get")
		req.Header.SetMethod("GET")

		if err := pool.Do(&req, &resp); err != nil {
			log.Println("pool Do error:", err)
		}
	}

	// High-performance pool with a single proxy
	hpPool := v2.NewHighPerfClientPool(10, "127.0.0.1:8080")
	if hpPool != nil {
		var req v2.Request
		var resp v2.Response

		req.SetRequestURI("https://httpbin.org/get")
		req.Header.SetMethod("GET")

		if err := hpPool.Do(&req, &resp); err != nil {
			log.Println("high-perf pool Do error:", err)
		}
	}

	// Proxy pools from slice and from string
	proxyPool := v2.NewProxyClientPool([]string{"127.0.0.1:8080", "socks5://127.0.0.1:9050"}, 2)
	stringPool := v2.NewProxyClientPoolFromString("127.0.0.1:8080\nsocks5://127.0.0.1:9050", 1)

	_ = proxyPool
	_ = stringPool
}
