package room

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"comedy-gobang/internal/game"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (m *Manager) HandleWebSocket(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		c.JSON(400, gin.H{"error": "WebSocket连接失败"})
		return
	}

	connection := &Connection{
		hub:    m.hub,
		wsConn: ws,
		send:   make(chan []byte, 256),
	}

	m.hub.register <- connection

	go connection.writePump()
	go connection.readPump(m)
}

func (c *Connection) readPump(mgr *Manager) {
	defer func() {
		c.hub.unregister <- c
		c.wsConn.Close()
	}()

	c.wsConn.SetReadLimit(512)

	for {
		_, messageBytes, err := c.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket read error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("message decode failed: %v", err)
			continue
		}

		c.handleMessage(mgr, &msg)
	}
}

func (c *Connection) writePump() {
	defer func() {
		c.wsConn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.wsConn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.wsConn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("websocket write failed: %v", err)
				return
			}
		}
	}
}

func (c *Connection) handleMessage(mgr *Manager, msg *Message) {
	if c.client != nil {
		c.client.LastSeen = time.Now()
	}

	switch msg.Type {
	case "join_room":
		c.handleJoinRoom(mgr, msg)
	case "leave_room":
		c.handleLeaveRoom(mgr, msg)
	case "set_ready":
		c.handleSetReady(mgr, msg)
	case "make_move":
		c.handleMakeMove(mgr, msg)
	case "use_skill":
		c.handleUseSkill(mgr, msg)
	case "ping":
		c.handlePing()
	default:
		log.Printf("unknown message type: %s", msg.Type)
	}
}

func (c *Connection) handleJoinRoom(mgr *Manager, msg *Message) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	roomID, ok := data["room_id"].(string)
	if !ok {
		return
	}

	playerName, ok := data["player_name"].(string)
	if !ok {
		playerName = "匿名玩家"
	}

	room, client, err := mgr.JoinRoomWithClient(roomID, playerName)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	foundClient := client
	for _, roomClient := range room.Clients {
		if roomClient.Name == playerName {
			foundClient = roomClient
			break
		}
	}

	if foundClient == nil {
		c.sendError("客户端信息未找到")
		return
	}

	c.client = client

	c.hub.mutex.Lock()
	for hubClient := range c.hub.clients {
		if hubClient == c {
			hubClient.client = c.client
			break
		}
	}
	c.hub.mutex.Unlock()

	c.sendClientID()
	c.sendRoomState(room)
}

func (c *Connection) handleLeaveRoom(mgr *Manager, msg *Message) {
	if c.client == nil {
		return
	}

	mgr.LeaveRoom(c.client.RoomID, c.client.ID)
	c.client = nil

	c.hub.mutex.Lock()
	for hubClient := range c.hub.clients {
		if hubClient == c {
			hubClient.client = nil
			break
		}
	}
	c.hub.mutex.Unlock()
}

func (c *Connection) handleSetReady(mgr *Manager, msg *Message) {
	if c.client == nil {
		c.sendError("未加入房间")
		return
	}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	ready, ok := data["ready"].(bool)
	if !ok {
		return
	}

	if err := mgr.SetPlayerReady(c.client.RoomID, c.client.ID, ready); err != nil {
		c.sendError(err.Error())
	}
}

func (c *Connection) handleMakeMove(mgr *Manager, msg *Message) {
	if c.client == nil {
		c.sendError("未加入房间")
		return
	}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	x, ok := data["x"].(float64)
	if !ok {
		return
	}

	y, ok := data["y"].(float64)
	if !ok {
		return
	}

	if err := mgr.MakeMove(c.client.RoomID, c.client.ID, int(x), int(y)); err != nil {
		c.sendError(err.Error())
	}
}

func (c *Connection) handleUseSkill(mgr *Manager, msg *Message) {
	if c.client == nil {
		c.sendError("未加入房间")
		return
	}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	skillTypeFloat, ok := data["skill_type"].(float64)
	if !ok {
		return
	}

	targetPositions, err := parseTargetPositions(data)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	if err := mgr.UseSkill(c.client.RoomID, c.client.ID, game.SkillType(int(skillTypeFloat)), targetPositions); err != nil {
		c.sendError(err.Error())
	}
}

func parseTargetPositions(data map[string]interface{}) ([]game.Position, error) {
	if targetList, ok := data["target_positions"].([]interface{}); ok {
		positions := make([]game.Position, 0, len(targetList))
		for _, item := range targetList {
			targetData, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("无效的目标位置")
			}

			pos, err := parsePosition(targetData)
			if err != nil {
				return nil, err
			}
			positions = append(positions, pos)
		}
		return positions, nil
	}

	if targetData, ok := data["target_pos"].(map[string]interface{}); ok {
		pos, err := parsePosition(targetData)
		if err != nil {
			return nil, err
		}
		return []game.Position{pos}, nil
	}

	return nil, nil
}

func parsePosition(targetData map[string]interface{}) (game.Position, error) {
	xFloat, ok := targetData["x"].(float64)
	if !ok {
		return game.Position{}, fmt.Errorf("无效的目标位置 x")
	}

	yFloat, ok := targetData["y"].(float64)
	if !ok {
		return game.Position{}, fmt.Errorf("无效的目标位置 y")
	}

	return game.Position{
		X: int(xFloat),
		Y: int(yFloat),
	}, nil
}

func (c *Connection) handlePing() {
	c.sendMessage(Message{
		Type: "pong",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	})
}

func (c *Connection) sendError(message string) {
	c.sendMessage(Message{
		Type: "error",
		Data: map[string]interface{}{
			"message": message,
		},
		Timestamp: time.Now().Unix(),
	})
}

func (c *Connection) sendRoomState(room *Room) {
	roomState := map[string]interface{}{
		"id":      room.ID,
		"game":    room.Game.GetGameState(),
		"clients": room.Clients,
		"full":    room.Game.IsFull(),
	}

	c.sendMessage(Message{
		Type:      "room_state",
		Data:      roomState,
		Timestamp: time.Now().Unix(),
	})
}

func (c *Connection) sendMessage(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("message encode failed: %v", err)
		return
	}

	select {
	case c.send <- data:
	default:
		close(c.send)
	}
}

func (c *Connection) sendClientID() {
	if c.client == nil {
		return
	}

	c.sendMessage(Message{
		Type: "client_id",
		Data: map[string]interface{}{
			"client_id": c.client.ID,
		},
		Timestamp: time.Now().Unix(),
	})
}
