package v2fasthttp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

type Request struct {
	Method string
	URI    string
	Header http.Header
	Body   []byte
}

var requestPool = sync.Pool{
	New: func() any {
		return &Request{
			Header: make(http.Header),
		}
	},
}

func AcquireRequest() *Request {
	r := requestPool.Get().(*Request)
	r.Reset()
	return r
}

func ReleaseRequest(r *Request) {
	if r == nil {
		return
	}
	r.Reset()
	requestPool.Put(r)
}

func (r *Request) Reset() {
	r.Method = ""
	r.URI = ""
	if r.Header == nil {
		r.Header = make(http.Header)
	} else {
		for k := range r.Header {
			delete(r.Header, k)
		}
	}
	if cap(r.Body) > 0 {
		r.Body = r.Body[:0]
	}
}

func (r *Request) SetRequestURI(uri string) {
	r.URI = uri
}

func (r *Request) SetMethod(method string) {
	r.Method = method
}

func (r *Request) SetBody(body []byte) {
	if len(body) == 0 {
		r.Body = r.Body[:0]
		return
	}
	r.Body = append(r.Body[:0], body...)
}

func (r *Request) HTTPRequest(ctx context.Context) (*http.Request, error) {
	method := r.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, r.URI, bytes.NewReader(r.Body))
	if err != nil {
		return nil, err
	}
	for k, values := range r.Header {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}
	return req, nil
}

func (r *Request) FromHTTP(req *http.Request) error {
	r.Reset()
	r.Method = req.Method
	if req.URL != nil {
		r.URI = req.URL.String()
	}
	for k, values := range req.Header {
		for _, v := range values {
			r.Header.Add(k, v)
		}
	}
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return err
		}
		r.Body = append(r.Body[:0], body...)
	}
	return nil
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

var responsePool = sync.Pool{
	New: func() any {
		return &Response{
			Header: make(http.Header),
		}
	},
}

func AcquireResponse() *Response {
	r := responsePool.Get().(*Response)
	r.Reset()
	return r
}

func ReleaseResponse(r *Response) {
	if r == nil {
		return
	}
	r.Reset()
	responsePool.Put(r)
}

func (r *Response) Reset() {
	r.StatusCode = 0
	if r.Header == nil {
		r.Header = make(http.Header)
	} else {
		for k := range r.Header {
			delete(r.Header, k)
		}
	}
	if cap(r.Body) > 0 {
		r.Body = r.Body[:0]
	}
}

func (r *Response) SetStatusCode(code int) {
	r.StatusCode = code
}

func (r *Response) SetBody(body []byte) {
	if len(body) == 0 {
		r.Body = r.Body[:0]
		return
	}
	r.Body = append(r.Body[:0], body...)
}

func (r *Response) FromHTTP(resp *http.Response) error {
	r.Reset()
	if resp == nil {
		return nil
	}
	r.StatusCode = resp.StatusCode
	for k, values := range resp.Header {
		for _, v := range values {
			r.Header.Add(k, v)
		}
	}
	if resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		r.Body = append(r.Body[:0], body...)
	}
	return nil
}

func (r *Response) WriteToHTTP(w http.ResponseWriter) error {
	for k, values := range r.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	if r.StatusCode != 0 {
		w.WriteHeader(r.StatusCode)
	}
	if len(r.Body) == 0 {
		return nil
	}
	_, err := w.Write(r.Body)
	return err
}

// WriteToCtx is kept for backward compatibility with older versions
// that exposed a server-side RequestCtx type. It is a no-op in the
// current client-only build.
func (r *Response) WriteToCtx(_ interface{}) error {
	return nil
}

func DoWithClient(ctx context.Context, c *Client, req *Request, resp *Response) error {
	httpReq, err := req.HTTPRequest(ctx)
	if err != nil {
		return err
	}
	httpResp, err := c.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	return resp.FromHTTP(httpResp)
}

var defaultClientOnce sync.Once
var defaultClientMu sync.RWMutex
var defaultClient *Client
var defaultClientErr error

func getDefaultClient() (*Client, error) {
	defaultClientOnce.Do(func() {
		cfg := DefaultClientConfig()
		c, err := NewClient(cfg)
		defaultClientMu.Lock()
		defaultClient = c
		defaultClientErr = err
		defaultClientMu.Unlock()
	})

	defaultClientMu.RLock()
	c, err := defaultClient, defaultClientErr
	defaultClientMu.RUnlock()
	return c, err
}

// SetDefaultClientConfig sets the configuration used by the
// global fasthttp-style client (used by Do/Get/Post helpers).
//
// It is safe to call this multiple times; the last successful
// call wins. For best results, call it during application
// startup before using the package-level helpers.
func SetDefaultClientConfig(cfg ClientConfig) error {
	c, err := NewClient(cfg)
	defaultClientMu.Lock()
	defaultClient = c
	defaultClientErr = err
	defaultClientMu.Unlock()

	// Mark the lazy initializer as done so getDefaultClient
	// doesn't overwrite the configured client.
	defaultClientOnce.Do(func() {})
	return err
}

// SetDefaultClient allows providing a fully constructed Client
// instance to be used by the global fasthttp-style helpers.
func SetDefaultClient(c *Client) {
	defaultClientMu.Lock()
	defaultClient = c
	defaultClientErr = nil
	defaultClientMu.Unlock()
	defaultClientOnce.Do(func() {})
}

func Do(req *Request, resp *Response) error {
	c, err := getDefaultClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	return DoWithClient(ctx, c, req, resp)
}

func DoTimeout(req *Request, resp *Response, timeout time.Duration) error {
	c, err := getDefaultClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return DoWithClient(ctx, c, req, resp)
}

func DoDeadline(req *Request, resp *Response, deadline time.Time) error {
	c, err := getDefaultClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	if !deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	return DoWithClient(ctx, c, req, resp)
}

func Get(url string, resp *Response) error {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetMethod(http.MethodGet)
	req.SetRequestURI(url)
	return Do(req, resp)
}

func GetTimeout(url string, resp *Response, timeout time.Duration) error {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetMethod(http.MethodGet)
	req.SetRequestURI(url)
	return DoTimeout(req, resp, timeout)
}

func GetDeadline(url string, resp *Response, deadline time.Time) error {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetMethod(http.MethodGet)
	req.SetRequestURI(url)
	return DoDeadline(req, resp, deadline)
}

func Post(url string, body []byte, resp *Response) error {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetMethod(http.MethodPost)
	req.SetRequestURI(url)
	req.SetBody(body)
	return Do(req, resp)
}

func PostTimeout(url string, body []byte, resp *Response, timeout time.Duration) error {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetMethod(http.MethodPost)
	req.SetRequestURI(url)
	req.SetBody(body)
	return DoTimeout(req, resp, timeout)
}

func PostDeadline(url string, body []byte, resp *Response, deadline time.Time) error {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetMethod(http.MethodPost)
	req.SetRequestURI(url)
	req.SetBody(body)
	return DoDeadline(req, resp, deadline)
}

func Delete(url string, resp *Response) error {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetMethod(http.MethodDelete)
	req.SetRequestURI(url)
	return Do(req, resp)
}

func DeleteTimeout(url string, resp *Response, timeout time.Duration) error {
	req := AcquireRequest()
	defer ReleaseRequest(req)
	req.SetMethod(http.MethodDelete)
	req.SetRequestURI(url)
	return DoTimeout(req, resp, timeout)
}
