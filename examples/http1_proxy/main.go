package main

import (
	"log"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	c := v2.NewHighPerfClient("")

	// HTTP proxy, بدون user/pass في المثال
	c.SetProxyHTTP("127.0.0.1:8080")

	body, status, err := c.GetBytes("https://httpbin.org/ip")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("status=%d body=%s", status, string(body))
}
