package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/seiffpes/v2fasthttp"
)

func main() {
	// HTTP/1.1 only client.
	http1Cfg := v2fasthttp.DefaultClientConfig()
	http1Cfg.DisableHTTP2 = true
	http1Client, err := v2fasthttp.NewClient(http1Cfg)
	if err != nil {
		log.Fatalf("new http1 client: %v", err)
	}

	// HTTP/1.1 + HTTP/2 (default) client.
	http2Cfg := v2fasthttp.DefaultClientConfig()
	http2Client, err := v2fasthttp.NewClient(http2Cfg)
	if err != nil {
		log.Fatalf("new http2 client: %v", err)
	}

	// HTTP/1.1 + HTTP/2 + HTTP/3 client (requires examples/h2h3/server).
	http3Cfg := v2fasthttp.DefaultClientConfig()
	http3Cfg.EnableHTTP3 = true
	http3Cfg.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	http3Client, err := v2fasthttp.NewClient(http3Cfg)
	if err != nil {
		log.Fatalf("new http3 client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use the fasthttp-style Request/Response API with multiple clients.
	doAll(ctx, "http://localhost:8080", http1Client, "http1")
	doAll(ctx, "http://localhost:8080", http2Client, "http2")

	// For HTTP/3, run examples/h2h3/server first.
	doGetOnly(ctx, "https://localhost:8443/", http3Client, "http3")

	// Also show the high level helpers.
	useHelpers(ctx, http2Client)
}

func doAll(ctx context.Context, base string, c *v2fasthttp.Client, name string) {
	fmt.Println("==== using client:", name, "====")

	// Simple GET.
	req := v2fasthttp.AcquireRequest()
	resp := v2fasthttp.AcquireResponse()
	defer v2fasthttp.ReleaseRequest(req)
	defer v2fasthttp.ReleaseResponse(resp)

	req.SetMethod(http.MethodGet)
	req.SetRequestURI(base + "/hello?name=" + name)
	if err := v2fasthttp.DoWithClient(ctx, c, req, resp); err != nil {
		log.Printf("[%s] GET error: %v\n", name, err)
	} else {
		fmt.Printf("[%s] GET status=%d body=%s\n", name, resp.StatusCode, resp.Body)
	}

	// POST.
	resp.Reset()
	req.Reset()
	req.SetMethod(http.MethodPost)
	req.SetRequestURI(base + "/echo")
	req.SetBody([]byte("hello from POST " + name))
	if err := v2fasthttp.DoWithClient(ctx, c, req, resp); err != nil {
		log.Printf("[%s] POST error: %v\n", name, err)
	} else {
		fmt.Printf("[%s] POST status=%d body=%s\n", name, resp.StatusCode, resp.Body)
	}

	// DELETE.
	resp.Reset()
	req.Reset()
	req.SetMethod(http.MethodDelete)
	req.SetRequestURI(base + "/resource/123")
	if err := v2fasthttp.DoWithClient(ctx, c, req, resp); err != nil {
		log.Printf("[%s] DELETE error: %v\n", name, err)
	} else {
		fmt.Printf("[%s] DELETE status=%d body=%s\n", name, resp.StatusCode, resp.Body)
	}
}

func doGetOnly(ctx context.Context, url string, c *v2fasthttp.Client, name string) {
	req := v2fasthttp.AcquireRequest()
	resp := v2fasthttp.AcquireResponse()
	defer v2fasthttp.ReleaseRequest(req)
	defer v2fasthttp.ReleaseResponse(resp)

	req.SetMethod(http.MethodGet)
	req.SetRequestURI(url)
	if err := v2fasthttp.DoWithClient(ctx, c, req, resp); err != nil {
		log.Printf("[%s] HTTP/3 GET error: %v\n", name, err)
		return
	}
	fmt.Printf("[%s] HTTP/3 GET status=%d body=%s\n", name, resp.StatusCode, resp.Body)
}

func useHelpers(ctx context.Context, c *v2fasthttp.Client) {
	fmt.Println("==== using helpers (GET/POST/DELETE) ====")

	data, status, err := c.GetBytes(ctx, "http://localhost:8080/")
	if err != nil {
		log.Printf("helpers GET error: %v\n", err)
	} else {
		fmt.Printf("helpers GET status=%d body=%s\n", status, string(data))
	}

	resp, err := c.PostBytes(ctx, "http://localhost:8080/echo", "text/plain", []byte("hello via helpers POST"))
	if err != nil {
		log.Printf("helpers POST error: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("helpers POST status=%d body=%s\n", resp.StatusCode, string(body))
	}

	resp, err = c.Delete(ctx, "http://localhost:8080/resource/456")
	if err != nil {
		log.Printf("helpers DELETE error: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("helpers DELETE status=%d body=%s\n", resp.StatusCode, string(body))
	}
}
