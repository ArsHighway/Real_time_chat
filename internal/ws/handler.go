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

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	val := r.Context().Value(userKey)
	if val == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID := val.(int)
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "anonym"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(r.Context())

	client := NewClient(conn, hub, ctx, cancel)
	client.userID = userID
	client.username = username

	hub.register <- client

	go client.readPump()
	go client.writePump()
	go client.pingLoop()
}
