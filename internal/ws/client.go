package ws

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	send chan Event

	userID   int
	username string

	hub   *Hub
	rooms map[string]bool

	ctx    context.Context
	cancel context.CancelFunc
}

func NewClient(
	conn *websocket.Conn,
	hub *Hub,
	ctx context.Context,
	cancel context.CancelFunc,
) *Client {

	c := &Client{
		conn:   conn,
		hub:    hub,
		send:   make(chan Event, 256),
		ctx:    ctx,
		cancel: cancel,
		rooms:  make(map[string]bool),
	}

	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	return c
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {

		case <-c.ctx.Done():
			return

		case msg, ok := <-c.send:
			if !ok {
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			err := c.conn.WriteJSON(msg)
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.cancel()
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		var event Event

		err := c.conn.ReadJSON(&event)
		if err != nil {
			return
		}

		event.UserID = c.userID
		event.Username = c.username

		select {
		case <-c.ctx.Done():
			return
		case c.hub.incoming <- event:
		}
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {

		case <-c.ctx.Done():
			return

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
