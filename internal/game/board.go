package game

import (
	"fmt"
	"math/rand"
	"time"
)

// 棋盘常量
const (
	BoardSize = 15
	Empty     = 0
	Black     = 1
	White     = 2
)

// Position 棋盘位置
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Board 棋盘结构
type Board struct {
	Grid [BoardSize][BoardSize]int `json:"grid"`
}

// NewBoard 创建新棋盘
func NewBoard() *Board {
	board := &Board{}
	// 初始化空棋盘
	for i := 0; i < BoardSize; i++ {
		for j := 0; j < BoardSize; j++ {
			board.Grid[i][j] = Empty
		}
	}
	return board
}

// PlacePiece 放置棋子
func (b *Board) PlacePiece(x, y, color int) error {
	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		return fmt.Errorf("位置超出棋盘范围")
	}
	
	if b.Grid[x][y] != Empty {
		return fmt.Errorf("该位置已有棋子")
	}
	
	b.Grid[x][y] = color
	return nil
}

// RemovePiece 移除棋子
func (b *Board) RemovePiece(x, y int) error {
	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		return fmt.Errorf("位置超出棋盘范围")
	}
	
	if b.Grid[x][y] == Empty {
		return fmt.Errorf("该位置没有棋子")
	}
	
	b.Grid[x][y] = Empty
	return nil
}

// GetPiece 获取棋子颜色
func (b *Board) GetPiece(x, y int) int {
	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		return -1
	}
	return b.Grid[x][y]
}

// CheckWin 检查是否获胜
func (b *Board) CheckWin(x, y, color int) bool {
	// 检查四个方向：水平、垂直、对角线1、对角线2
	directions := [][2]int{
		{0, 1},  // 水平
		{1, 0},  // 垂直
		{1, 1},  // 对角线1
		{1, -1}, // 对角线2
	}
	
	for _, dir := range directions {
		count := 1 // 包含当前位置
		
		// 正方向检查
		for i := 1; i < 5; i++ {
			nx, ny := x+dir[0]*i, y+dir[1]*i
			if nx < 0 || nx >= BoardSize || ny < 0 || ny >= BoardSize {
				break
			}
			if b.Grid[nx][ny] != color {
				break
			}
			count++
		}
		
		// 反方向检查
		for i := 1; i < 5; i++ {
			nx, ny := x-dir[0]*i, y-dir[1]*i
			if nx < 0 || nx >= BoardSize || ny < 0 || ny >= BoardSize {
				break
			}
			if b.Grid[nx][ny] != color {
				break
			}
			count++
		}
		
		if count >= 5 {
			return true
		}
	}
	
	return false
}

// GetRandomPieces 获取随机指定颜色的棋子位置
func (b *Board) GetRandomPieces(color, count int) []Position {
	var pieces []Position
	
	for i := 0; i < BoardSize; i++ {
		for j := 0; j < BoardSize; j++ {
			if b.Grid[i][j] == color {
				pieces = append(pieces, Position{X: i, Y: j})
			}
		}
	}
	
	if len(pieces) <= count {
		return pieces
	}
	
	// 随机选择
	rand.Seed(time.Now().UnixNano())
	result := make([]Position, 0, count)
	indices := rand.Perm(len(pieces))
	
	for i := 0; i < count; i++ {
		result = append(result, pieces[indices[i]])
	}
	
	return result
}

// SwapPieces 交换两颗棋子的位置
func (b *Board) SwapPieces(pos1, pos2 Position) error {
	piece1 := b.GetPiece(pos1.X, pos1.Y)
	piece2 := b.GetPiece(pos2.X, pos2.Y)
	
	if piece1 == -1 || piece2 == -1 {
		return fmt.Errorf("无效的位置")
	}
	
	b.Grid[pos1.X][pos1.Y] = piece2
	b.Grid[pos2.X][pos2.Y] = piece1
	
	return nil
}

// GetBoardState 获取棋盘状态（用于发送给前端）
func (b *Board) GetBoardState() [][]int {
	state := make([][]int, BoardSize)
	for i := 0; i < BoardSize; i++ {
		state[i] = make([]int, BoardSize)
		for j := 0; j < BoardSize; j++ {
			state[i][j] = b.Grid[i][j]
		}
	}
	return state
}
