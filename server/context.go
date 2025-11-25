package server

import (
	"net"
	"net/http"
	"net/url"
	"sync"
)

type RequestCtx struct {
	w      http.ResponseWriter
	r      *http.Request
	path   []byte
	meth   []byte
	params []routeParam
}

type RequestHandler func(ctx *RequestCtx)

var ctxPool = sync.Pool{
	New: func() any {
		return &RequestCtx{}
	},
}

func acquireCtx(w http.ResponseWriter, r *http.Request) *RequestCtx {
	ctx := ctxPool.Get().(*RequestCtx)
	ctx.w = w
	ctx.r = r
	ctx.resetParams()
	ctx.path = append(ctx.path[:0], r.URL.Path...)
	ctx.meth = append(ctx.meth[:0], r.Method...)
	return ctx
}

func releaseCtx(ctx *RequestCtx) {
	ctx.w = nil
	ctx.r = nil
	if cap(ctx.path) > 0 {
		ctx.path = ctx.path[:0]
	}
	if cap(ctx.meth) > 0 {
		ctx.meth = ctx.meth[:0]
	}
	if len(ctx.params) > 0 {
		ctx.params = ctx.params[:0]
	}
	ctxPool.Put(ctx)
}

func HandlerToHTTP(h RequestHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := acquireCtx(w, r)
		defer releaseCtx(ctx)
		h(ctx)
	})
}

func (ctx *RequestCtx) Path() []byte {
	return ctx.path
}

func (ctx *RequestCtx) Method() []byte {
	return ctx.meth
}

func (ctx *RequestCtx) QueryArgs() url.Values {
	return ctx.r.URL.Query()
}

func (ctx *RequestCtx) Host() string {
	return ctx.r.Host
}

func (ctx *RequestCtx) RemoteIP() net.IP {
	host, _, err := net.SplitHostPort(ctx.r.RemoteAddr)
	if err != nil {
		return nil
	}
	return net.ParseIP(host)
}

func (ctx *RequestCtx) SetStatusCode(code int) {
	ctx.w.WriteHeader(code)
}

func (ctx *RequestCtx) SetContentType(v string) {
	ctx.w.Header().Set("Content-Type", v)
}

func (ctx *RequestCtx) Header() http.Header {
	return ctx.w.Header()
}

func (ctx *RequestCtx) Request() *http.Request {
	return ctx.r
}

func (ctx *RequestCtx) Write(p []byte) (int, error) {
	return ctx.w.Write(p)
}

func (ctx *RequestCtx) WriteString(s string) (int, error) {
	return ctx.w.Write([]byte(s))
}

func (ctx *RequestCtx) resetParams() {
	if len(ctx.params) > 0 {
		ctx.params = ctx.params[:0]
	}
}

func (ctx *RequestCtx) addParam(key, value string) {
	ctx.params = append(ctx.params, routeParam{key: key, value: value})
}

func (ctx *RequestCtx) UserValue(key string) string {
	for i := range ctx.params {
		if ctx.params[i].key == key {
			return ctx.params[i].value
		}
	}
	return ""
}
