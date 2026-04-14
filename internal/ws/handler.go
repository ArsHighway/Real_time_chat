package ws

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
)

type contextKey string

const userKey contextKey = "jwt_user"

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWS(hub Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	val := r.Context().Value(userKey)
	if val == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	JWTUser := val.(int)

	username := r.URL.Query().Get("username")
	if username == "" {
		username = "anonym"
	}

	client := NewClient(conn, &hub)
	client.username = username
	client.jwtuser = JWTUser

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	hub.register <- client

	go client.ReadLoop(ctx)
	go client.WriteLoop(ctx)
	go client.PingLoop(ctx)

	go func() {
		<-ctx.Done()
		conn.Close()
	}()
}
