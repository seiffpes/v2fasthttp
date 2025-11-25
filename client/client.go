package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type Config struct {
	MaxConnsPerHost int

	MaxIdleConns int

	MaxIdleConnsPerHost int

	IdleConnTimeout time.Duration

	DialTimeout time.Duration

	TLSHandshakeTimeout time.Duration

	ExpectContinueTimeout time.Duration

	DisableCompression bool

	// DisableHTTP2 disables HTTP/2 support on the underlying
	// net/http Transport. When set, all requests will use HTTP/1.1
	// over TCP/TLS (with optional HTTP/3 if enabled).
	DisableHTTP2 bool

	ProxyURL string

	ProxyUsername string
	ProxyPassword string

	ProxyDialTimeout      time.Duration
	ProxyHandshakeTimeout time.Duration

	MaxResponseBodySize int64

	Name               string
	NoDefaultUserAgent bool

	MaxIdemponentCallAttempts int
	RetryIf                   func(req *http.Request, resp *http.Response, err error) bool

	OnRequest  func(req *http.Request)
	OnResponse func(resp *http.Response)
	OnRetry    func(req *http.Request, attempt int, err error)
	OnError    func(req *http.Request, err error)

	// EnableHTTP3 enables an additional HTTP/3 client for HTTPS
	// requests when there is no proxy configured.
	EnableHTTP3 bool

	TLSClientConfig *tls.Config
}

func DefaultConfig() Config {
	return Config{
		MaxConnsPerHost:       512,
		MaxIdleConns:          2048,
		MaxIdleConnsPerHost:   512,
		IdleConnTimeout:       90 * time.Second,
		DialTimeout:           5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}
}

func (c *Config) SetHTTPProxy(hostport string) {
	c.ProxyURL = "http://" + hostport
}

func (c *Config) SetSOCKS5Proxy(hostport string) {
	c.ProxyURL = "socks5://" + hostport
}

func (c *Config) SetSOCKS4Proxy(hostport string) {
	c.ProxyURL = "socks4://" + hostport
}

var ErrBodyTooLarge = errors.New("v2fasthttp: response body too large")

type Client struct {
	httpClient *http.Client

	bufPool sync.Pool

	http3 http3Client

	name                      string
	noDefaultUserAgent        bool
	maxResponseBodySize       int64
	maxIdemponentCallAttempts int
	retryIf                   func(req *http.Request, resp *http.Response, err error) bool

	onRequest  func(req *http.Request)
	onResponse func(resp *http.Response)
	onRetry    func(req *http.Request, attempt int, err error)
	onError    func(req *http.Request, err error)
}

func New(cfg Config) (*Client, error) {
	applyDefaults(&cfg)

	dialer := &net.Dialer{
		Timeout:   cfg.DialTimeout,
		KeepAlive: 30 * time.Second,
	}

	proxyFunc, dialContext, err := buildProxy(cfg, dialer)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy:                 proxyFunc,
		DialContext:           dialContext,
		ForceAttemptHTTP2:     !cfg.DisableHTTP2,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ExpectContinueTimeout: cfg.ExpectContinueTimeout,
		DisableCompression:    cfg.DisableCompression,
		TLSClientConfig:       cfg.TLSClientConfig,
	}

	if !cfg.DisableHTTP2 {
		if err := http2.ConfigureTransport(transport); err != nil {
			return nil, err
		}
	}

	c := &Client{
		httpClient: &http.Client{
			Transport: transport,
		},
		bufPool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 32*1024))
			},
		},
	}

	c.http3 = newHTTP3Client(cfg)

	c.name = cfg.Name
	c.noDefaultUserAgent = cfg.NoDefaultUserAgent
	c.maxResponseBodySize = cfg.MaxResponseBodySize
	c.maxIdemponentCallAttempts = cfg.MaxIdemponentCallAttempts
	c.retryIf = cfg.RetryIf

	c.onRequest = cfg.OnRequest
	c.onResponse = cfg.OnResponse
	c.onRetry = cfg.OnRetry
	c.onError = cfg.OnError

	return c, nil
}

func applyDefaults(cfg *Config) {
	def := DefaultConfig()
	if cfg.MaxConnsPerHost == 0 {
		cfg.MaxConnsPerHost = def.MaxConnsPerHost
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = def.MaxIdleConns
	}
	if cfg.MaxIdleConnsPerHost == 0 {
		cfg.MaxIdleConnsPerHost = def.MaxIdleConnsPerHost
	}
	if cfg.IdleConnTimeout == 0 {
		cfg.IdleConnTimeout = def.IdleConnTimeout
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = def.DialTimeout
	}
	if cfg.TLSHandshakeTimeout == 0 {
		cfg.TLSHandshakeTimeout = def.TLSHandshakeTimeout
	}
	if cfg.ExpectContinueTimeout == 0 {
		cfg.ExpectContinueTimeout = def.ExpectContinueTimeout
	}
	if cfg.ProxyDialTimeout == 0 {
		cfg.ProxyDialTimeout = cfg.DialTimeout
	}
	if cfg.ProxyHandshakeTimeout == 0 {
		cfg.ProxyHandshakeTimeout = cfg.DialTimeout
	}
	if cfg.MaxIdemponentCallAttempts <= 0 {
		cfg.MaxIdemponentCallAttempts = 1
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	maxAttempts := c.maxIdemponentCallAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	idempotent := isIdempotentMethod(req.Method)
	if !idempotent || (req.Body != nil && req.GetBody == nil) {
		maxAttempts = 1
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 && req.Body != nil && req.GetBody != nil {
			var rb io.ReadCloser
			rb, err = req.GetBody()
			if err != nil {
				break
			}
			req.Body = rb
		}

		resp, err = c.doOnce(req)

		if err != nil && c.onError != nil {
			c.onError(req, err)
		}

		shouldRetry := false
		if c.retryIf != nil {
			shouldRetry = c.retryIf(req, resp, err)
		} else {
			shouldRetry = err != nil && idempotent
		}

		if !shouldRetry || attempt+1 >= maxAttempts {
			break
		}

		if c.onRetry != nil {
			c.onRetry(req, attempt+1, err)
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	return resp, err
}

func (c *Client) GetBytes(ctx context.Context, url string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	buf := c.bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer c.bufPool.Put(buf)

	var reader io.Reader = resp.Body
	if c.maxResponseBodySize > 0 {
		reader = &io.LimitedReader{
			R: resp.Body,
			N: c.maxResponseBodySize + 1,
		}
	}

	_, copyErr := io.Copy(buf, reader)
	if copyErr != nil && copyErr != io.EOF {
		return nil, resp.StatusCode, copyErr
	}

	if c.maxResponseBodySize > 0 && int64(buf.Len()) > c.maxResponseBodySize {
		return nil, resp.StatusCode, ErrBodyTooLarge
	}

	data := make([]byte, buf.Len())
	copy(data, buf.Bytes())

	return data, resp.StatusCode, nil
}

func (c *Client) CloseIdleConnections() {
	if tr, ok := c.httpClient.Transport.(interface{ CloseIdleConnections() }); ok {
		tr.CloseIdleConnections()
	}
	if c.http3 != nil {
		c.http3.CloseIdleConnections()
	}
}

func (c *Client) doOnce(req *http.Request) (*http.Response, error) {
	if c.onRequest != nil {
		c.onRequest(req)
	}

	if !c.noDefaultUserAgent {
		if req.Header.Get("User-Agent") == "" {
			if c.name != "" {
				req.Header.Set("User-Agent", c.name)
			} else {
				req.Header.Set("User-Agent", "v2fasthttp-client")
			}
		}
	}

	if c.http3 != nil {
		if resp, ok, err := c.http3MaybeDo(c.http3, req); ok || err != nil {
			if err == nil && c.onResponse != nil {
				c.onResponse(resp)
			}
			return resp, err
		}
	}

	resp, err := c.httpClient.Do(req)
	if err == nil && c.onResponse != nil {
		c.onResponse(resp)
	}
	return resp, err
}

func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodPut, http.MethodTrace:
		return true
	default:
		return false
	}
}
