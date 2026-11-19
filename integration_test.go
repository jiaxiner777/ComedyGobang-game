package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"comedy-gobang/internal/game"
	roompkg "comedy-gobang/internal/room"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type wsEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type playerSnapshot struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color int    `json:"color"`
	Mana  int    `json:"mana"`
	Ready bool   `json:"ready"`
}

type positionSnapshot struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type gameSnapshot struct {
	ID            string           `json:"id"`
	Board         [][]int          `json:"board"`
	Players       []playerSnapshot `json:"players"`
	CurrentPlayer int              `json:"currentPlayer"`
	State         int              `json:"state"`
	Winner        int              `json:"winner"`
	TurnCount     int              `json:"turnCount"`
	LastMove      positionSnapshot `json:"lastMove"`
	SkippedPlayer int              `json:"skippedPlayer"`
}

type roomSnapshot struct {
	ID   string       `json:"id"`
	Game gameSnapshot `json:"game"`
	Full bool         `json:"full"`
}

type gameStartSnapshot struct {
	Message string       `json:"message"`
	Game    gameSnapshot `json:"game"`
}

type gameOverSnapshot struct {
	Winner string       `json:"winner"`
	Game   gameSnapshot `json:"game"`
}

type clientIDSnapshot struct {
	ClientID string `json:"client_id"`
}

type errorSnapshot struct {
	Message string `json:"message"`
}

type wsState struct {
	clientID string
	room     *roomSnapshot
	game     *gameSnapshot
	gameOver *gameOverSnapshot
	errors   []string
}

type wsClient struct {
	conn  *websocket.Conn
	inbox chan wsEnvelope
	errs  chan error
	state *wsState
}

func newTestServer(t *testing.T) (*httptest.Server, *roompkg.Manager) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	manager := roompkg.NewManager()
	router := gin.New()
	router.GET("/ws", manager.HandleWebSocket)
	router.POST("/api/rooms", func(c *gin.Context) {
		c.JSON(200, gin.H{"room": manager.CreateRoom()})
	})

	server := httptest.NewServer(router)
	return server, manager
}

func websocketURL(serverURL string) string {
	u, _ := url.Parse(serverURL)
	u.Scheme = strings.Replace(u.Scheme, "http", "ws", 1)
	u.Path = "/ws"
	return u.String()
}

func dialWS(t *testing.T, serverURL string) *wsClient {
	t.Helper()

	conn, _, err := websocket.DefaultDialer.Dial(websocketURL(serverURL), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	client := &wsClient{
		conn:  conn,
		inbox: make(chan wsEnvelope, 64),
		errs:  make(chan error, 1),
		state: &wsState{},
	}

	go func() {
		defer close(client.inbox)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				client.errs <- err
				return
			}

			var envelope wsEnvelope
			if err := json.Unmarshal(raw, &envelope); err != nil {
				client.errs <- err
				return
			}

			client.inbox <- envelope
		}
	}()

	return client
}

func sendWSMessage(t *testing.T, client *wsClient, msgType string, data any) {
	t.Helper()

	payload := map[string]any{
		"type":      msgType,
		"data":      data,
		"timestamp": time.Now().UnixMilli(),
	}
	if err := client.conn.WriteJSON(payload); err != nil {
		t.Fatalf("send %s: %v", msgType, err)
	}
}

func applyEnvelope(t *testing.T, state *wsState, envelope wsEnvelope) {
	t.Helper()

	switch envelope.Type {
	case "client_id":
		var payload clientIDSnapshot
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			t.Fatalf("decode client_id: %v", err)
		}
		state.clientID = payload.ClientID
	case "room_state":
		var payload roomSnapshot
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			t.Fatalf("decode room_state: %v", err)
		}
		state.room = &payload
		state.game = &payload.Game
	case "game_state":
		var payload gameSnapshot
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			t.Fatalf("decode game_state: %v", err)
		}
		state.game = &payload
	case "game_start":
		var payload gameStartSnapshot
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			t.Fatalf("decode game_start: %v", err)
		}
		state.game = &payload.Game
	case "game_over":
		var payload gameOverSnapshot
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			t.Fatalf("decode game_over: %v", err)
		}
		state.gameOver = &payload
		state.game = &payload.Game
	case "error":
		var payload errorSnapshot
		if err := json.Unmarshal(envelope.Data, &payload); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		state.errors = append(state.errors, payload.Message)
	}
}

func pumpClient(t *testing.T, client *wsClient) {
	t.Helper()

	for {
		select {
		case envelope, ok := <-client.inbox:
			if !ok {
				select {
				case err := <-client.errs:
					if err != nil {
						t.Fatalf("websocket reader stopped: %v", err)
					}
				default:
				}
				return
			}
			applyEnvelope(t, client.state, envelope)
		case err := <-client.errs:
			if err != nil {
				t.Fatalf("websocket reader error: %v", err)
			}
		default:
			return
		}
	}
}

func awaitState(t *testing.T, timeout time.Duration, step func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if step() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("condition not met within %s", timeout)
}

func joinRoom(t *testing.T, client *wsClient, roomID, playerName string) {
	t.Helper()

	sendWSMessage(t, client, "join_room", map[string]any{
		"room_id":     roomID,
		"player_name": playerName,
	})

	awaitState(t, 2*time.Second, func() bool {
		pumpClient(t, client)
		return client.state.clientID != "" && client.state.room != nil
	})
}

func playerIndexByClientID(snapshot *gameSnapshot, clientID string) int {
	if snapshot == nil {
		return -1
	}
	for idx, player := range snapshot.Players {
		if player.ID == clientID {
			return idx
		}
	}
	return -1
}

func waitForTurnCount(t *testing.T, clientA *wsClient, clientB *wsClient, turnCount int) {
	t.Helper()

	awaitState(t, 2*time.Second, func() bool {
		pumpClient(t, clientA)
		pumpClient(t, clientB)
		return clientA.state.game != nil && clientB.state.game != nil &&
			clientA.state.game.TurnCount >= turnCount && clientB.state.game.TurnCount >= turnCount
	})
}

func currentConn(snapshot *gameSnapshot, clientA *wsClient, clientB *wsClient) (*wsClient, int) {
	indexA := playerIndexByClientID(snapshot, clientA.state.clientID)
	if indexA == snapshot.CurrentPlayer {
		return clientA, indexA
	}
	return clientB, playerIndexByClientID(snapshot, clientB.state.clientID)
}

func TestWebSocketFullGameFlow(t *testing.T) {
	server, manager := newTestServer(t)
	defer server.Close()

	roomID := manager.CreateRoom().ID

	clientA := dialWS(t, server.URL)
	defer clientA.conn.Close()
	clientB := dialWS(t, server.URL)
	defer clientB.conn.Close()

	joinRoom(t, clientA, roomID, "Alice")
	joinRoom(t, clientB, roomID, "Bob")

	sendWSMessage(t, clientA, "set_ready", map[string]any{"ready": true})
	sendWSMessage(t, clientB, "set_ready", map[string]any{"ready": true})

	readyDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(readyDeadline) {
		pumpClient(t, clientA)
		pumpClient(t, clientB)
		if clientA.state.game != nil && clientB.state.game != nil && clientA.state.game.State == int(game.StatePlaying) && clientB.state.game.State == int(game.StatePlaying) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if clientA.state.game == nil || clientB.state.game == nil || clientA.state.game.State != int(game.StatePlaying) || clientB.state.game.State != int(game.StatePlaying) {
		room := manager.GetRoom(roomID)
		t.Fatalf("game did not start: room_game=%+v A_game=%+v A_errors=%v B_game=%+v B_errors=%v", room.Game.GetGameState(), clientA.state.game, clientA.state.errors, clientB.state.game, clientB.state.errors)
	}

	openingMoves := []positionSnapshot{{X: 7, Y: 7}, {X: 7, Y: 8}, {X: 8, Y: 7}}
	ownership := map[string]string{}

	for idx, move := range openingMoves {
		activeClient, _ := currentConn(clientA.state.game, clientA, clientB)
		sendWSMessage(t, activeClient, "make_move", map[string]any{
			"x": move.X,
			"y": move.Y,
		})
		ownership[fmt.Sprintf("%d,%d", move.X, move.Y)] = activeClient.state.clientID
		waitForTurnCount(t, clientA, clientB, idx+1)
	}

	skillClient, skillPlayerIndex := currentConn(clientA.state.game, clientA, clientB)
	if clientA.state.game.Players[skillPlayerIndex].Mana < 30 {
		t.Fatalf("expected current player to have enough mana for roar, got %d", clientA.state.game.Players[skillPlayerIndex].Mana)
	}

	var target positionSnapshot
	foundTarget := false
	for key, owner := range ownership {
		if owner == skillClient.state.clientID {
			continue
		}
		if _, err := fmt.Sscanf(key, "%d,%d", &target.X, &target.Y); err == nil {
			foundTarget = true
			break
		}
	}
	if !foundTarget {
		t.Fatal("expected an opponent piece to target with roar")
	}

	sendWSMessage(t, skillClient, "use_skill", map[string]any{
		"skill_type": int(game.SkillRoar),
		"target_positions": []map[string]any{{
			"x": target.X,
			"y": target.Y,
		}},
	})

	awaitState(t, 2*time.Second, func() bool {
		pumpClient(t, clientA)
		pumpClient(t, clientB)
		return clientA.state.game != nil && clientB.state.game != nil &&
			clientA.state.game.Board[target.X][target.Y] == game.Empty &&
			clientB.state.game.Board[target.X][target.Y] == game.Empty
	})

	if got := clientA.state.game.Board[target.X][target.Y]; got != game.Empty {
		t.Fatalf("expected roar target to be removed, got %d", got)
	}

	if clientA.state.game.CurrentPlayer != skillPlayerIndex {
		t.Fatalf("expected roar to keep the same current player, got %d", clientA.state.game.CurrentPlayer)
	}

	sendWSMessage(t, skillClient, "make_move", map[string]any{
		"x": 9,
		"y": 9,
	})
	waitForTurnCount(t, clientA, clientB, 4)

	room := manager.GetRoom(roomID)
	if room == nil {
		t.Fatal("room not found after skill test")
	}

	sitDownClient, sitDownPlayerIndex := currentConn(clientA.state.game, clientA, clientB)
	room.Game.Players[sitDownPlayerIndex].Mana = 100

	sendWSMessage(t, sitDownClient, "use_skill", map[string]any{
		"skill_type":       int(game.SkillSitDown),
		"target_positions": []map[string]any{},
	})

	awaitState(t, 2*time.Second, func() bool {
		pumpClient(t, clientA)
		pumpClient(t, clientB)
		return clientA.state.game != nil && clientB.state.game != nil &&
			clientA.state.game.CurrentPlayer == sitDownPlayerIndex &&
			clientA.state.game.SkippedPlayer == room.Game.Players[1-sitDownPlayerIndex].Color
	})

	sendWSMessage(t, sitDownClient, "make_move", map[string]any{
		"x": 9,
		"y": 10,
	})
	waitForTurnCount(t, clientA, clientB, 5)

	if clientA.state.game.CurrentPlayer != sitDownPlayerIndex {
		t.Fatalf("expected sit down to skip opponent turn, current player = %d", clientA.state.game.CurrentPlayer)
	}

	room = manager.GetRoom(roomID)
	if room == nil {
		t.Fatal("room not found before game over setup")
	}

	winnerIndex := clientA.state.game.CurrentPlayer
	winnerName := room.Game.Players[winnerIndex].Name
	winnerClient := clientA
	if playerIndexByClientID(clientA.state.game, clientA.state.clientID) != winnerIndex {
		winnerClient = clientB
	}

	room.Game.Board = game.NewBoard()
	room.Game.State = game.StatePlaying
	room.Game.Winner = -1
	room.Game.SkippedPlayer = -1
	room.Game.LastMove = game.Position{X: -1, Y: -1}
	room.Game.CurrentPlayer = winnerIndex
	room.Game.TurnCount = 9

	color := room.Game.Players[winnerIndex].Color
	for i := 0; i < 4; i++ {
		room.Game.Board.Grid[5+i][5] = color
	}

	sendWSMessage(t, winnerClient, "make_move", map[string]any{
		"x": 9,
		"y": 5,
	})

	awaitState(t, 2*time.Second, func() bool {
		pumpClient(t, clientA)
		pumpClient(t, clientB)
		return clientA.state.gameOver != nil && clientB.state.gameOver != nil
	})

	if clientA.state.game == nil || clientA.state.game.State != int(game.StateFinished) {
		t.Fatalf("expected finished game state, got %+v", clientA.state.game)
	}
	if clientA.state.game.Winner != winnerIndex {
		t.Fatalf("expected winner index %d, got %d", winnerIndex, clientA.state.game.Winner)
	}
	if clientA.state.gameOver.Winner != winnerName {
		t.Fatalf("expected winner name %q, got %q", winnerName, clientA.state.gameOver.Winner)
	}
}

func TestJoinFullRoomGetsError(t *testing.T) {
	server, manager := newTestServer(t)
	defer server.Close()

	roomID := manager.CreateRoom().ID

	clientA := dialWS(t, server.URL)
	defer clientA.conn.Close()
	clientB := dialWS(t, server.URL)
	defer clientB.conn.Close()
	clientC := dialWS(t, server.URL)
	defer clientC.conn.Close()

	joinRoom(t, clientA, roomID, "Alice")
	joinRoom(t, clientB, roomID, "Bob")

	sendWSMessage(t, clientC, "join_room", map[string]any{
		"room_id":     roomID,
		"player_name": "Carol",
	})

	awaitState(t, 2*time.Second, func() bool {
		pumpClient(t, clientC)
		return len(clientC.state.errors) > 0
	})

	if got := clientC.state.errors[len(clientC.state.errors)-1]; got == "" {
		t.Fatal("expected a non-empty join error for third player")
	}
	if room := manager.GetRoom(roomID); room == nil || len(room.Clients) != 2 {
		t.Fatalf("expected room to keep exactly 2 clients, got %+v", room)
	}
}

func TestCleanupRemovesExpiredClients(t *testing.T) {
	manager := roompkg.NewManager()
	roomID := manager.CreateRoom().ID

	room, client, err := manager.JoinRoomWithClient(roomID, "Alice")
	if err != nil {
		t.Fatalf("join room: %v", err)
	}

	client.LastSeen = time.Now().Add(-31 * time.Minute)

	done := make(chan struct{})
	go func() {
		manager.Cleanup()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("cleanup deadlocked")
	}

	if got := manager.GetRoom(room.ID); got != nil {
		t.Fatalf("expected expired client room to be removed, got %+v", got)
	}
	if len(manager.Clients) != 0 {
		t.Fatalf("expected all expired clients to be removed, got %d", len(manager.Clients))
	}
}
