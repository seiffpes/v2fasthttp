package main

import (
	"log"
	"time"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	// GET كسلسلة نصية
	s, status, err := v2.GetStringURL("https://httpbin.org/get")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("GET status=%d body=%s", status, s)

	// POST body بايتس مع timeout
	body := []byte(`{"hello":"world"}`)
	s, status, err = v2.PostStringTimeoutURL("https://httpbin.org/post", body, 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("POST status=%d body=%s", status, s)
}
