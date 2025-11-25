package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/seiffpes/v2fasthttp"
)

// Example: net/http-based client from your library
// with HTTP/1.1 + HTTP/2 + optional HTTP/3 and proxy support.
func main() {
	cfg := v2fasthttp.DefaultClientConfig()

	// Enable HTTP/3 (requires remote server that supports h3).
	cfg.EnableHTTP3 = true
	cfg.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	// Configure basic connection options.
	cfg.MaxConnsPerHost = 512
	cfg.MaxIdleConns = 1024
	cfg.MaxIdleConnsPerHost = 512

	// Optional: HTTP or SOCKS proxy.
	// cfg.SetProxyHTTP("127.0.0.1:8080")
	// cfg.SetSOCKS5Proxy("socks5://127.0.0.1:9050")

	client, err := v2fasthttp.NewClient(cfg)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use helper GET/POST methods.
	doHTTPGet(ctx, client, "https://httpbin.org/get")
	doHTTPPost(ctx, client, "https://httpbin.org/post")
}

func doHTTPGet(ctx context.Context, c *v2fasthttp.Client, url string) {
	body, status, err := c.GetBytes(ctx, url)
	if err != nil {
		log.Printf("[http2/3] GET error: %v", err)
		return
	}
	fmt.Printf("[http2/3] GET %s status=%d body=%s\n", url, status, string(body))
}

func doHTTPPost(ctx context.Context, c *v2fasthttp.Client, url string) {
	resp, err := c.PostBytes(ctx, url, "application/json", []byte(`{"msg":"hello from http2/3 client"}`))
	if err != nil {
		log.Printf("[http2/3] POST error: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[http2/3] POST read error: %v", err)
		return
	}
	fmt.Printf("[http2/3] POST %s status=%d body=%s\n", url, resp.StatusCode, string(body))
}
