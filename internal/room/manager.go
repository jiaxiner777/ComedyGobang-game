package room

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"comedy-gobang/internal/game"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Room struct {
	ID          string             `json:"id"`
	Game        *game.Game         `json:"game"`
	Clients     map[string]*Client `json:"-"`
	CreatedAt   time.Time          `json:"created_at"`
	QuoteEngine *game.QuoteEngine  `json:"-"`
	mutex       sync.RWMutex       `json:"-"`
}

type Client struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	RoomID   string    `json:"room_id"`
	LastSeen time.Time `json:"last_seen"`
}

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	RoomID    string      `json:"room_id,omitempty"`
	PlayerID  string      `json:"player_id,omitempty"`
}

type Manager struct {
	Rooms   map[string]*Room   `json:"-"`
	Clients map[string]*Client `json:"-"`
	mutex   sync.RWMutex       `json:"-"`
	hub     *Hub               `json:"-"`
}

type Hub struct {
	clients    map[*Connection]bool
	broadcast  chan BroadcastMessage
	register   chan *Connection
	unregister chan *Connection
	roomMgr    *Manager
	mutex      sync.RWMutex
}

type Connection struct {
	hub    *Hub
	client *Client
	send   chan []byte
	wsConn *websocket.Conn
}

type BroadcastMessage struct {
	message []byte
	roomID  string
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				if client.client != nil {
					h.roomMgr.LeaveRoom(client.client.RoomID, client.client.ID)
				}
			}
			h.mutex.Unlock()

		case broadcastMsg := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				if client.client != nil && client.client.RoomID == broadcastMsg.roomID {
					select {
					case client.send <- broadcastMsg.message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func NewManager() *Manager {
	hub := &Hub{
		clients:    make(map[*Connection]bool),
		broadcast:  make(chan BroadcastMessage),
		register:   make(chan *Connection),
		unregister: make(chan *Connection),
	}

	mgr := &Manager{
		Rooms:   make(map[string]*Room),
		Clients: make(map[string]*Client),
		hub:     hub,
	}

	hub.roomMgr = mgr
	go hub.run()

	return mgr
}

func (m *Manager) CreateRoom() *Room {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	roomID := uuid.New().String()[:8]
	room := &Room{
		ID:          roomID,
		Game:        game.NewGame(roomID),
		Clients:     make(map[string]*Client),
		CreatedAt:   time.Now(),
		QuoteEngine: game.NewQuoteEngine(),
	}

	m.Rooms[roomID] = room
	log.Printf("created room: %s", roomID)

	return room
}

func (m *Manager) GetRoom(roomID string) *Room {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.Rooms[roomID]
}

func (m *Manager) JoinRoom(roomID, playerName string) (*Room, error) {
	room, _, err := m.JoinRoomWithClient(roomID, playerName)
	return room, err
}

func (m *Manager) JoinRoomWithClient(roomID, playerName string) (*Room, *Client, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	room, exists := m.Rooms[roomID]
	if !exists {
		return nil, nil, fmt.Errorf("room not found")
	}

	clientID := uuid.New().String()
	client := &Client{
		ID:       clientID,
		Name:     playerName,
		RoomID:   roomID,
		LastSeen: time.Now(),
	}

	room.Clients[clientID] = client
	m.Clients[clientID] = client

	if err := room.Game.AddPlayer(clientID, playerName); err != nil {
		delete(room.Clients, clientID)
		delete(m.Clients, clientID)
		return nil, nil, err
	}

	log.Printf("player %s (%s) joined room %s", playerName, clientID, roomID)
	m.broadcastRoomState(roomID)

	return room, client, nil
}

func (m *Manager) LeaveRoom(roomID, clientID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	room, exists := m.Rooms[roomID]
	if !exists {
		return
	}

	client, exists := room.Clients[clientID]
	if !exists {
		return
	}

	room.Game.RemovePlayer(clientID)

	delete(room.Clients, clientID)
	delete(m.Clients, clientID)

	log.Printf("player %s (%s) left room %s", client.Name, clientID, roomID)

	if len(room.Clients) == 0 {
		delete(m.Rooms, roomID)
		log.Printf("deleted empty room: %s", roomID)
		return
	}

	m.broadcastRoomState(roomID)
}

func (m *Manager) SetPlayerReady(roomID, clientID string, ready bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	room, exists := m.Rooms[roomID]
	if !exists {
		return fmt.Errorf("房间不存在")
	}

	if err := room.Game.SetPlayerReady(clientID, ready); err != nil {
		return err
	}

	m.broadcastGameState(roomID)

	if room.Game.State == game.StatePlaying {
		m.broadcastMessage(roomID, Message{
			Type: "game_start",
			Data: map[string]interface{}{
				"message": "游戏开始！",
				"game":    room.Game.GetGameState(),
			},
			Timestamp: time.Now().Unix(),
		})
	}

	return nil
}

func (m *Manager) MakeMove(roomID, clientID string, x, y int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	room, exists := m.Rooms[roomID]
	if !exists {
		return fmt.Errorf("房间不存在")
	}

	if err := room.Game.MakeMove(clientID, x, y); err != nil {
		return err
	}

	m.broadcastGameState(roomID)

	quote := room.QuoteEngine.GetContextualQuote(room.Game, "move", clientID)
	m.broadcastMessage(roomID, Message{
		Type:      "quote",
		Data:      quote,
		Timestamp: time.Now().Unix(),
		PlayerID:  clientID,
	})

	if room.Game.State == game.StateFinished {
		winner := room.Game.Players[room.Game.Winner]
		loser := room.Game.Players[1-room.Game.Winner]

		winQuote := room.QuoteEngine.GetContextualQuote(room.Game, "win", winner.ID)
		m.broadcastMessage(roomID, Message{
			Type:      "quote",
			Data:      winQuote,
			Timestamp: time.Now().Unix(),
			PlayerID:  winner.ID,
		})

		loseQuote := room.QuoteEngine.GetContextualQuote(room.Game, "lose", loser.ID)
		m.broadcastMessage(roomID, Message{
			Type:      "quote",
			Data:      loseQuote,
			Timestamp: time.Now().Unix(),
			PlayerID:  loser.ID,
		})

		m.broadcastMessage(roomID, Message{
			Type: "game_over",
			Data: map[string]interface{}{
				"winner": winner.Name,
				"game":   room.Game.GetGameState(),
			},
			Timestamp: time.Now().Unix(),
		})
	}

	return nil
}

func (m *Manager) UseSkill(roomID, clientID string, skillType game.SkillType, targetPositions []game.Position) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	room, exists := m.Rooms[roomID]
	if !exists {
		return fmt.Errorf("房间不存在")
	}

	if err := room.Game.UseSkill(clientID, skillType, targetPositions); err != nil {
		return err
	}

	m.broadcastGameState(roomID)

	quote := room.QuoteEngine.GetContextualQuote(room.Game, "skill", clientID)
	m.broadcastMessage(roomID, Message{
		Type:      "quote",
		Data:      quote,
		Timestamp: time.Now().Unix(),
		PlayerID:  clientID,
	})

	return nil
}

func (m *Manager) broadcastRoomState(roomID string) {
	room := m.Rooms[roomID]
	if room == nil {
		return
	}

	roomState := map[string]interface{}{
		"id":      room.ID,
		"game":    room.Game.GetGameState(),
		"clients": room.Clients,
		"full":    room.Game.IsFull(),
	}

	m.broadcastMessage(roomID, Message{
		Type:      "room_state",
		Data:      roomState,
		Timestamp: time.Now().Unix(),
	})
}

func (m *Manager) broadcastGameState(roomID string) {
	room := m.Rooms[roomID]
	if room == nil {
		return
	}

	m.broadcastMessage(roomID, Message{
		Type:      "game_state",
		Data:      room.Game.GetGameState(),
		Timestamp: time.Now().Unix(),
	})
}

func (m *Manager) broadcastMessage(roomID string, message Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("broadcast encode failed: %v", err)
		return
	}

	m.hub.broadcast <- BroadcastMessage{
		message: data,
		roomID:  roomID,
	}
}

func (m *Manager) GetRoomList() []map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var roomList []map[string]interface{}
	for _, room := range m.Rooms {
		roomInfo := map[string]interface{}{
			"id":        room.ID,
			"full":      room.Game.IsFull(),
			"state":     room.Game.State,
			"clients":   len(room.Clients),
			"createdAt": room.CreatedAt,
		}
		roomList = append(roomList, roomInfo)
	}

	return roomList
}

func (m *Manager) Cleanup() {
	now := time.Now()
	var expired []*Client

	m.mutex.RLock()
	for _, client := range m.Clients {
		if now.Sub(client.LastSeen) > 30*time.Minute {
			expired = append(expired, client)
		}
	}
	m.mutex.RUnlock()

	for _, client := range expired {
		m.LeaveRoom(client.RoomID, client.ID)
	}
}
