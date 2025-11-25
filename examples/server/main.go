package main

import (
	"io"
	"log"

	"v2fasthttp/server"
)

func main() {
	router := server.NewRouter()

	router.GET("/", func(ctx *server.RequestCtx) {
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.WriteString("hello from v2fasthttp (fasthttp-style API)\n")
	})

	router.GET("/hello", func(ctx *server.RequestCtx) {
		name := ctx.QueryArgs().Get("name")
		if name == "" {
			name = "world"
		}
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.WriteString("hello, " + name)
	})

	router.GET("/user/:id", func(ctx *server.RequestCtx) {
		id := ctx.UserValue("id")
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.WriteString("user id = " + id)
	})

	router.POST("/echo", func(ctx *server.RequestCtx) {
		ctx.SetContentType("text/plain; charset=utf-8")
		body, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			ctx.SetStatusCode(500)
			ctx.WriteString("failed to read body")
			return
		}
		ctx.Write(body)
	})

	router.DELETE("/resource/:id", func(ctx *server.RequestCtx) {
		id := ctx.UserValue("id")
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.WriteString("deleted resource " + id)
	})

	router.GET("/static/*filepath", server.FileServerFromDir("./public"))

	s := server.NewFast(router.Handler, server.DefaultConfig())

	log.Println("listening on :8080")
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
