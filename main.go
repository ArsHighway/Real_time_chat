package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn        *websocket.Conn
	sendMessage chan Message
	clientID    int
	hub         *Hub
}

type Hub struct {
	clients map[*Client]bool
	rooms   map[string]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

type Message struct {
	UserID int
	Text   string
	Time   time.Time
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.sendMessage)
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.sendMessage <- Message{
					UserID: client.clientID,
					Text:   string(msg),
					Time:   time.Now(),
				}:
				default:
					close(client.sendMessage)
					delete(h.clients, client)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) WriteLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.sendMessage:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg.Text))
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadLoop(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		c.hub.broadcast <- msg
	}
}

func (c *Client) PingLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		}
	}
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	c := &Client{
		conn:        conn,
		hub:         hub,
		sendMessage: make(chan Message, 256),
	}
	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	return c
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	client := NewClient(conn, hub)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	hub.register <- client

	go client.ReadLoop(ctx)
	go client.WriteLoop(ctx)
}
func main() {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
	ctx := context.Background()
	go hub.Run(ctx)
	http.HandleFunc("/global", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})
	http.ListenAndServe(":8080", nil)
}
