package ws

import (
	"context"
	"time"
)

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Message, 256),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case c := <-h.register:
			if h.clients == nil {
				h.clients = make(map[*Client]bool)
			}
			h.clients[c] = true
			h.broadcast <- Message{
				Type:   TypeJoin,
				UserID: c.clientID,
				Text:   c.username + " joined",
				Time:   time.Now(),
			}

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.sendMessage)
				select {
				case h.broadcast <- Message{
					Type:   TypeLeave,
					UserID: c.clientID,
					Text:   c.username + " leave",
					Time:   time.Now(),
				}:
				default:
				}
			}

		case msg := <-h.broadcast:
			msg.Type = TypeMessage
			for client := range h.clients {
				select {
				case client.sendMessage <- msg:
				default:
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
