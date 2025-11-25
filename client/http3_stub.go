package client

import (
	"net/http"

	"github.com/quic-go/quic-go/http3"
)

type http3Client interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

type quicHTTP3Client struct {
	client    *http.Client
	transport *http3.Transport
}

func newHTTP3Client(cfg Config) http3Client {
	if !cfg.EnableHTTP3 {
		return nil
	}
	if cfg.ProxyURL != "" || cfg.ProxyUsername != "" || cfg.ProxyPassword != "" {
		return nil
	}

	tr := &http3.Transport{
		TLSClientConfig:    cfg.TLSClientConfig,
		DisableCompression: cfg.DisableCompression,
	}

	return &quicHTTP3Client{
		client: &http.Client{
			Transport: tr,
		},
		transport: tr,
	}
}

func (c *quicHTTP3Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *quicHTTP3Client) CloseIdleConnections() {
	c.transport.CloseIdleConnections()
}

func (c *Client) http3MaybeDo(h3 http3Client, req *http.Request) (*http.Response, bool, error) {
	if h3 == nil || req.URL == nil || req.URL.Scheme != "https" {
		return nil, false, nil
	}

	resp, err := h3.Do(req)
	return resp, true, err
}
