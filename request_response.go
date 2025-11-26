package v2fasthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
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

func (r *Request) SetHeader(key, value string) {
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set(key, value)
}

func (r *Request) AddHeader(key, value string) {
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Add(key, value)
}

func (r *Request) DelHeader(key string) {
	if r.Header == nil {
		return
	}
	r.Header.Del(key)
}

func (r *Request) HeaderValue(key string) string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get(key)
}

func (r *Request) SetQueryParam(key, value string) {
	u, err := url.Parse(r.URI)
	if err != nil {
		return
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	r.URI = u.String()
}

func (r *Request) AddQueryParam(key, value string) {
	u, err := url.Parse(r.URI)
	if err != nil {
		return
	}
	q := u.Query()
	q.Add(key, value)
	u.RawQuery = q.Encode()
	r.URI = u.String()
}

func (r *Request) DelQueryParam(key string) {
	u, err := url.Parse(r.URI)
	if err != nil {
		return
	}
	q := u.Query()
	q.Del(key)
	u.RawQuery = q.Encode()
	r.URI = u.String()
}

func (r *Request) QueryParam(key string) string {
	u, err := url.Parse(r.URI)
	if err != nil {
		return ""
	}
	return u.Query().Get(key)
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

func (r *Response) SetBodyString(body string) {
	r.SetBody([]byte(body))
}

func (r *Response) AppendBody(body []byte) {
	if len(body) == 0 {
		return
	}
	r.Body = append(r.Body, body...)
}

func (r *Response) SetHeader(key, value string) {
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Set(key, value)
}

func (r *Response) AddHeader(key, value string) {
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	r.Header.Add(key, value)
}

func (r *Response) DelHeader(key string) {
	if r.Header == nil {
		return
	}
	r.Header.Del(key)
}

func (r *Response) HeaderValue(key string) string {
	if r.Header == nil {
		return ""
	}
	return r.Header.Get(key)
}

func (r *Response) BodyBytes() []byte {
	if len(r.Body) == 0 {
		return nil
	}
	out := make([]byte, len(r.Body))
	copy(out, r.Body)
	return out
}

func (r *Response) BodyString() string {
	return string(r.Body)
}

func (r *Response) StatusCodeValue() int {
	return r.StatusCode
}

func (r *Response) SetContentType(ct string) {
	r.SetHeader("Content-Type", ct)
}

func (r *Response) ContentType() string {
	return r.HeaderValue("Content-Type")
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

func SetDefaultClientConfig(cfg ClientConfig) error {
	c, err := NewClient(cfg)
	defaultClientMu.Lock()
	defaultClient = c
	defaultClientErr = err
	defaultClientMu.Unlock()

	defaultClientOnce.Do(func() {})
	return err
}

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

func GetBytesURL(targetURL string) ([]byte, int, error) {
	c, err := getDefaultClient()
	if err != nil {
		return nil, 0, err
	}
	return c.GetBytes(context.Background(), targetURL)
}

func GetBytesTimeout(targetURL string, timeout time.Duration) ([]byte, int, error) {
	c, err := getDefaultClient()
	if err != nil {
		return nil, 0, err
	}
	return c.GetBytesTimeout(targetURL, timeout)
}

func GetBytesDeadline(targetURL string, deadline time.Time) ([]byte, int, error) {
	c, err := getDefaultClient()
	if err != nil {
		return nil, 0, err
	}
	return c.GetBytesDeadline(targetURL, deadline)
}

func GetStringURL(targetURL string) (string, int, error) {
	body, status, err := GetBytesURL(targetURL)
	return string(body), status, err
}

func GetStringTimeout(targetURL string, timeout time.Duration) (string, int, error) {
	body, status, err := GetBytesTimeout(targetURL, timeout)
	return string(body), status, err
}

func PostBytesURL(targetURL string, body []byte) ([]byte, int, error) {
	resp := AcquireResponse()
	defer ReleaseResponse(resp)
	if err := Post(targetURL, body, resp); err != nil {
		return nil, 0, err
	}
	out := make([]byte, len(resp.Body))
	copy(out, resp.Body)
	return out, resp.StatusCode, nil
}

func PostBytesTimeout(targetURL string, body []byte, timeout time.Duration) ([]byte, int, error) {
	resp := AcquireResponse()
	defer ReleaseResponse(resp)
	if err := PostTimeout(targetURL, body, resp, timeout); err != nil {
		return nil, 0, err
	}
	out := make([]byte, len(resp.Body))
	copy(out, resp.Body)
	return out, resp.StatusCode, nil
}

func PostStringURL(targetURL string, body []byte) (string, int, error) {
	b, status, err := PostBytesURL(targetURL, body)
	return string(b), status, err
}

func PostStringTimeout(targetURL string, body []byte, timeout time.Duration) (string, int, error) {
	b, status, err := PostBytesTimeout(targetURL, body, timeout)
	return string(b), status, err
}

func PostJSON(url string, v any, resp *Response) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return Post(url, data, resp)
}

func PostJSONBytesURL(targetURL string, v any) ([]byte, int, error) {
	resp := AcquireResponse()
	defer ReleaseResponse(resp)
	if err := PostJSON(targetURL, v, resp); err != nil {
		return nil, 0, err
	}
	out := make([]byte, len(resp.Body))
	copy(out, resp.Body)
	return out, resp.StatusCode, nil
}
