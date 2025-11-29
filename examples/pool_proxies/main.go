package main

import (
	"log"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	proxies := []string{
		"user:pass@127.0.0.1:8080",
		"127.0.0.1:8081",
		"socks5://127.0.0.1:9050",
	}

	pool := v2.NewProxyClientPool(proxies, 2)
	if pool == nil {
		log.Fatal("no proxies configured")
	}

	var req v2.Request
	var resp v2.Response

	req.SetRequestURI("https://httpbin.org/get")
	req.Header.SetMethod("GET")

	for i := 0; i < 10; i++ {
		if err := pool.Do(&req, &resp); err != nil {
			log.Println("request error:", err)
			continue
		}
		log.Printf("req #%d status=%d", i+1, resp.StatusCode())
	}
}
