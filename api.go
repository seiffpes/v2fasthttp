package v2fasthttp

import (
	"github.com/seiffpes/v2fasthttp/client"
	"github.com/seiffpes/v2fasthttp/fastclient"
	"github.com/valyala/fasthttp"
)

type (
	// ClientConfig is the configuration for the net/http-based client
	// that supports HTTP/1.1, HTTP/2 and HTTP/3.
	ClientConfig = client.Config
	// Client is the high-level net/http-based client.
	Client = client.Client

	// FastClient is a fasthttp-based client that mirrors
	// fasthttp.Client behavior but lives inside this library.
	// Use it when you want HTTP/1.1-only, fasthttp-style performance
	// and configuration.
	FastClient = fastclient.Client

	// FastRequest and FastResponse are aliases for fasthttp types
	// so callers can use them via this library without importing
	// fasthttp directly.
	FastRequest  = fasthttp.Request
	FastResponse = fasthttp.Response
)

var ErrBodyTooLarge = client.ErrBodyTooLarge

func NewClient(cfg ClientConfig) (*Client, error) {
	return client.New(cfg)
}

func DefaultClientConfig() ClientConfig {
	return client.DefaultConfig()
}
