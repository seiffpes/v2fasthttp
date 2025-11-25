package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

type Config struct {
	Addr string

	ReadTimeout time.Duration

	ReadHeaderTimeout time.Duration

	WriteTimeout time.Duration

	IdleTimeout time.Duration

	MaxHeaderBytes int

	TLSConfig *tls.Config

	EnableHTTP3 bool

	HTTP3Addr string
}

func DefaultConfig() Config {
	return Config{
		Addr:              ":8080",
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      0, // no hard write timeout for streaming
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
}

type Server struct {
	httpServer  *http.Server
	http3Server *http3.Server
}

func New(handler http.Handler, cfg Config) *Server {
	def := DefaultConfig()
	if cfg.Addr == "" {
		cfg.Addr = def.Addr
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = def.ReadTimeout
	}
	if cfg.ReadHeaderTimeout == 0 {
		cfg.ReadHeaderTimeout = def.ReadHeaderTimeout
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = def.IdleTimeout
	}
	if cfg.MaxHeaderBytes == 0 {
		cfg.MaxHeaderBytes = def.MaxHeaderBytes
	}

	s := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
		TLSConfig:         cfg.TLSConfig,
	}

	var h3Server *http3.Server

	if cfg.TLSConfig != nil {
		http2Server := &http2.Server{}
		_ = http2.ConfigureServer(s, http2Server)

		if cfg.EnableHTTP3 {
			addr := cfg.Addr
			if cfg.HTTP3Addr != "" {
				addr = cfg.HTTP3Addr
			}

			h3TLS := cfg.TLSConfig.Clone()
			h3TLS = http3.ConfigureTLSConfig(h3TLS)

			h3Server = &http3.Server{
				Addr:           addr,
				Handler:        handler,
				TLSConfig:      h3TLS,
				MaxHeaderBytes: cfg.MaxHeaderBytes,
			}
		}
	}

	return &Server{
		httpServer:  s,
		http3Server: h3Server,
	}
}

func NewFast(handler RequestHandler, cfg Config) *Server {
	return New(HandlerToHTTP(handler), cfg)
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	if s.http3Server == nil {
		return s.httpServer.ListenAndServeTLS(certFile, keyFile)
	}

	errCh := make(chan error, 2)

	go func() {
		errCh <- s.httpServer.ListenAndServeTLS(certFile, keyFile)
	}()

	go func() {
		errCh <- s.http3Server.ListenAndServeTLS(certFile, keyFile)
	}()

	return <-errCh
}

func (s *Server) Serve(l net.Listener) error {
	return s.httpServer.Serve(l)
}

func (s *Server) Shutdown(ctx context.Context) error {
	err := s.httpServer.Shutdown(ctx)
	if s.http3Server != nil {
		if err2 := s.http3Server.Shutdown(ctx); err == nil && err2 != nil {
			err = err2
		}
	}
	return err
}
