package v2fasthttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go/http3"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/http2"
	xnetproxy "golang.org/x/net/proxy"
)

type (
	HTTPVersion int

	Client struct {
		fasthttp.Client
		httpVersion HTTPVersion
		httpClient  *http.Client
	}
	Request        = fasthttp.Request
	Response       = fasthttp.Response
	RequestCtx     = fasthttp.RequestCtx
	RequestHandler = fasthttp.RequestHandler
)

const (
	HTTP1 HTTPVersion = iota + 1
	HTTP2
	HTTP3
)

var defaultClient = &Client{
	httpVersion: HTTP1,
}

func Do(req *Request, resp *Response) error {
	return defaultClient.Do(req, resp)
}

func DoTimeout(req *Request, resp *Response, timeout time.Duration) error {
	return defaultClient.DoTimeout(req, resp, timeout)
}

func Get(dst []byte, url string) (statusCode int, body []byte, err error) {
	return defaultClient.Get(dst, url)
}

func GetTimeout(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error) {
	return defaultClient.GetTimeout(dst, url, timeout)
}

func Post(dst []byte, url string, postArgs *fasthttp.Args) (statusCode int, body []byte, err error) {
	return defaultClient.Post(dst, url, postArgs)
}

func (c *Client) useNetHTTP() bool {
	return c != nil && (c.httpVersion == HTTP2 || c.httpVersion == HTTP3) && c.httpClient != nil
}

func (c *Client) Do(req *Request, resp *Response) error {
	if !c.useNetHTTP() {
		return c.Client.Do(req, resp)
	}
	httpReq, err := convertRequestToHTTP(req)
	if err != nil {
		return err
	}
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	return convertHTTPResponse(httpResp, resp)
}

func (c *Client) DoTimeout(req *Request, resp *Response, timeout time.Duration) error {
	if !c.useNetHTTP() {
		return c.Client.DoTimeout(req, resp, timeout)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	httpReq, err := convertRequestToHTTP(req)
	if err != nil {
		return err
	}
	httpReq = httpReq.WithContext(ctx)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	return convertHTTPResponse(httpResp, resp)
}

func (c *Client) SetProxyHTTP(proxy string) {
	if c == nil {
		return
	}
	c.Client.Dial = fasthttpproxy.FasthttpHTTPDialer(proxy)

	tr := trFromHTTPClient(c.httpClient)
	if tr == nil {
		return
	}
	u, err := parseProxyURL(proxy, "http")
	if err != nil {
		return
	}
	tr.Proxy = http.ProxyURL(u)
}

func (c *Client) SetSOCKS5Proxy(proxyAddr string) {
	if c == nil {
		return
	}
	c.Client.Dial = fasthttpproxy.FasthttpSocksDialer(proxyAddr)

	tr := trFromHTTPClient(c.httpClient)
	if tr == nil {
		return
	}
	setHTTPClientSOCKS5(tr, proxyAddr)
}

func (c *Client) SetProxy(proxy string) {
	if proxy == "" {
		c.Client.Dial = nil
		return
	}
	if strings.HasPrefix(proxy, "socks5://") {
		c.SetSOCKS5Proxy(proxy)
		return
	}
	c.SetProxyHTTP(proxy)
}

func (c *Client) SetProxyFromEnvironment() {
	if c == nil {
		return
	}
	c.Client.Dial = fasthttpproxy.FasthttpProxyHTTPDialer()

	tr := trFromHTTPClient(c.httpClient)
	if tr == nil {
		return
	}
	tr.Proxy = http.ProxyFromEnvironment
}

func (c *Client) SetProxyFromEnvironmentTimeout(timeout time.Duration) {
	if c == nil {
		return
	}
	c.Client.Dial = fasthttpproxy.FasthttpProxyHTTPDialerTimeout(timeout)

	tr := trFromHTTPClient(c.httpClient)
	if tr == nil {
		return
	}
	tr.Proxy = http.ProxyFromEnvironment
	if timeout > 0 && c.httpClient.Timeout == 0 {
		c.httpClient.Timeout = timeout
	}
}

func (c *Client) DoBytes(method, url string, body []byte) ([]byte, int, error) {
	var req Request
	var resp Response
	req.SetRequestURI(url)
	req.Header.SetMethod(method)
	if len(body) != 0 {
		req.SetBody(body)
	}
	if err := c.Do(&req, &resp); err != nil {
		return nil, 0, err
	}
	b := resp.Body()
	out := make([]byte, len(b))
	copy(out, b)
	return out, resp.StatusCode(), nil
}

func (c *Client) DoBytesTimeout(method, url string, body []byte, timeout time.Duration) ([]byte, int, error) {
	var req Request
	var resp Response
	req.SetRequestURI(url)
	req.Header.SetMethod(method)
	if len(body) != 0 {
		req.SetBody(body)
	}
	if err := c.DoTimeout(&req, &resp, timeout); err != nil {
		return nil, 0, err
	}
	b := resp.Body()
	out := make([]byte, len(b))
	copy(out, b)
	return out, resp.StatusCode(), nil
}

func (c *Client) GetBytes(url string) ([]byte, int, error) {
	return c.DoBytes(fasthttp.MethodGet, url, nil)
}

func (c *Client) GetBytesTimeout(url string, timeout time.Duration) ([]byte, int, error) {
	return c.DoBytesTimeout(fasthttp.MethodGet, url, nil, timeout)
}

func (c *Client) PostBytes(url string, body []byte) ([]byte, int, error) {
	return c.DoBytes(fasthttp.MethodPost, url, body)
}

func (c *Client) PostBytesTimeout(url string, body []byte, timeout time.Duration) ([]byte, int, error) {
	return c.DoBytesTimeout(fasthttp.MethodPost, url, body, timeout)
}

func (c *Client) PostJSON(url string, v any) ([]byte, int, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, 0, err
	}
	var req Request
	var resp Response
	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetBody(data)
	if err := c.Do(&req, &resp); err != nil {
		return nil, 0, err
	}
	b := resp.Body()
	out := make([]byte, len(b))
	copy(out, b)
	return out, resp.StatusCode(), nil
}

func (c *Client) PostJSONTimeout(url string, v any, timeout time.Duration) ([]byte, int, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, 0, err
	}
	var req Request
	var resp Response
	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/json")
	req.SetBody(data)
	if err := c.DoTimeout(&req, &resp, timeout); err != nil {
		return nil, 0, err
	}
	b := resp.Body()
	out := make([]byte, len(b))
	copy(out, b)
	return out, resp.StatusCode(), nil
}

func GetBytesURL(url string) ([]byte, int, error) {
	return defaultClient.GetBytes(url)
}

func GetBytesTimeoutURL(url string, timeout time.Duration) ([]byte, int, error) {
	return defaultClient.GetBytesTimeout(url, timeout)
}

func PostBytesURL(url string, body []byte) ([]byte, int, error) {
	return defaultClient.PostBytes(url, body)
}

func PostBytesTimeoutURL(url string, body []byte, timeout time.Duration) ([]byte, int, error) {
	return defaultClient.PostBytesTimeout(url, body, timeout)
}

func PostJSONURL(url string, v any) ([]byte, int, error) {
	return defaultClient.PostJSON(url, v)
}

func PostJSONTimeoutURL(url string, v any, timeout time.Duration) ([]byte, int, error) {
	return defaultClient.PostJSONTimeout(url, v, timeout)
}

func (c *Client) GetString(url string) (string, int, error) {
	b, status, err := c.GetBytes(url)
	return string(b), status, err
}

func (c *Client) GetStringTimeout(url string, timeout time.Duration) (string, int, error) {
	b, status, err := c.GetBytesTimeout(url, timeout)
	return string(b), status, err
}

func (c *Client) PostString(url string, body []byte) (string, int, error) {
	b, status, err := c.PostBytes(url, body)
	return string(b), status, err
}

func (c *Client) PostStringTimeout(url string, body []byte, timeout time.Duration) (string, int, error) {
	b, status, err := c.PostBytesTimeout(url, body, timeout)
	return string(b), status, err
}

func GetStringURL(url string) (string, int, error) {
	b, status, err := GetBytesURL(url)
	return string(b), status, err
}

func GetStringTimeoutURL(url string, timeout time.Duration) (string, int, error) {
	b, status, err := GetBytesTimeoutURL(url, timeout)
	return string(b), status, err
}

func PostStringURL(url string, body []byte) (string, int, error) {
	b, status, err := PostBytesURL(url, body)
	return string(b), status, err
}

func PostStringTimeoutURL(url string, body []byte, timeout time.Duration) (string, int, error) {
	b, status, err := PostBytesTimeoutURL(url, body, timeout)
	return string(b), status, err
}

type ClientOptions struct {
	HTTPVersion                   HTTPVersion
	MaxConnsPerHost               int
	MaxIdleConnDuration           time.Duration
	MaxConnDuration               time.Duration
	MaxIdemponentCallAttempts     int
	ReadBufferSize                int
	WriteBufferSize               int
	ReadTimeout                   time.Duration
	WriteTimeout                  time.Duration
	MaxResponseBodySize           int
	NoDefaultUserAgentHeader      bool
	DisableHeaderNamesNormalizing bool
	DisablePathNormalizing        bool
	MaxConnWaitTimeout            time.Duration
	TLSConfig                     *tls.Config
	ProxyHTTP                     string
	SOCKS5Proxy                   string
}

func NewClientWithOptions(opt ClientOptions) *Client {
	c := &Client{}

	if opt.HTTPVersion == 0 {
		opt.HTTPVersion = HTTP1
	}
	if opt.HTTPVersion == HTTP3 && (opt.ProxyHTTP != "" || opt.SOCKS5Proxy != "") {
		// HTTP/3 over HTTP/SOCKS5 proxies isn't supported; fall back to HTTP/2.
		opt.HTTPVersion = HTTP2
	}
	c.httpVersion = opt.HTTPVersion

	if opt.MaxConnsPerHost > 0 {
		c.MaxConnsPerHost = opt.MaxConnsPerHost
	} else {
		c.MaxConnsPerHost = 1024
	}
	if opt.MaxIdleConnDuration > 0 {
		c.MaxIdleConnDuration = opt.MaxIdleConnDuration
	} else {
		c.MaxIdleConnDuration = 90 * time.Second
	}
	if opt.MaxConnDuration > 0 {
		c.MaxConnDuration = opt.MaxConnDuration
	}
	if opt.MaxIdemponentCallAttempts > 0 {
		c.MaxIdemponentCallAttempts = opt.MaxIdemponentCallAttempts
	}
	if opt.ReadBufferSize > 0 {
		c.ReadBufferSize = opt.ReadBufferSize
	}
	if opt.WriteBufferSize > 0 {
		c.WriteBufferSize = opt.WriteBufferSize
	}
	if opt.ReadTimeout > 0 {
		c.ReadTimeout = opt.ReadTimeout
	}
	if opt.WriteTimeout > 0 {
		c.WriteTimeout = opt.WriteTimeout
	}
	if opt.MaxResponseBodySize > 0 {
		c.MaxResponseBodySize = opt.MaxResponseBodySize
	}
	c.NoDefaultUserAgentHeader = opt.NoDefaultUserAgentHeader
	c.DisableHeaderNamesNormalizing = opt.DisableHeaderNamesNormalizing
	c.DisablePathNormalizing = opt.DisablePathNormalizing
	if opt.MaxConnWaitTimeout > 0 {
		c.MaxConnWaitTimeout = opt.MaxConnWaitTimeout
	}
	c.TLSConfig = opt.TLSConfig

	if opt.HTTPVersion == HTTP2 || opt.HTTPVersion == HTTP3 {
		c.httpClient = newHTTPClient(opt.HTTPVersion, opt)
	}

	if opt.ProxyHTTP != "" {
		c.SetProxyHTTP(opt.ProxyHTTP)
	}
	if opt.SOCKS5Proxy != "" {
		c.SetSOCKS5Proxy(opt.SOCKS5Proxy)
	}

	return c
}

func newHTTPClient(version HTTPVersion, opt ClientOptions) *http.Client {
	timeout := opt.ReadTimeout
	if opt.WriteTimeout > timeout {
		timeout = opt.WriteTimeout
	}

	switch version {
	case HTTP2:
		tr := &http.Transport{
			TLSClientConfig: opt.TLSConfig,
		}
		if opt.MaxConnsPerHost > 0 {
			tr.MaxConnsPerHost = opt.MaxConnsPerHost
		}
		if opt.MaxIdleConnDuration > 0 {
			tr.IdleConnTimeout = opt.MaxIdleConnDuration
		}
		if opt.ReadBufferSize > 0 {
			tr.ReadBufferSize = opt.ReadBufferSize
		}
		if opt.WriteBufferSize > 0 {
			tr.WriteBufferSize = opt.WriteBufferSize
		}
		_ = http2.ConfigureTransport(tr)

		client := &http.Client{
			Transport: tr,
		}
		if timeout > 0 {
			client.Timeout = timeout
		}
		return client
	case HTTP3:
		rt := &http3.Transport{
			TLSClientConfig: opt.TLSConfig,
		}
		client := &http.Client{
			Transport: rt,
		}
		if timeout > 0 {
			client.Timeout = timeout
		}
		return client
	default:
		return nil
	}
}

func NewHighPerfClient(proxy string) *Client {
	opt := ClientOptions{
		HTTPVersion:                   HTTP1,
		MaxConnsPerHost:               100000,
		MaxIdleConnDuration:           100 * time.Millisecond,
		ReadBufferSize:                64 * 1024,
		WriteBufferSize:               64 * 1024,
		NoDefaultUserAgentHeader:      true,
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
	}
	c := NewClientWithOptions(opt)
	if proxy != "" {
		c.SetProxy(proxy)
	}
	return c
}

type ClientPool struct {
	clients []*Client
	idx     uint32
}

func NewClientPool(size int, factory func() *Client) *ClientPool {
	if size <= 0 {
		size = 1
	}
	clients := make([]*Client, size)
	for i := 0; i < size; i++ {
		c := factory()
		if c == nil {
			c = &Client{}
		}
		clients[i] = c
	}
	return &ClientPool{clients: clients}
}

func (p *ClientPool) Next() *Client {
	if p == nil || len(p.clients) == 0 {
		return nil
	}
	i := atomic.AddUint32(&p.idx, 1)
	return p.clients[i%uint32(len(p.clients))]
}

func (p *ClientPool) Do(req *Request, resp *Response) error {
	c := p.Next()
	if c == nil {
		return fasthttp.ErrNoFreeConns
	}
	return c.Do(req, resp)
}

func NewProxyClientPool(proxies []string, perProxy int) *ClientPool {
	if len(proxies) == 0 {
		return nil
	}
	if perProxy <= 0 {
		perProxy = 1
	}
	total := len(proxies) * perProxy
	clients := make([]*Client, 0, total)
	for _, pxy := range proxies {
		for i := 0; i < perProxy; i++ {
			clients = append(clients, NewHighPerfClient(pxy))
		}
	}
	return &ClientPool{clients: clients}
}

func NewHighPerfClientPool(size int, proxy string) *ClientPool {
	return NewClientPool(size, func() *Client {
		return NewHighPerfClient(proxy)
	})
}

func convertRequestToHTTP(req *Request) (*http.Request, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	uri := req.URI()
	urlStr := string(uri.FullURI())

	body := req.Body()
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	httpReq, err := http.NewRequest(string(req.Header.Method()), urlStr, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.VisitAll(func(k, v []byte) {
		key := string(k)
		value := string(v)
		if strings.EqualFold(key, "Host") {
			httpReq.Host = value
			return
		}
		httpReq.Header.Add(key, value)
	})

	return httpReq, nil
}

func convertHTTPResponse(httpResp *http.Response, resp *Response) error {
	if httpResp == nil || resp == nil {
		return nil
	}
	defer httpResp.Body.Close()

	resp.Reset()
	resp.SetStatusCode(httpResp.StatusCode)
	for k, values := range httpResp.Header {
		for _, v := range values {
			resp.Header.Add(k, v)
		}
	}
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}
	resp.SetBody(body)
	return nil
}

func parseProxyURL(proxyStr, defaultScheme string) (*url.URL, error) {
	if proxyStr == "" {
		return nil, errors.New("empty proxy")
	}
	if !strings.Contains(proxyStr, "://") {
		proxyStr = defaultScheme + "://" + proxyStr
	}
	return url.Parse(proxyStr)
}

func trFromHTTPClient(c *http.Client) *http.Transport {
	if c == nil {
		return nil
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		return nil
	}
	return tr
}

func setHTTPClientSOCKS5(tr *http.Transport, proxyAddr string) {
	if tr == nil || proxyAddr == "" {
		return
	}
	addr := strings.TrimPrefix(proxyAddr, "socks5://")
	dialer, err := xnetproxy.SOCKS5("tcp", addr, nil, xnetproxy.Direct)
	if err != nil {
		return
	}
	tr.Proxy = nil
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.Dial(network, addr)
	}
}

func NewProxyClientPoolFromString(list string, perProxy int) *ClientPool {
	if list == "" {
		return nil
	}
	fields := strings.FieldsFunc(list, func(r rune) bool {
		switch r {
		case '\n', '\r', '\t', ' ', ',', ';':
			return true
		default:
			return false
		}
	})
	if len(fields) == 0 {
		return nil
	}
	return NewProxyClientPool(fields, perProxy)
}
