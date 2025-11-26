package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
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

	DisableHTTP2 bool

	ProxyURL string

	ProxyHTTP string

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

	EnableHTTP3 bool

	TLSClientConfig *tls.Config

	Dial func(ctx context.Context, network, addr string) (net.Conn, error)

	MaxIdleConnDuration time.Duration

	NoDefaultUserAgentHeader bool

	TLSConfig *tls.Config

	MaxConnWaitTimeout        time.Duration
	DisableHeaderNamesNormalizing bool
	DisablePathNormalizing        bool
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

func (c *Config) SetProxyHTTP(addr string) {
	c.SetHTTPProxy(addr)
}

func (c *Config) SetProxy(proxy string) {
	c.ProxyURL = proxy
}

func (c *Config) SetSOCKS5Proxy(hostport string) {
	c.ProxyURL = "socks5://" + hostport
}

func (c *Config) SetSOCKS4Proxy(hostport string) {
	c.ProxyURL = "socks4://" + hostport
}

func (c *Config) SetProxyAuth(username, password string) {
	c.ProxyUsername = username
	c.ProxyPassword = password
}

func (c *Client) SetHTTPProxy(hostport string) {
	c.Config.SetHTTPProxy(hostport)
}

func (c *Client) SetProxyHTTP(addr string) {
	c.Config.SetProxyHTTP(addr)
}

func (c *Client) SetSOCKS5Proxy(hostport string) {
	c.Config.SetSOCKS5Proxy(hostport)
}

func (c *Client) SetSOCKS4Proxy(hostport string) {
	c.Config.SetSOCKS4Proxy(hostport)
}

func (c *Client) SetProxy(proxy string) {
	c.Config.SetProxy(proxy)
}

func (c *Client) SetProxyAuth(username, password string) {
	c.Config.SetProxyAuth(username, password)
}

var ErrBodyTooLarge = errors.New("v2fasthttp: response body too large")

type Client struct {
	Config

	httpClient *http.Client

	bufPool sync.Pool

	http3 http3Client

	initOnce sync.Once
	initErr  error

	proxyAuthorization string
}

func New(cfg Config) (*Client, error) {
	c := &Client{
		Config: cfg,
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) init() error {
	c.initOnce.Do(func() {
		cfg := c.Config
		applyDefaults(&cfg)

		var proxyFunc func(*http.Request) (*url.URL, error)
		var dialContext func(context.Context, string, string) (net.Conn, error)
		var err error

		if cfg.Dial != nil {
			proxyFunc = nil
			dialContext = cfg.Dial
		} else {
			dialer := &net.Dialer{
				Timeout:   cfg.DialTimeout,
				KeepAlive: 30 * time.Second,
			}

			proxyFunc, dialContext, err = buildProxy(cfg, dialer)
			if err != nil {
				c.initErr = err
				return
			}
		}

		var proxyAuthHeader string
		if cfg.ProxyURL != "" {
			if u, perr := url.Parse(cfg.ProxyURL); perr == nil {
				username := cfg.ProxyUsername
				password := cfg.ProxyPassword
				if u.User != nil {
					username = u.User.Username()
					if p, ok := u.User.Password(); ok {
						password = p
					}
				}
				scheme := strings.ToLower(u.Scheme)
				if (scheme == "http" || scheme == "https") && (username != "" || password != "") {
					creds := username + ":" + password
					proxyAuthHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
				}
			}
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

		if proxyAuthHeader != "" {
			transport.ProxyConnectHeader = http.Header{
				"Proxy-Authorization": []string{proxyAuthHeader},
			}
		}

		if !cfg.DisableHTTP2 {
			if err := http2.ConfigureTransport(transport); err != nil {
				c.initErr = err
				return
			}
		}

		c.httpClient = &http.Client{
			Transport: transport,
		}

		c.bufPool = sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 32*1024))
			},
		}

		c.http3 = newHTTP3Client(cfg)

		// Store back the normalized config (with defaults applied).
		c.Config = cfg
		c.proxyAuthorization = proxyAuthHeader
	})
	return c.initErr
}

func applyDefaults(cfg *Config) {
	def := DefaultConfig()
	if cfg.ProxyURL == "" && cfg.ProxyHTTP != "" {
		cfg.ProxyURL = cfg.ProxyHTTP
	}
	if cfg.MaxIdleConnDuration > 0 && cfg.IdleConnTimeout == 0 {
		cfg.IdleConnTimeout = cfg.MaxIdleConnDuration
	}
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
	if cfg.NoDefaultUserAgentHeader {
		cfg.NoDefaultUserAgent = true
	}
	if cfg.TLSClientConfig == nil && cfg.TLSConfig != nil {
		cfg.TLSClientConfig = cfg.TLSConfig
	}
	if cfg.MaxIdemponentCallAttempts <= 0 {
		cfg.MaxIdemponentCallAttempts = 1
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if err := c.init(); err != nil {
		return nil, err
	}

	maxAttempts := c.MaxIdemponentCallAttempts
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

		if err != nil && c.OnError != nil {
			c.OnError(req, err)
		}

		shouldRetry := false
		if c.RetryIf != nil {
			shouldRetry = c.RetryIf(req, resp, err)
		} else {
			shouldRetry = err != nil && idempotent
		}

		if !shouldRetry || attempt+1 >= maxAttempts {
			break
		}

		if c.OnRetry != nil {
			c.OnRetry(req, attempt+1, err)
		}

		if resp != nil {
			resp.Body.Close()
		}
	}

	return resp, err
}

func (c *Client) GetBytes(ctx context.Context, url string) ([]byte, int, error) {
	if err := c.init(); err != nil {
		return nil, 0, err
	}

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
	if c.MaxResponseBodySize > 0 {
		reader = &io.LimitedReader{
			R: resp.Body,
			N: c.MaxResponseBodySize + 1,
		}
	}

	_, copyErr := io.Copy(buf, reader)
	if copyErr != nil && copyErr != io.EOF {
		return nil, resp.StatusCode, copyErr
	}

	if c.MaxResponseBodySize > 0 && int64(buf.Len()) > c.MaxResponseBodySize {
		return nil, resp.StatusCode, ErrBodyTooLarge
	}

	data := make([]byte, buf.Len())
	copy(data, buf.Bytes())

	return data, resp.StatusCode, nil
}

func (c *Client) CloseIdleConnections() {
	if err := c.init(); err != nil {
		return
	}

	if tr, ok := c.httpClient.Transport.(interface{ CloseIdleConnections() }); ok {
		tr.CloseIdleConnections()
	}
	if c.http3 != nil {
		c.http3.CloseIdleConnections()
	}
}

func (c *Client) doOnce(req *http.Request) (*http.Response, error) {
	if err := c.init(); err != nil {
		return nil, err
	}

	if c.proxyAuthorization != "" && req.Header.Get("Proxy-Authorization") == "" {
		req.Header.Set("Proxy-Authorization", c.proxyAuthorization)
	}

	if c.OnRequest != nil {
		c.OnRequest(req)
	}

	if !c.NoDefaultUserAgent {
		if req.Header.Get("User-Agent") == "" {
			if c.Name != "" {
				req.Header.Set("User-Agent", c.Name)
			} else {
				req.Header.Set("User-Agent", "v2fasthttp-client")
			}
		}
	}

	if c.http3 != nil {
		if resp, ok, err := c.http3MaybeDo(c.http3, req); ok || err != nil {
			if err == nil && c.OnResponse != nil {
				c.OnResponse(resp)
			}
			return resp, err
		}
	}

	resp, err := c.httpClient.Do(req)
	if err == nil && c.OnResponse != nil {
		c.OnResponse(resp)
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
