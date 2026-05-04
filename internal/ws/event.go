package ws

type Event struct {
	Type string `json:"type"` // leave, join, message

	Room string `json:"room,omitempty"`
	Text string `json:"text,omitempty"`

	UserID   int    `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
}
