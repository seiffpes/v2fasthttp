package v2fasthttp

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
)

func TestNewClientWithOptionsDefaults(t *testing.T) {
	c := NewClientWithOptions(ClientOptions{})

	if c.httpVersion != HTTP1 {
		t.Fatalf("expected default httpVersion HTTP1, got %v", c.httpVersion)
	}
	if c.httpClient != nil {
		t.Fatalf("expected no net/http client for HTTP1 by default")
	}
	if c.MaxConnsPerHost != 1024 {
		t.Fatalf("expected MaxConnsPerHost default 1024, got %d", c.MaxConnsPerHost)
	}
	if c.MaxIdleConnDuration != 90*time.Second {
		t.Fatalf("unexpected MaxIdleConnDuration default: %s", c.MaxIdleConnDuration)
	}
}

func TestNewClientWithOptionsHTTP2(t *testing.T) {
	c := NewClientWithOptions(ClientOptions{
		HTTPVersion:     HTTP2,
		MaxConnsPerHost: 10,
	})

	if c.httpVersion != HTTP2 {
		t.Fatalf("expected httpVersion HTTP2, got %v", c.httpVersion)
	}
	if c.httpClient == nil {
		t.Fatalf("expected httpClient for HTTP2")
	}
	tr := trFromHTTPClient(c.httpClient)
	if tr == nil {
		t.Fatalf("expected *http.Transport for HTTP2 client")
	}
	if tr.MaxConnsPerHost != 10 {
		t.Fatalf("expected MaxConnsPerHost propagated to transport, got %d", tr.MaxConnsPerHost)
	}
}

func TestNewClientWithOptionsHTTP3(t *testing.T) {
	c := NewClientWithOptions(ClientOptions{
		HTTPVersion: HTTP3,
	})

	if c.httpVersion != HTTP3 {
		t.Fatalf("expected httpVersion HTTP3, got %v", c.httpVersion)
	}
	if c.httpClient == nil {
		t.Fatalf("expected httpClient for HTTP3")
	}
	rt, ok := c.httpClient.Transport.(*http3.Transport)
	if !ok || rt == nil {
		t.Fatalf("expected http3.Transport for HTTP3 client, got %T", c.httpClient.Transport)
	}
}

func TestNewClientWithOptionsHTTP3WithProxyFallsBackToHTTP2(t *testing.T) {
	c := NewClientWithOptions(ClientOptions{
		HTTPVersion: HTTP3,
		ProxyHTTP:   "127.0.0.1:8080",
	})

	if c.httpVersion != HTTP2 {
		t.Fatalf("expected fallback to HTTP2 when HTTP3 + proxy, got %v", c.httpVersion)
	}
	tr := trFromHTTPClient(c.httpClient)
	if tr == nil {
		t.Fatalf("expected *http.Transport after fallback")
	}

	// Proxy should be configured.
	if tr.Proxy == nil {
		t.Fatalf("expected Proxy function to be set on transport")
	}
	req := &http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}}
	u, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("unexpected error from Proxy: %v", err)
	}
	if u == nil || u.Host != "127.0.0.1:8080" {
		t.Fatalf("unexpected proxy URL: %#v", u)
	}
}

func TestSetProxyHTTPConfiguresTransportProxy(t *testing.T) {
	c := NewClientWithOptions(ClientOptions{
		HTTPVersion: HTTP2,
	})

	c.SetProxyHTTP("127.0.0.1:8080")

	tr := trFromHTTPClient(c.httpClient)
	if tr == nil {
		t.Fatalf("expected *http.Transport")
	}
	if tr.Proxy == nil {
		t.Fatalf("expected Proxy to be set")
	}
	u, err := tr.Proxy(&http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}})
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}
	if u == nil || u.Host != "127.0.0.1:8080" {
		t.Fatalf("unexpected proxy URL: %#v", u)
	}
}

func TestSetSOCKS5ProxyConfiguresDialContext(t *testing.T) {
	c := NewClientWithOptions(ClientOptions{
		HTTPVersion: HTTP2,
	})

	tr := trFromHTTPClient(c.httpClient)
	if tr == nil {
		t.Fatalf("expected *http.Transport")
	}

	if tr.DialContext != nil {
		t.Fatalf("expected DialContext to be nil before SetSOCKS5Proxy")
	}

	c.SetSOCKS5Proxy("socks5://127.0.0.1:9050")

	tr = trFromHTTPClient(c.httpClient)
	if tr.DialContext == nil {
		t.Fatalf("expected DialContext to be set after SetSOCKS5Proxy")
	}
	if tr.Proxy != nil {
		t.Fatalf("expected Proxy to be nil for SOCKS5 transport")
	}
}

func TestNewProxyClientPoolFromString(t *testing.T) {
	pool := NewProxyClientPoolFromString("127.0.0.1:8080\nsocks5://127.0.0.1:9050", 2)
	if pool == nil {
		t.Fatalf("expected non-nil pool")
	}
	if pool.Next() == nil {
		t.Fatalf("expected Next() to return a client")
	}

	if p := NewProxyClientPoolFromString("", 2); p != nil {
		t.Fatalf("expected nil pool for empty list")
	}
}

