package main

import (
	"crypto/tls"
	"log"

	"github.com/quic-go/quic-go/http3"
	"github.com/seiffpes/v2fasthttp/server"
)

func main() {
	router := server.NewRouter()

	router.GET("/", func(ctx *server.RequestCtx) {
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.WriteString("hello from h1/h2/h3 server\n")
	})

	certFile := "server.crt"
	keyFile := "server.key"

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("load cert: %v", err)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{http3.NextProtoH3, "h2", "http/1.1"},
	}

	cfg := server.DefaultConfig()
	cfg.Addr = ":8443"
	cfg.TLSConfig = tlsCfg
	cfg.EnableHTTP3 = true

	s := server.NewFast(router.Handler, cfg)

	log.Println("serving HTTPS on https://localhost:8443 (h1/h2/h3)")
	if err := s.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatal(err)
	}
}
