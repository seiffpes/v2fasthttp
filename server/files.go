package server

import "net/http"

func FileServer(fs http.FileSystem) RequestHandler {
	handler := http.FileServer(fs)
	return func(ctx *RequestCtx) {
		handler.ServeHTTP(ctx.w, ctx.r)
	}
}

func FileServerFromDir(root string) RequestHandler {
	return FileServer(http.Dir(root))
}
