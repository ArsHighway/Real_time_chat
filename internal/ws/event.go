package ws

type Event struct {
	Type string `json:"type"` // leave, join, message, history

	Room string `json:"room,omitempty"`
	Text string `json:"text,omitempty"`

	UserID   int    `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`

	// History заполняется только при type == "history" (последние сообщения комнаты).
	History []Event `json:"history,omitempty"`
}
