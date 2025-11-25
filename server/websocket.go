package server

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type WSUpgrader struct {
	Upgrader websocket.Upgrader
}

func NewWSUpgrader() *WSUpgrader {
	return &WSUpgrader{
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (u *WSUpgrader) Upgrade(ctx *RequestCtx) (*websocket.Conn, error) {
	return u.Upgrader.Upgrade(ctx.w, ctx.r, nil)
}

func WebSocketHandler(u *WSUpgrader, fn func(*websocket.Conn)) RequestHandler {
	return func(ctx *RequestCtx) {
		conn, err := u.Upgrade(ctx)
		if err != nil {
			ctx.SetStatusCode(http.StatusBadRequest)
			return
		}
		defer conn.Close()
		fn(conn)
	}
}
