package main

import (
	"context"
	"net/http"

	"github.com/ArsHighway/Real_time_chat/internal/middleware"
	"github.com/ArsHighway/Real_time_chat/internal/ws"
	"github.com/go-chi/chi/v5"
)

func main() {
	hub := ws.NewHub()
	ctx := context.Background()
	go hub.Run(ctx)
	r := chi.NewRouter()
	r.With(middleware.JWTMiddleware).Get("/global", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWS(hub, w, r)
	})
	http.ListenAndServe(":8080", r)
}
