package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"v2fasthttp"
)

// This example shows how to configure the global
// fasthttp-style client (Do/Get/Post helpers) so you
// can tune MaxConnsPerHost, timeouts, HTTP2/HTTP3, etc.
func main() {
	// 1) Configure the default client once at startup.
	cfg := v2fasthttp.DefaultClientConfig()

	// Tune connection limits.
	cfg.MaxConnsPerHost = 1024
	cfg.MaxIdleConns = 4096
	cfg.MaxIdleConnsPerHost = 1024

	// Timeouts.
	cfg.DialTimeout = 3 * time.Second
	cfg.IdleConnTimeout = 60 * time.Second

	// Protocols.
	cfg.DisableHTTP2 = false // use HTTP/1.1 + HTTP/2
	cfg.EnableHTTP3 = false  // turn on only if you have an h3 server

	// Optional: name and user agent.
	cfg.Name = "v2fasthttp-default-client"

	if err := v2fasthttp.SetDefaultClientConfig(cfg); err != nil {
		log.Fatalf("SetDefaultClientConfig: %v", err)
	}

	// 2) Simple GET using global helpers (like fasthttp.Get).
	resp := v2fasthttp.AcquireResponse()
	defer v2fasthttp.ReleaseResponse(resp)

	if err := v2fasthttp.Get("http://localhost:8080/hello?name=default", resp); err != nil {
		log.Fatalf("global GET: %v", err)
	}
	fmt.Printf("GET /hello status=%d body=%s\n", resp.StatusCode, resp.Body)

	// 3) POST using the global helpers.
	resp.Reset()
	body := []byte("hello from default client")
	if err := v2fasthttp.Post("http://localhost:8080/echo", body, resp); err != nil {
		log.Fatalf("global POST: %v", err)
	}
	fmt.Printf("POST /echo status=%d body=%s\n", resp.StatusCode, resp.Body)

	// 4) Using the full Request/Response API with DoTimeout.
	req := v2fasthttp.AcquireRequest()
	defer v2fasthttp.ReleaseRequest(req)
	resp.Reset()

	req.SetMethod(http.MethodDelete)
	req.SetRequestURI("http://localhost:8080/resource/777")

	if err := v2fasthttp.DoTimeout(req, resp, 3*time.Second); err != nil {
		log.Fatalf("global DoTimeout DELETE: %v", err)
	}
	fmt.Printf("DELETE /resource/777 status=%d body=%s\n", resp.StatusCode, resp.Body)

	// 5) Using DoWithClient with the same default client instance.
	c, err := v2fasthttp.NewClient(cfg)
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	req.Reset()
	resp.Reset()
	req.SetMethod(http.MethodGet)
	req.SetRequestURI("http://localhost:8080/")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := v2fasthttp.DoWithClient(ctx, c, req, resp); err != nil {
		log.Fatalf("DoWithClient: %v", err)
	}
	fmt.Printf("GET / status=%d body=%s\n", resp.StatusCode, resp.Body)
}

