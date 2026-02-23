package room

import (
	"encoding/json"
	"errors"
	"sort"
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
	BuyIn() int64
	Send([]byte) error
	Close()
}

type Session struct {
	id         string
	table      *engine.Table
	clients    map[string]Client
	hostUserID string
	mu         sync.Mutex
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
	if prev, ok := s.clients[c.ID()]; ok && prev != c {
		prev.Close()
	}
	s.clients[c.ID()] = c
	if s.hostUserID == "" {
		s.hostUserID = c.ID()
	}
	s.table.AddOrReconnectPlayer(c.ID(), c.Name(), c.BuyIn())
	s.pushSnapshotLocked()
}

func (s *Session) Leave(c Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.clients[c.ID()]
	if !ok || current != c {
		return
	}
	delete(s.clients, c.ID())
	s.table.DisconnectPlayer(c.ID())
	if s.hostUserID == c.ID() {
		s.hostUserID = s.pickNewHostLocked()
	}
	s.pushSnapshotLocked()
}

func (s *Session) RemoveUser(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[userID]
	if ok {
		delete(s.clients, userID)
		c.Close()
	}
	s.table.RemovePlayer(userID)
	if s.hostUserID == userID {
		s.hostUserID = s.pickNewHostLocked()
	}
	s.pushSnapshotLocked()
}

func (s *Session) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients) == 0
}

func (s *Session) Handle(userID string, raw []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var in Inbound
	if err := json.Unmarshal(raw, &in); err != nil {
		return err
	}
	s.table.AddOrReconnectPlayer(userID, "", 0)

	switch in.Type {
	case "start_hand":
		if userID != s.hostUserID {
			return errors.New("only host can start hand")
		}
		var req struct {
			Mode string `json:"mode"`
		}
		_ = json.Unmarshal(in.Payload, &req)
		s.table.SetDeckMode(req.Mode)
		if err := s.table.StartHand(); err != nil {
			return err
		}
	case "restart_hand":
		if userID != s.hostUserID {
			return errors.New("only host can restart hand")
		}
		if err := s.table.RestartHand(); err != nil {
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
	case "join_table":
		var req struct {
			Seat int `json:"seat"`
		}
		_ = json.Unmarshal(in.Payload, &req)
		if err := s.table.RequestJoinTable(userID, req.Seat); err != nil {
			return err
		}
	case "set_seat":
		var req struct {
			Seat int `json:"seat"`
		}
		if err := json.Unmarshal(in.Payload, &req); err != nil {
			return err
		}
		if err := s.table.ChangeSeat(userID, req.Seat); err != nil {
			return err
		}
	case "reveal_cards":
		if err := s.table.RevealCards(userID); err != nil {
			return err
		}
	default:
		return errors.New("unknown message type")
	}

	s.pushSnapshotLocked()
	return nil
}

func (s *Session) Dissolve(byUser string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if byUser != s.hostUserID {
		return errors.New("only host can dissolve room")
	}
	out := Outbound{Type: "error", Payload: map[string]string{"message": "room dissolved by host"}}
	payload, _ := json.Marshal(out)
	for _, c := range s.clients {
		_ = c.Send(payload)
		c.Close()
	}
	s.clients = map[string]Client{}
	s.hostUserID = ""
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
		snapshot := s.table.SnapshotFor(userID, s.hostUserID)
		out := Outbound{Type: "snapshot", Payload: snapshot}
		payload, _ := json.Marshal(out)
		_ = c.Send(payload)
	}
}

func (s *Session) pickNewHostLocked() string {
	if len(s.clients) == 0 {
		return ""
	}
	ids := make([]string, 0, len(s.clients))
	for uid := range s.clients {
		ids = append(ids, uid)
	}
	sort.Strings(ids)
	return ids[0]
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

func (m *Manager) Leave(roomID string, c Client) {
	if roomID == "" {
		roomID = "main"
	}
	m.mu.Lock()
	s, ok := m.rooms[roomID]
	m.mu.Unlock()
	if !ok {
		return
	}
	s.Leave(c)
	if s.IsEmpty() {
		m.mu.Lock()
		if cur, ok := m.rooms[roomID]; ok && cur == s {
			delete(m.rooms, roomID)
		}
		m.mu.Unlock()
	}
}

func (m *Manager) Dissolve(roomID, byUser string) error {
	if roomID == "" {
		roomID = "main"
	}
	m.mu.Lock()
	s, ok := m.rooms[roomID]
	m.mu.Unlock()
	if !ok {
		return errors.New("room not found")
	}
	if err := s.Dissolve(byUser); err != nil {
		return err
	}
	m.mu.Lock()
	if cur, ok := m.rooms[roomID]; ok && cur == s {
		delete(m.rooms, roomID)
	}
	m.mu.Unlock()
	return nil
}

func (m *Manager) RemoveUser(roomID, userID string) {
	if roomID == "" {
		roomID = "main"
	}
	m.mu.Lock()
	s, ok := m.rooms[roomID]
	m.mu.Unlock()
	if !ok {
		return
	}
	s.RemoveUser(userID)
	if s.IsEmpty() {
		m.mu.Lock()
		if cur, ok := m.rooms[roomID]; ok && cur == s {
			delete(m.rooms, roomID)
		}
		m.mu.Unlock()
	}
}

func HandleWithAck(session *Session, userID string, raw []byte) {
	if err := session.Handle(userID, raw); err != nil {
		session.pushError(userID, err.Error())
	}
}

func ServerNow() string {
	return time.Now().Format(time.RFC3339)
}
