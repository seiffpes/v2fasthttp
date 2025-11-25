package fastclient

import (
	"crypto/tls"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)


type Client struct {
	fasthttp.Client
}

// DefaultClient returns a Client with sensible defaults similar to fasthttp.
func DefaultClient() *Client {
	return &Client{
		Client: fasthttp.Client{
			MaxIdleConnDuration:      90 * time.Second,
			NoDefaultUserAgentHeader: true,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
}

// SetProxyHTTP configures HTTP proxy dialing using a single string,
// compatible with fasthttpproxy.FasthttpHTTPDialer.
//
// Examples:
//   "127.0.0.1:8080"
//   "user:pass@127.0.0.1:8080"
func (c *Client) SetProxyHTTP(proxy string) {
	c.Dial = fasthttpproxy.FasthttpHTTPDialer(proxy)
}

// SetProxyHTTPTimeout is like SetProxyHTTP but allows specifying
// a custom dial timeout.
func (c *Client) SetProxyHTTPTimeout(proxy string, timeout time.Duration) {
	c.Dial = fasthttpproxy.FasthttpHTTPDialerTimeout(proxy, timeout)
}

// SetSOCKS5Proxy configures a SOCKS5 proxy dialer using the given address.
//
// Example:
//   "socks5://127.0.0.1:9050"
func (c *Client) SetSOCKS5Proxy(proxyAddr string) {
	c.Dial = fasthttpproxy.FasthttpSocksDialer(proxyAddr)
}

