package main

import (
	"crypto/tls"
	"log"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	opt := v2.ClientOptions{
		HTTPVersion: v2.HTTP3,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	c := v2.NewClientWithOptions(opt)

	var req v2.Request
	var resp v2.Response

	req.SetRequestURI("https://cloudflare-quic.com/")
	req.Header.SetMethod("GET")

	if err := c.Do(&req, &resp); err != nil {
		log.Fatal(err)
	}

	log.Printf("status=%d len=%d", resp.StatusCode(), len(resp.Body()))
}
