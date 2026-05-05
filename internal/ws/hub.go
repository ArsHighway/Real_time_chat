package ws

import (
	"context"
	"encoding/json"
	"os"

	"github.com/redis/go-redis/v9"
)

type Hub struct {
	rooms map[string]*Room
	users map[int]*Client

	register   chan *Client
	unregister chan *Client
	incoming   chan Event

	rdb *redis.Client
}

func NewHub() *Hub {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	return &Hub{
		rooms:      make(map[string]*Room),
		users:      make(map[int]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		incoming:   make(chan Event, 256),
		rdb: redis.NewClient(&redis.Options{
			Addr: redisAddr,
		}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	go h.Subscriber(ctx)

	for {
		select {

		case client := <-h.register:
			h.users[client.userID] = client

		case client := <-h.unregister:
			h.removeClient(client)

		case event := <-h.incoming:
			h.handleClientEvent(ctx, event)

		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) handleClientEvent(ctx context.Context, event Event) {
	if event.Room == "" {
		return
	}

	channel := "room:" + event.Room

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	if event.Type == "message" {
		_ = h.appendRoomMessage(ctx, event.Room, data)
	}

	_ = h.Publish(ctx, channel, string(data))
}

func (h *Hub) Publish(ctx context.Context, channel string, msg string) error {
	return h.rdb.Publish(ctx, channel, msg).Err()
}

func (h *Hub) Subscriber(ctx context.Context) {
	sub := h.rdb.PSubscribe(ctx, "room:*")
	defer sub.Close()

	ch := sub.Channel()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}

			switch event.Type {
			case "message":
				h.broadcast(event)
			case "join":
				if client := h.findClient(event.UserID); client != nil {
					h.joinRoom(event.Room, event.UserID, client)
					h.pushHistory(ctx, client, event.Room)
				}
			case "leave":
				h.leaveRoom(event.Room, event.UserID)
			}
		}
	}
}

func (h *Hub) createRoom(name string) {
	if _, ok := h.rooms[name]; ok {
		return
	}

	h.rooms[name] = &Room{
		id:      name,
		clients: make(map[int]*Client),
	}
}

func (h *Hub) joinRoom(roomID string, userID int, c *Client) {
	if roomID == "" {
		return
	}

	h.createRoom(roomID)

	room, ok := h.rooms[roomID]
	if !ok {
		return
	}

	room.clients[userID] = c
	c.rooms[roomID] = true
}

func (h *Hub) leaveRoom(roomID string, userID int) {
	if roomID == "" {
		return
	}

	room, ok := h.rooms[roomID]
	if !ok {
		return
	}

	delete(room.clients, userID)
	if len(room.clients) == 0 {
		delete(h.rooms, roomID)
	}
}

func (h *Hub) broadcast(event Event) {
	room, ok := h.rooms[event.Room]
	if !ok {
		return
	}

	for _, client := range room.clients {
		select {
		case client.send <- event:
		default:
			h.unregister <- client
		}
	}
}

func (h *Hub) removeClient(c *Client) {
	if _, ok := h.users[c.userID]; !ok {
		return
	}

	for roomID := range c.rooms {
		h.leaveRoom(roomID, c.userID)
	}
	delete(h.users, c.userID)
	close(c.send)
}

func (h *Hub) findClient(userID int) *Client {
	return h.users[userID]
}
