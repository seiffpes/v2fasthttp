package v2fasthttp

import (
	"encoding/json"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

type (
	Client       struct{ fasthttp.Client }
	Request      = fasthttp.Request
	Response     = fasthttp.Response
	RequestCtx   = fasthttp.RequestCtx
	RequestHandler = fasthttp.RequestHandler
)

var defaultClient = &Client{}

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

func (c *Client) SetProxyHTTP(proxy string) {
	c.Client.Dial = fasthttpproxy.FasthttpHTTPDialer(proxy)
}

func (c *Client) SetSOCKS5Proxy(proxyAddr string) {
	c.Client.Dial = fasthttpproxy.FasthttpSocksDialer(proxyAddr)
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
