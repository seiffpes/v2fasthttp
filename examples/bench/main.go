package main

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seiffpes/v2fasthttp/client"
	"github.com/seiffpes/v2fasthttp/server"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	router := server.NewRouter()
	router.GET("/", func(ctx *server.RequestCtx) {
		ctx.WriteString("OK")
	})

	srvCfg := server.DefaultConfig()
	srvCfg.Addr = ":8090"
	srv := server.NewFast(router.Handler, srvCfg)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(300 * time.Millisecond)

	clCfg := client.DefaultConfig()
	c, err := client.New(clCfg)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	const total = 100000
	const concurrency = 200

	var done int64
	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(concurrency)

	ctx := context.Background()

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			for {
				n := atomic.AddInt64(&done, 1)
				if n > total {
					return
				}
				_, _, err := c.GetBytes(ctx, "http://localhost:8090/")
				if err != nil {
					continue
				}
			}
		}()
	}

	wg.Wait()
	dur := time.Since(start)
	qps := float64(total) / dur.Seconds()

	log.Printf("completed %d requests in %s (%.0f req/s)", total, dur, qps)
}
