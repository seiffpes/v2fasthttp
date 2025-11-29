package v2fasthttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/valyala/fasthttp"
)

var (
	benchServerOnce sync.Once
	benchServerURL  string
)

func getBenchServerURL() string {
	benchServerOnce.Do(func() {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))
		benchServerURL = srv.URL
	})
	return benchServerURL
}

func BenchmarkV2FastHTTP_DefaultClient_Do(b *testing.B) {
	url := getBenchServerURL()

	var req Request
	var resp Response
	req.SetRequestURI(url)
	req.Header.SetMethod("GET")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Do(&req, &resp); err != nil {
			b.Fatal(err)
		}
		resp.ResetBody()
	}
}

func BenchmarkV2FastHTTP_HighPerfClient_Do(b *testing.B) {
	url := getBenchServerURL()

	c := NewHighPerfClient("")

	var req Request
	var resp Response
	req.SetRequestURI(url)
	req.Header.SetMethod("GET")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.Do(&req, &resp); err != nil {
			b.Fatal(err)
		}
		resp.ResetBody()
	}
}

func BenchmarkV2FastHTTP_Client_GetBytes(b *testing.B) {
	url := getBenchServerURL()

	c := &Client{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := c.GetBytes(url); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNetHTTP_Client_Do(b *testing.B) {
	url := getBenchServerURL()

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		b.Fatalf("NewRequest: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkFastHTTP_Client_Do(b *testing.B) {
	url := getBenchServerURL()

	var req fasthttp.Request
	var resp fasthttp.Response
	client := &fasthttp.Client{}

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := client.Do(&req, &resp); err != nil {
			b.Fatal(err)
		}
		resp.ResetBody()
	}
}
