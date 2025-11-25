package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/seiffpes/v2fasthttp"
	"github.com/valyala/fasthttp"
)

// Example: HTTP/1.1-only client built on fasthttp via v2fasthttp.FastClient.
// Shows GET and POST with and without proxy.
func main() {
	// Build a fasthttp-style client using your library.
	client := &v2fasthttp.FastClient{
		Client: fasthttp.Client{
			MaxConnsPerHost:               100000,
			MaxIdleConnDuration:           100 * time.Millisecond,
			NoDefaultUserAgentHeader:      true,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Optional: enable HTTP proxy for all requests.
	// Accepts "ip:port" or "user:pass@ip:port".
	// client.SetProxyHTTP("127.0.0.1:8080")

	// Simple GET.
	if err := doFastGet(client, "https://httpbin.org/get"); err != nil {
		log.Fatalf("fastclient GET error: %v", err)
	}

	// Simple POST.
	if err := doFastPost(client, "https://httpbin.org/post", []byte("hello from fastclient")); err != nil {
		log.Fatalf("fastclient POST error: %v", err)
	}
}

func doFastGet(c *v2fasthttp.FastClient, url string) error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodGet)

	if err := c.Do(req, resp); err != nil {
		return err
	}

	fmt.Printf("[fastclient] GET %s status=%d body=%s\n", url, resp.StatusCode(), resp.Body())
	return nil
}

func doFastPost(c *v2fasthttp.FastClient, url string, body []byte) error {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetBody(body)

	if err := c.Do(req, resp); err != nil {
		return err
	}

	fmt.Printf("[fastclient] POST %s status=%d body=%s\n", url, resp.StatusCode(), resp.Body())
	return nil
}

