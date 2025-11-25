package v2fasthttp

import (
	"github.com/seiffpes/v2fasthttp/client"
	"github.com/seiffpes/v2fasthttp/server"
)

type (
	ClientConfig = client.Config
	Client       = client.Client

	ServerConfig = server.Config
	Server       = server.Server

	RequestCtx     = server.RequestCtx
	RequestHandler = server.RequestHandler

	Router     = server.Router
	WSUpgrader = server.WSUpgrader
)

var ErrBodyTooLarge = client.ErrBodyTooLarge

func NewClient(cfg ClientConfig) (*Client, error) {
	return client.New(cfg)
}

func DefaultClientConfig() ClientConfig {
	return client.DefaultConfig()
}

func NewServer(handler RequestHandler, cfg ServerConfig) *Server {
	return server.NewFast(handler, cfg)
}

func DefaultServerConfig() ServerConfig {
	return server.DefaultConfig()
}

func NewRouter() *Router {
	return server.NewRouter()
}

func NewWSUpgrader() *WSUpgrader {
	return server.NewWSUpgrader()
}
