package server

import (
	"net/http"
	"strings"
)

type Router struct {
	routes   map[string][]route
	NotFound RequestHandler
}

type route struct {
	path        string
	handler     RequestHandler
	segments    []routeSegment
	hasWildcard bool
}

type routeSegment struct {
	raw      string
	param    bool
	wildcard bool
	name     string
}

type routeParam struct {
	key   string
	value string
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[string][]route),
		NotFound: func(ctx *RequestCtx) {
			ctx.SetStatusCode(http.StatusNotFound)
			_, _ = ctx.WriteString("404 page not found")
		},
	}
}

func (r *Router) Handle(method, path string, h RequestHandler) {
	segments, hasWildcard := parseRoutePattern(path)
	r.routes[method] = append(r.routes[method], route{
		path:        path,
		handler:     h,
		segments:    segments,
		hasWildcard: hasWildcard,
	})
}

func (r *Router) GET(path string, h RequestHandler) {
	r.Handle(http.MethodGet, path, h)
}

func (r *Router) POST(path string, h RequestHandler) {
	r.Handle(http.MethodPost, path, h)
}

func (r *Router) PUT(path string, h RequestHandler) {
	r.Handle(http.MethodPut, path, h)
}

func (r *Router) DELETE(path string, h RequestHandler) {
	r.Handle(http.MethodDelete, path, h)
}

func (r *Router) PATCH(path string, h RequestHandler) {
	r.Handle(http.MethodPatch, path, h)
}

func (r *Router) HEAD(path string, h RequestHandler) {
	r.Handle(http.MethodHead, path, h)
}

func (r *Router) OPTIONS(path string, h RequestHandler) {
	r.Handle(http.MethodOptions, path, h)
}

func (r *Router) Handler(ctx *RequestCtx) {
	method := string(ctx.Method())
	path := string(ctx.Path())

	routes := r.routes[method]
	for i := range routes {
		if matchRoute(&routes[i], path, ctx) {
			routes[i].handler(ctx)
			return
		}
	}

	if r.NotFound != nil {
		r.NotFound(ctx)
		return
	}

	ctx.SetStatusCode(http.StatusNotFound)
	_, _ = ctx.WriteString("404 page not found")
}

func parseRoutePattern(pattern string) ([]routeSegment, bool) {
	if pattern == "" {
		pattern = "/"
	}
	if pattern == "/" {
		return nil, false
	}

	trimmed := strings.Trim(pattern, "/")
	parts := strings.Split(trimmed, "/")
	segments := make([]routeSegment, 0, len(parts))
	hasWildcard := false

	for _, p := range parts {
		if p == "" {
			continue
		}
		seg := routeSegment{raw: p}
		if strings.HasPrefix(p, ":") && len(p) > 1 {
			seg.param = true
			seg.name = p[1:]
		} else if strings.HasPrefix(p, "*") && len(p) > 1 {
			seg.wildcard = true
			seg.name = p[1:]
			hasWildcard = true
		}
		segments = append(segments, seg)
		if seg.wildcard {
			break
		}
	}

	return segments, hasWildcard
}

func splitPath(path string) []string {
	if path == "" || path == "/" {
		return nil
	}
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func matchRoute(rt *route, path string, ctx *RequestCtx) bool {
	if len(rt.segments) == 0 {
		return rt.path == path
	}

	pathSegs := splitPath(path)
	patternSegs := rt.segments

	if !rt.hasWildcard && len(pathSegs) != len(patternSegs) {
		return false
	}
	if rt.hasWildcard && len(pathSegs) < len(patternSegs)-1 {
		return false
	}

	ctx.resetParams()

	i := 0
	for pi := 0; pi < len(patternSegs); pi++ {
		seg := patternSegs[pi]
		if seg.wildcard {
			if i >= len(pathSegs) {
				ctx.addParam(seg.name, "")
			} else {
				ctx.addParam(seg.name, strings.Join(pathSegs[i:], "/"))
			}
			return true
		}

		if i >= len(pathSegs) {
			return false
		}
		part := pathSegs[i]
		i++

		if seg.param {
			ctx.addParam(seg.name, part)
			continue
		}

		if seg.raw != part {
			return false
		}
	}

	return i == len(pathSegs)
}
