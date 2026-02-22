package room

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"texaspoker/server/internal/engine"
	"texaspoker/server/internal/model"
)

type Outbound struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Inbound struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Client interface {
	ID() string
	Name() string
	Send([]byte) error
}

type Session struct {
	id      string
	table   *engine.Table
	clients map[string]Client
	mu      sync.Mutex
}

func NewSession(roomID string) *Session {
	return &Session{
		id:      roomID,
		table:   engine.NewTable(roomID),
		clients: map[string]Client{},
	}
}

func (s *Session) Join(c Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID()] = c
	s.table.AddOrReconnectPlayer(c.ID(), c.Name())
	s.pushSnapshotLocked()
}

func (s *Session) Leave(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, userID)
	s.table.DisconnectPlayer(userID)
	s.pushSnapshotLocked()
}

func (s *Session) Handle(userID string, raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var in Inbound
	if err := json.Unmarshal(raw, &in); err != nil {
		return err
	}
	s.table.AddOrReconnectPlayer(userID, "")

	switch in.Type {
	case "start_hand":
		if err := s.table.StartHand(); err != nil {
			return err
		}
	case "action":
		var req struct {
			Action string `json:"action"`
			Amount int64  `json:"amount"`
		}
		if err := json.Unmarshal(in.Payload, &req); err != nil {
			return err
		}
		a := model.ActionInput{Type: model.ActionType(req.Action), Amount: req.Amount}
		if err := s.table.ApplyAction(userID, a); err != nil {
			return err
		}
	default:
		return errors.New("unknown message type")
	}

	s.pushSnapshotLocked()
	return nil
}

func (s *Session) pushError(userID, message string) {
	c, ok := s.clients[userID]
	if !ok {
		return
	}
	out := Outbound{Type: "error", Payload: map[string]string{"message": message}}
	payload, _ := json.Marshal(out)
	_ = c.Send(payload)
}

func (s *Session) pushSnapshotLocked() {
	for userID, c := range s.clients {
		snapshot := s.table.SnapshotFor(userID)
		out := Outbound{Type: "snapshot", Payload: snapshot}
		payload, _ := json.Marshal(out)
		_ = c.Send(payload)
	}
}

type Manager struct {
	mu    sync.Mutex
	rooms map[string]*Session
}

func NewManager() *Manager {
	return &Manager{rooms: map[string]*Session{}}
}

func (m *Manager) Get(roomID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	if roomID == "" {
		roomID = "main"
	}
	if s, ok := m.rooms[roomID]; ok {
		return s
	}
	s := NewSession(roomID)
	m.rooms[roomID] = s
	return s
}

func HandleWithAck(session *Session, userID string, raw []byte) {
	if err := session.Handle(userID, raw); err != nil {
		session.pushError(userID, err.Error())
	}
}

func ServerNow() string {
	return time.Now().Format(time.RFC3339)
}
