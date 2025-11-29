package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	v2 "github.com/seiffpes/v2fasthttp"
)

func main() {
	url := flag.String("url", "https://httpbin.org/get", "target URL")
	total := flag.Int64("total", 200000, "total number of requests")
	concurrency := flag.Int("concurrency", 200, "number of concurrent workers")
	proxy := flag.String("proxy", "", "optional HTTP proxy, e.g. user:pass@127.0.0.1:8080")
	version := flag.String("version", "1", "HTTP version to use: 1, 2, or 3")
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	pool := v2.NewClientPool(*concurrency, func() *v2.Client {
		switch *version {
		case "2":
			return v2.NewClientWithOptions(v2.ClientOptions{
				HTTPVersion:                   v2.HTTP2,
				MaxConnsPerHost:               100000,
				MaxIdleConnDuration:           100 * time.Millisecond,
				ReadBufferSize:                64 * 1024,
				WriteBufferSize:               64 * 1024,
				NoDefaultUserAgentHeader:      true,
				DisableHeaderNamesNormalizing: true,
				DisablePathNormalizing:        true,
				ProxyHTTP:                     *proxy,
			})
		case "3":
			return v2.NewClientWithOptions(v2.ClientOptions{
				HTTPVersion:                   v2.HTTP3,
				MaxConnsPerHost:               100000,
				MaxIdleConnDuration:           100 * time.Millisecond,
				ReadBufferSize:                64 * 1024,
				WriteBufferSize:               64 * 1024,
				NoDefaultUserAgentHeader:      true,
				DisableHeaderNamesNormalizing: true,
				DisablePathNormalizing:        true,
				ProxyHTTP:                     *proxy,
			})
		default:
			return v2.NewHighPerfClient(*proxy)
		}
	})

	var done int64
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(*concurrency)

	for i := 0; i < *concurrency; i++ {
		go func() {
			defer wg.Done()

			var req v2.Request
			var resp v2.Response

			req.SetRequestURI(*url)
			req.Header.SetMethod("GET")

			for {
				n := atomic.AddInt64(&done, 1)
				if n > *total {
					return
				}
				if err := pool.Do(&req, &resp); err != nil {
					continue
				}
				_ = resp.Body()
			}
		}()
	}

	wg.Wait()
	dur := time.Since(start)
	qps := float64(*total) / dur.Seconds()

	fmt.Printf("completed %d requests in %s (≈%.0f req/s)\n", *total, dur, qps)
	if *proxy != "" {
		log.Printf("used proxy: %s\n", *proxy)
	}
}
