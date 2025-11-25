package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"v2fasthttp/client"
)

func main() {
	cfg := client.DefaultConfig()
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

	body, status, err := c.GetBytes(ctx, "https://localhost:8443/")
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	fmt.Printf("status=%d body=%s\n", status, string(body))
}
