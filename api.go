package v2fasthttp

import (
	"github.com/seiffpes/v2fasthttp/client"
	"github.com/seiffpes/v2fasthttp/fastclient"
	"github.com/valyala/fasthttp"
)

type (
	ClientConfig = client.Config
	Client       = client.Client

	FastClient  = fastclient.Client
	FastRequest = fasthttp.Request
	FastResponse = fasthttp.Response
)

var ErrBodyTooLarge = client.ErrBodyTooLarge

func NewClient(cfg ClientConfig) (*Client, error) {
	return client.New(cfg)
}

func DefaultClientConfig() ClientConfig {
	return client.DefaultConfig()
}
