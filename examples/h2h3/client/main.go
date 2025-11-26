package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/seiffpes/v2fasthttp/client"
)

// Example: HTTP/2/HTTP/3 client against a remote endpoint that
// supports h2/h3. Adjust the URL to point to a server you control.
func main() {
	cfg := client.DefaultConfig()

	// Enable HTTP/3 (requires remote server with h3 support).
	cfg.EnableHTTP3 = true
	cfg.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	c, err := client.New(cfg)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Replace this URL with an h2/h3-capable endpoint.
	url := "https://httpbin.org/get"
	body, status, err := c.GetBytes(ctx, url)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	fmt.Printf("GET %s status=%d body=%s\n", url, status, string(body))
}
