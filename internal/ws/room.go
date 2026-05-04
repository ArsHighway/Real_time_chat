package ws

type Room struct {
	id      string
	clients map[int]*Client
}