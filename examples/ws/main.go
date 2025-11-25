package main

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
	"v2fasthttp/server"
)

func main() {
	router := server.NewRouter()

	upgrader := server.NewWSUpgrader()

	router.GET("/ws", server.WebSocketHandler(upgrader, func(conn *websocket.Conn) {
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			out := append([]byte(time.Now().Format(time.RFC3339)+" "), msg...)
			if err := conn.WriteMessage(mt, out); err != nil {
				return
			}
		}
	}))

	s := server.NewFast(router.Handler, server.DefaultConfig())

	log.Println("websocket server on :8081/ws")
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
