package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/seiffpes/v2fasthttp"
)

// Example: configure the global client once and use the package-level
// helpers (Get/Post/Do) in a fasthttp-style way.
func main() {
	// 1) Configure the default client once at startup.
	cfg := v2fasthttp.DefaultClientConfig()

	// Tune connection limits.
	cfg.MaxConnsPerHost = 512
	cfg.MaxIdleConns = 2048
	cfg.MaxIdleConnsPerHost = 512

	// Timeouts.
	cfg.DialTimeout = 3 * time.Second
	cfg.IdleConnTimeout = 60 * time.Second

	// Protocols.
	cfg.DisableHTTP2 = false // HTTP/1.1 + HTTP/2
	cfg.EnableHTTP3 = false  // set to true if the remote supports h3

	// Optional: name and user agent.
	cfg.Name = "v2fasthttp-default-client"

	if err := v2fasthttp.SetDefaultClientConfig(cfg); err != nil {
		log.Fatalf("SetDefaultClientConfig: %v", err)
	}

	// 2) Simple GET using global helpers (like fasthttp.Get).
	resp := v2fasthttp.AcquireResponse()
	defer v2fasthttp.ReleaseResponse(resp)

	if err := v2fasthttp.Get("https://httpbin.org/get?name=default", resp); err != nil {
		log.Fatalf("global GET: %v", err)
	}
	fmt.Printf("GET status=%d body=%s\n", resp.StatusCode, resp.Body)

	// 3) POST using the global helpers.
	resp.Reset()
	body := []byte("hello from default client")
	if err := v2fasthttp.Post("https://httpbin.org/post", body, resp); err != nil {
		log.Fatalf("global POST: %v", err)
	}
	fmt.Printf("POST status=%d body=%s\n", resp.StatusCode, resp.Body)

	// 4) Using the full Request/Response API with DoTimeout.
	req := v2fasthttp.AcquireRequest()
	defer v2fasthttp.ReleaseRequest(req)
	resp.Reset()

	req.SetMethod(http.MethodGet)
	req.SetRequestURI("https://httpbin.org/headers")
	req.SetHeader("X-Example", "v2fasthttp")

	if err := v2fasthttp.DoTimeout(req, resp, 3*time.Second); err != nil {
		log.Fatalf("global DoTimeout GET: %v", err)
	}
	fmt.Printf("GET /headers status=%d body=%s\n", resp.StatusCode, resp.Body)

	// 5) Using DoWithClient with a dedicated client instance.
	c, err := v2fasthttp.NewClient(cfg)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	req.Reset()
	resp.Reset()
	req.SetMethod(http.MethodGet)
	req.SetRequestURI("https://httpbin.org/ip")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := v2fasthttp.DoWithClient(ctx, c, req, resp); err != nil {
		log.Fatalf("DoWithClient: %v", err)
	}
	fmt.Printf("GET /ip status=%d body=%s\n", resp.StatusCode, resp.Body)
}
