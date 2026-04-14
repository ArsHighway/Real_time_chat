package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn        *websocket.Conn
	sendMessage chan Message
	clientID    int
	username    string
	hub         *Hub
	jwtuser     int
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	c := &Client{
		conn:        conn,
		hub:         hub,
		sendMessage: make(chan Message, 256),
	}
	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	return c
}

func (c *Client) WriteLoop(ctx context.Context) {
	defer func() {
		//log
		c.conn.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.sendMessage:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			data, err := json.Marshal(msg)
			if err != nil {
				return
			}
			err = c.conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadLoop(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		c.conn.Close()
	}()
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		var input InputMessage
		err = json.Unmarshal(data, &input)
		if err != nil {
			continue
		}
		msg := Message{
			UserID: c.clientID,
			Time:   time.Now(),
			Text:   c.username + ": " + input.Text,
		}
		c.hub.broadcast <- msg
	}
}

func (c *Client) PingLoop(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
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
