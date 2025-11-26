package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/seiffpes/v2fasthttp/client"
)

// Basic example: net/http-based client from the low-level package.
// Demonstrates a simple GET to a public endpoint.
func main() {
	cfg := client.DefaultConfig()

	// Optional: tune a few settings.
	cfg.MaxConnsPerHost = 128
	cfg.IdleConnTimeout = 60 * time.Second

	c, err := client.New(cfg)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	data, status, err := c.GetBytes(ctx, "https://httpbin.org/get")
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	fmt.Printf("status=%d body=%s\n", status, string(data))
}
