package main

import (
	"comedy-gobang/internal/game"
	"testing"
)

func TestBoardPlacePiece(t *testing.T) {
	board := game.NewBoard()

	// 测试正常落子
	err := board.PlacePiece(7, 7, game.Black)
	if err != nil {
		t.Errorf("落子失败: %v", err)
	}

	// 检查棋子是否正确放置
	if board.GetPiece(7, 7) != game.Black {
		t.Error("棋子未正确放置")
	}

	// 测试重复落子
	err = board.PlacePiece(7, 7, game.White)
	if err == nil {
		t.Error("应该不允许重复落子")
	}

	// 测试超出边界
	err = board.PlacePiece(-1, 7, game.Black)
	if err == nil {
		t.Error("应该不允许超出边界的落子")
	}

	err = board.PlacePiece(7, 15, game.Black)
	if err == nil {
		t.Error("应该不允许超出边界的落子")
	}
}

func TestBoardCheckWin(t *testing.T) {
	board := game.NewBoard()

	// 测试水平五子连珠
	for i := 0; i < 5; i++ {
		board.PlacePiece(5+i, 5, game.Black)
	}

	if !board.CheckWin(9, 5, game.Black) {
		t.Error("水平五子连珠应该获胜")
	}

	// 重置棋盘
	board = game.NewBoard()

	// 测试垂直五子连珠
	for i := 0; i < 5; i++ {
		board.PlacePiece(5, 5+i, game.Black)
	}

	if !board.CheckWin(5, 9, game.Black) {
		t.Error("垂直五子连珠应该获胜")
	}

	// 重置棋盘
	board = game.NewBoard()

	// 测试对角线五子连珠
	for i := 0; i < 5; i++ {
		board.PlacePiece(5+i, 5+i, game.Black)
	}

	if !board.CheckWin(9, 9, game.Black) {
		t.Error("对角线五子连珠应该获胜")
	}
}

func TestGameBasicFlow(t *testing.T) {
	g := game.NewGame("test")

	// 添加玩家
	err := g.AddPlayer("player1", "玩家1")
	if err != nil {
		t.Errorf("添加玩家1失败: %v", err)
	}

	err = g.AddPlayer("player2", "玩家2")
	if err != nil {
		t.Errorf("添加玩家2失败: %v", err)
	}

	// 设置准备状态
	err = g.SetPlayerReady("player1", true)
	if err != nil {
		t.Errorf("玩家1准备失败: %v", err)
	}

	err = g.SetPlayerReady("player2", true)
	if err != nil {
		t.Errorf("玩家2准备失败: %v", err)
	}

	// 检查游戏是否开始
	if g.State != game.StatePlaying {
		t.Error("游戏应该已经开始")
	}

	// 测试落子
	previousPlayer := g.CurrentPlayer
	currentPlayer := g.GetCurrentPlayer()
	err = g.MakeMove(currentPlayer.ID, 7, 7)
	if err != nil {
		t.Errorf("落子失败: %v", err)
	}

	// 检查回合是否切换
	if g.CurrentPlayer == previousPlayer {
		t.Error("回合应该已经切换")
	}
}

func TestSkills(t *testing.T) {
	skill := game.NewSkill(game.SkillRoar)

	// 测试技能初始状态
	if skill.CurrentCD != 0 {
		t.Error("技能初始冷却时间应该为0")
	}

	// 测试技能使用
	err := skill.Use(30)
	if err != nil {
		t.Errorf("使用技能失败: %v", err)
	}

	// 检查冷却时间
	if skill.CurrentCD != skill.Cooldown {
		t.Error("技能应该进入冷却")
	}

	// 测试冷却中的技能
	err = skill.Use(30)
	if err == nil {
		t.Error("冷却中的技能不应该能够使用")
	}

	// 测试能量不足
	skill.CurrentCD = 0
	err = skill.Use(10)
	if err == nil {
		t.Error("能量不足时技能不应该能够使用")
	}
}

func TestQuoteEngine(t *testing.T) {
	engine := game.NewQuoteEngine()

	// 测试获取语录
	quote := engine.GetRandomQuote(game.QuoteNormal)
	if quote == nil {
		t.Error("应该能够获取语录")
	}

	if quote.Type != game.QuoteNormal {
		t.Error("语录类型不正确")
	}

	if quote.Text == "" {
		t.Error("语录内容不应该为空")
	}
}

// 基准测试
func BenchmarkBoardPlacePiece(b *testing.B) {
	board := game.NewBoard()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := i % game.BoardSize
		y := (i / game.BoardSize) % game.BoardSize
		board.PlacePiece(x, y, game.Black)
	}
}

func BenchmarkBoardCheckWin(b *testing.B) {
	board := game.NewBoard()

	// 放置一些棋子
	for i := 0; i < 5; i++ {
		board.PlacePiece(5+i, 5, game.Black)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.CheckWin(7, 5, game.Black)
	}
}

func TestRoarTargetsOpponentPiece(t *testing.T) {
	g := game.NewGame("skill-roar")
	if err := g.AddPlayer("p1", "P1"); err != nil {
		t.Fatalf("add player 1: %v", err)
	}
	if err := g.AddPlayer("p2", "P2"); err != nil {
		t.Fatalf("add player 2: %v", err)
	}

	g.State = game.StatePlaying
	g.CurrentPlayer = 0
	g.Players[0].Mana = 100
	g.Board.Grid[7][7] = game.White

	if err := g.UseSkill("p1", game.SkillRoar, []game.Position{{X: 7, Y: 7}}); err != nil {
		t.Fatalf("roar should remove opponent piece: %v", err)
	}

	if got := g.Board.GetPiece(7, 7); got != game.Empty {
		t.Fatalf("expected target piece removed, got %d", got)
	}

	if g.CurrentPlayer != 0 {
		t.Fatalf("expected turn to stay with caster after roar, current player = %d", g.CurrentPlayer)
	}
}

func TestSkillCanBeUsedBeforeMoveAndSitDownSkipsOpponentTurn(t *testing.T) {
	g := game.NewGame("skill-sitdown")
	if err := g.AddPlayer("p1", "P1"); err != nil {
		t.Fatalf("add player 1: %v", err)
	}
	if err := g.AddPlayer("p2", "P2"); err != nil {
		t.Fatalf("add player 2: %v", err)
	}

	g.State = game.StatePlaying
	g.CurrentPlayer = 0
	g.Players[0].Mana = 100

	if err := g.UseSkill("p1", game.SkillSitDown, nil); err != nil {
		t.Fatalf("sit down should succeed: %v", err)
	}

	if g.CurrentPlayer != 0 {
		t.Fatalf("expected caster to keep turn after using skill, current player = %d", g.CurrentPlayer)
	}

	if g.SkippedPlayer != game.White {
		t.Fatalf("expected white player to be marked as skipped, got %d", g.SkippedPlayer)
	}

	if err := g.MakeMove("p1", 7, 7); err != nil {
		t.Fatalf("expected player to still be able to move after skill: %v", err)
	}

	if g.CurrentPlayer != 0 {
		t.Fatalf("expected opponent turn to be skipped after move, current player = %d", g.CurrentPlayer)
	}

	if g.SkippedPlayer != -1 {
		t.Fatalf("expected skipped player marker to be cleared after move, got %d", g.SkippedPlayer)
	}
}

func TestReversalRandomlySwapsBlackAndWhitePieces(t *testing.T) {
	g := game.NewGame("skill-reversal")
	if err := g.AddPlayer("p1", "P1"); err != nil {
		t.Fatalf("add player 1: %v", err)
	}
	if err := g.AddPlayer("p2", "P2"); err != nil {
		t.Fatalf("add player 2: %v", err)
	}

	g.State = game.StatePlaying
	g.CurrentPlayer = 0
	g.Players[0].Mana = 100
	if got := g.Players[0].Skills[2].ManaCost; got != 60 {
		t.Fatalf("expected reversal mana cost to be 60, got %d", got)
	}
	g.Board.Grid[3][3] = game.Black
	g.Board.Grid[10][10] = game.White

	if err := g.UseSkill("p1", game.SkillReversal, nil); err != nil {
		t.Fatalf("reversal should randomly swap pieces: %v", err)
	}

	if got := g.Board.GetPiece(3, 3); got != game.White {
		t.Fatalf("expected white piece at original black position, got %d", got)
	}

	if got := g.Board.GetPiece(10, 10); got != game.Black {
		t.Fatalf("expected black piece at original white position, got %d", got)
	}

	if g.CurrentPlayer != 0 {
		t.Fatalf("expected turn to stay with caster after reversal, current player = %d", g.CurrentPlayer)
	}
}
