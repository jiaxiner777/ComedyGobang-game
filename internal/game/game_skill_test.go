package game

import "testing"

func TestResolveSkillWinFinishesGameForCurrentPlayer(t *testing.T) {
	g := NewGame("skill-win-current")
	g.State = StatePlaying
	g.CurrentPlayer = 0
	g.Winner = -1

	g.Board.Grid[3][7] = Black
	g.Board.Grid[4][7] = Black
	g.Board.Grid[5][7] = Black
	g.Board.Grid[6][7] = Black
	g.Board.Grid[7][7] = Black
	g.Board.Grid[10][10] = White

	g.resolveSkillWin([]Position{{X: 3, Y: 7}, {X: 10, Y: 10}})

	if g.State != StateFinished {
		t.Fatalf("expected game to finish after skill-created five in a row, got state %d", g.State)
	}
	if g.Winner != 0 {
		t.Fatalf("expected current player to win, got winner %d", g.Winner)
	}
	if g.LastMove != (Position{X: 3, Y: 7}) {
		t.Fatalf("expected last move marker to point to winning skill position, got %+v", g.LastMove)
	}
}

func TestResolveSkillWinCanAwardOpponentIfSkillCreatesOpponentFive(t *testing.T) {
	g := NewGame("skill-win-opponent")
	g.State = StatePlaying
	g.CurrentPlayer = 0
	g.Winner = -1

	g.Board.Grid[8][3] = White
	g.Board.Grid[8][4] = White
	g.Board.Grid[8][5] = White
	g.Board.Grid[8][6] = White
	g.Board.Grid[8][7] = White
	g.Board.Grid[1][1] = Black

	g.resolveSkillWin([]Position{{X: 1, Y: 1}, {X: 8, Y: 3}})

	if g.State != StateFinished {
		t.Fatalf("expected game to finish when opponent gets five from skill, got state %d", g.State)
	}
	if g.Winner != 1 {
		t.Fatalf("expected opponent to win, got winner %d", g.Winner)
	}
}
