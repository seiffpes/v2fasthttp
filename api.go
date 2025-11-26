package v2fasthttp

import (
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

func Do(req *Request, resp *Response) error {
	return fasthttp.Do(req, resp)
}

func DoTimeout(req *Request, resp *Response, timeout time.Duration) error {
	return fasthttp.DoTimeout(req, resp, timeout)
}

func Get(dst []byte, url string) (statusCode int, body []byte, err error) {
	return fasthttp.Get(dst, url)
}

func GetTimeout(dst []byte, url string, timeout time.Duration) (statusCode int, body []byte, err error) {
	return fasthttp.GetTimeout(dst, url, timeout)
}

func Post(dst []byte, url string, postArgs *fasthttp.Args) (statusCode int, body []byte, err error) {
	return fasthttp.Post(dst, url, postArgs)
}

func (c *Client) SetProxyHTTP(proxy string) {
	c.Client.Dial = fasthttpproxy.FasthttpHTTPDialer(proxy)
}

func (c *Client) SetSOCKS5Proxy(proxyAddr string) {
	c.Client.Dial = fasthttpproxy.FasthttpSocksDialer(proxyAddr)
}
