package game

import (
	"fmt"
	"math/rand"
	"time"
)

type GameState int

const (
	StateWaiting GameState = iota
	StatePlaying
	StateFinished
)

type Player struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Color  int      `json:"color"`
	Mana   int      `json:"mana"`
	Skills []*Skill `json:"skills"`
	Ready  bool     `json:"ready"`
}

type Game struct {
	ID            string     `json:"id"`
	Board         *Board     `json:"board"`
	Players       [2]*Player `json:"players"`
	CurrentPlayer int        `json:"current_player"`
	State         GameState  `json:"state"`
	Winner        int        `json:"winner"`
	SkippedPlayer int        `json:"skipped_player"`
	TurnCount     int        `json:"turn_count"`
	LastMove      Position   `json:"last_move"`
}

func NewGame(gameID string) *Game {
	game := &Game{
		ID:            gameID,
		Board:         NewBoard(),
		CurrentPlayer: 0,
		State:         StateWaiting,
		Winner:        -1,
		SkippedPlayer: -1,
		TurnCount:     0,
		LastMove:      Position{X: -1, Y: -1},
	}

	game.Players[0] = &Player{
		ID:    "",
		Name:  "玩家1",
		Color: Black,
		Mana:  0,
		Skills: []*Skill{
			NewSkill(SkillRoar),
			NewSkill(SkillSitDown),
			NewSkill(SkillReversal),
		},
		Ready: false,
	}

	game.Players[1] = &Player{
		ID:    "",
		Name:  "玩家2",
		Color: White,
		Mana:  0,
		Skills: []*Skill{
			NewSkill(SkillRoar),
			NewSkill(SkillSitDown),
			NewSkill(SkillReversal),
		},
		Ready: false,
	}

	return game
}

func (g *Game) AddPlayer(playerID, playerName string) error {
	if g.State != StateWaiting {
		return fmt.Errorf("游戏已经开始或已结束")
	}

	if g.Players[0].ID == "" {
		g.Players[0].ID = playerID
		g.Players[0].Name = playerName
		return nil
	}

	if g.Players[1].ID == "" {
		g.Players[1].ID = playerID
		g.Players[1].Name = playerName
		return nil
	}

	return fmt.Errorf("游戏已满")
}

func (g *Game) RemovePlayer(playerID string) {
	if g.Players[0].ID == playerID {
		g.Players[0].ID = ""
		g.Players[0].Ready = false
	}

	if g.Players[1].ID == playerID {
		g.Players[1].ID = ""
		g.Players[1].Ready = false
	}

	if g.State == StatePlaying {
		g.State = StateWaiting
		g.Board = NewBoard()
		g.CurrentPlayer = 0
		g.Winner = -1
		g.SkippedPlayer = -1
		g.TurnCount = 0
		g.LastMove = Position{X: -1, Y: -1}
		g.Players[0].Mana = 0
		g.Players[1].Mana = 0
	}
}

func (g *Game) SetPlayerReady(playerID string, ready bool) error {
	player := g.GetPlayerByID(playerID)
	if player == nil {
		return fmt.Errorf("玩家不存在")
	}

	player.Ready = ready

	if g.Players[0].Ready && g.Players[1].Ready && g.State == StateWaiting {
		g.State = StatePlaying
		g.StartGame()
	}

	return nil
}

func (g *Game) StartGame() {
	rand.Seed(time.Now().UnixNano())
	if rand.Intn(2) == 1 {
		g.CurrentPlayer = 1
	}

	g.TurnCount = 0
	g.SkippedPlayer = -1
	g.LastMove = Position{X: -1, Y: -1}
	g.Players[0].Mana = 20
	g.Players[1].Mana = 20
}

func (g *Game) GetPlayerByID(playerID string) *Player {
	if g.Players[0].ID == playerID {
		return g.Players[0]
	}
	if g.Players[1].ID == playerID {
		return g.Players[1]
	}
	return nil
}

func (g *Game) GetCurrentPlayer() *Player {
	return g.Players[g.CurrentPlayer]
}

func (g *Game) MakeMove(playerID string, x, y int) error {
	if g.State != StatePlaying {
		return fmt.Errorf("游戏未开始或已结束")
	}

	currentPlayer := g.GetCurrentPlayer()
	if currentPlayer.ID != playerID {
		return fmt.Errorf("不是你的回合")
	}

	if g.SkippedPlayer == currentPlayer.Color {
		return fmt.Errorf("你这回合被禁手了")
	}

	if err := g.Board.PlacePiece(x, y, currentPlayer.Color); err != nil {
		return err
	}

	g.LastMove = Position{X: x, Y: y}
	g.TurnCount++

	currentPlayer.Mana += 5
	if currentPlayer.Mana > 100 {
		currentPlayer.Mana = 100
	}

	if g.Board.CheckWin(x, y, currentPlayer.Color) {
		g.Winner = g.CurrentPlayer
		g.State = StateFinished
		return nil
	}

	g.SwitchTurn()
	return nil
}

func (g *Game) UseSkill(playerID string, skillType SkillType, targetPositions []Position) error {
	if g.State != StatePlaying {
		return fmt.Errorf("游戏未开始或已结束")
	}

	currentPlayer := g.GetCurrentPlayer()
	if currentPlayer.ID != playerID {
		return fmt.Errorf("不是你的回合")
	}

	if g.SkippedPlayer == currentPlayer.Color {
		return fmt.Errorf("你这回合被禁手了，无法使用技能")
	}

	var skill *Skill
	for _, s := range currentPlayer.Skills {
		if s.Type == skillType {
			skill = s
			break
		}
	}

	if skill == nil {
		return fmt.Errorf("技能不存在")
	}

	if err := skill.Use(currentPlayer.Mana); err != nil {
		return err
	}

	currentPlayer.Mana -= skill.ManaCost

	result := ExecuteSkill(skillType, g, targetPositions)
	if !result.Success {
		currentPlayer.Mana += skill.ManaCost
		skill.CurrentCD = 0
		return fmt.Errorf(result.Message)
	}

	g.resolveSkillWin(result.TargetPos)

	return nil
}

func (g *Game) SwitchTurn() {
	for _, player := range g.Players {
		for _, skill := range player.Skills {
			skill.ReduceCooldown()
		}
	}

	g.CurrentPlayer = 1 - g.CurrentPlayer
	if g.SkippedPlayer == g.GetCurrentPlayer().Color {
		g.CurrentPlayer = 1 - g.CurrentPlayer
	}
	g.SkippedPlayer = -1

	nextPlayer := g.GetCurrentPlayer()
	nextPlayer.Mana += 3
	if nextPlayer.Mana > 100 {
		nextPlayer.Mana = 100
	}
}

func (g *Game) resolveSkillWin(targets []Position) {
	if g.State != StatePlaying {
		return
	}

	colors := []int{
		g.Players[g.CurrentPlayer].Color,
		g.Players[1-g.CurrentPlayer].Color,
	}

	for _, color := range colors {
		for _, pos := range targets {
			if g.Board.GetPiece(pos.X, pos.Y) != color {
				continue
			}
			if !g.Board.CheckWin(pos.X, pos.Y, color) {
				continue
			}

			g.Winner = g.playerIndexByColor(color)
			g.State = StateFinished
			g.LastMove = pos
			return
		}
	}
}

func (g *Game) playerIndexByColor(color int) int {
	for idx, player := range g.Players {
		if player != nil && player.Color == color {
			return idx
		}
	}
	return -1
}

func (g *Game) IsFull() bool {
	return g.Players[0].ID != "" && g.Players[1].ID != ""
}

func (g *Game) GetGameState() map[string]interface{} {
	return map[string]interface{}{
		"id":            g.ID,
		"board":         g.Board.GetBoardState(),
		"players":       g.Players,
		"currentPlayer": g.CurrentPlayer,
		"state":         g.State,
		"winner":        g.Winner,
		"turnCount":     g.TurnCount,
		"lastMove":      g.LastMove,
		"skippedPlayer": g.SkippedPlayer,
	}
}
