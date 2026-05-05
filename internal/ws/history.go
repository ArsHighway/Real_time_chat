package ws

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	historyListMax   = 200
	historyFetchSize = 50
)

func roomHistoryKey(room string) string {
	return fmt.Sprintf("chat:room:%s:messages", room)
}

func (h *Hub) appendRoomMessage(ctx context.Context, room string, jsonLine []byte) error {
	key := roomHistoryKey(room)
	pipe := h.rdb.Pipeline()
	pipe.LPush(ctx, key, jsonLine)
	pipe.LTrim(ctx, key, 0, historyListMax-1)
	_, err := pipe.Exec(ctx)
	return err
}

func (h *Hub) recentRoomMessages(ctx context.Context, room string, n int) ([]Event, error) {
	if n <= 0 {
		n = historyFetchSize
	}
	if n > historyListMax {
		n = historyListMax
	}
	key := roomHistoryKey(room)
	raws, err := h.rdb.LRange(ctx, key, 0, int64(n-1)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(raws))
	for _, s := range raws {
		var ev Event
		if err := json.Unmarshal([]byte(s), &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	// В списке сначала новые (LPUSH); клиенту отдаём по времени от старых к новым.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (h *Hub) pushHistory(ctx context.Context, c *Client, room string) {
	msgs, err := h.recentRoomMessages(ctx, room, historyFetchSize)
	if err != nil || len(msgs) == 0 {
		return
	}
	ev := Event{Type: "history", Room: room, History: msgs}
	select {
	case c.send <- ev:
	default:
		h.unregister <- c
	}
}
