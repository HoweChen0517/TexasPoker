package network

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"texaspoker/server/internal/room"
)

type SSEClient struct {
	userID string
	name   string
	buyIn  int64
	ch     chan []byte
	once   sync.Once
}

func (c *SSEClient) ID() string   { return c.userID }
func (c *SSEClient) Name() string { return c.name }
func (c *SSEClient) BuyIn() int64 { return c.buyIn }
func (c *SSEClient) Send(msg []byte) error {
	select {
	case c.ch <- msg:
	default:
	}
	return nil
}
func (c *SSEClient) Close() {
	c.once.Do(func() { close(c.ch) })
}

func NewEventsHandler(m *room.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("room")
		userID := r.URL.Query().Get("user")
		name := r.URL.Query().Get("name")
		buyInRaw := r.URL.Query().Get("buy_in")
		if userID == "" {
			http.Error(w, "missing user", http.StatusBadRequest)
			return
		}
		if name == "" {
			name = userID
		}
		buyIn, _ := strconv.ParseInt(buyInRaw, 10, 64)
		if buyIn <= 0 {
			buyIn = 2000
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "stream unsupported", http.StatusInternalServerError)
			return
		}

		client := &SSEClient{userID: userID, name: name, buyIn: buyIn, ch: make(chan []byte, 128)}
		session := m.Get(roomID)
		session.Join(client)
		defer func() {
			session.Leave(client.userID)
			client.Close()
		}()

		writer := bufio.NewWriter(w)
		fmt.Fprintf(writer, "event: ready\ndata: {\"ok\":true}\n\n")
		_ = writer.Flush()
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-client.ch:
				if !ok {
					return
				}
				fmt.Fprintf(writer, "event: message\ndata: %s\n\n", string(msg))
				if err := writer.Flush(); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

func NewActionHandler(m *room.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		roomID := r.URL.Query().Get("room")
		userID := r.URL.Query().Get("user")
		if userID == "" {
			http.Error(w, "missing user", http.StatusBadRequest)
			return
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		msg, _ := json.Marshal(payload)
		session := m.Get(roomID)
		room.HandleWithAck(session, userID, msg)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}
}
