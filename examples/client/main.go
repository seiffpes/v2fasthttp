package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/seiffpes/v2fasthttp/client"
)

func main() {
	cfg := client.DefaultConfig()

	c, err := client.New(cfg)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	data, status, err := c.GetBytes(ctx, "http://localhost:8080/")
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	fmt.Printf("status=%d body=%s\n", status, string(data))
}
