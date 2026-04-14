package ws

import "time"

type MessageType string

const (
	TypeMessage MessageType = "message"
	TypeJoin    MessageType = "join"
	TypeLeave   MessageType = "leave"
)

type InputMessage struct {
	Text string `json:"text"`
}

type Message struct {
	Type   MessageType `json:"type"`
	UserID int         `json:"user_id"`
	Text   string      `json:"text"`
	Time   time.Time   `json:"time"`
}
