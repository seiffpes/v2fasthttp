package main

import (
	"log"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	var req v2.Request
	var resp v2.Response

	req.SetRequestURI("https://httpbin.org/get")
	req.Header.SetMethod("GET")

	if err := v2.Do(&req, &resp); err != nil {
		log.Fatal(err)
	}

	log.Printf("status=%d body=%s", resp.StatusCode(), resp.Body())
}
