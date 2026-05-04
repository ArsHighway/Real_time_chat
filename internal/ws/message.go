package ws

type InputMessage struct {
	Text string `json:"text"`
}

type Message struct {
	RoomID string
	Sender string
	Data   []byte
}
