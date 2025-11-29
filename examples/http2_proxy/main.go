package main

import (
	"crypto/tls"
	"log"
	"time"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	opt := v2.ClientOptions{
		HTTPVersion:                   v2.HTTP2,
		MaxConnsPerHost:               1000,
		MaxIdleConnDuration:           30 * time.Second,
		ReadTimeout:                   10 * time.Second,
		WriteTimeout:                  10 * time.Second,
		ProxyHTTP:                     "127.0.0.1:8080",
		NoDefaultUserAgentHeader:      true,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	c := v2.NewClientWithOptions(opt)

	body, status, err := c.GetBytes("https://httpbin.org/get")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("status=%d body=%s", status, string(body))
}
