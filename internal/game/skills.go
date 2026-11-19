package game

import "fmt"

type SkillType int

const (
	SkillRoar SkillType = iota
	SkillSitDown
	SkillReversal
)

type Skill struct {
	Type        SkillType `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Cooldown    int       `json:"cooldown"`
	CurrentCD   int       `json:"current_cd"`
	ManaCost    int       `json:"mana_cost"`
}

func NewSkill(skillType SkillType) *Skill {
	skills := map[SkillType]Skill{
		SkillRoar: {
			Type:        SkillRoar,
			Name:        "费总的咆哮",
			Description: "强行移除对方棋盘上的任意一颗棋子",
			Cooldown:    3,
			ManaCost:    30,
		},
		SkillSitDown: {
			Type:        SkillSitDown,
			Name:        "淡定一坐",
			Description: "让对方下一回合无法行动",
			Cooldown:    4,
			ManaCost:    40,
		},
		SkillReversal: {
			Type:        SkillReversal,
			Name:        "神级反转",
			Description: "随机交换棋盘上一颗黑棋和一颗白棋的位置",
			Cooldown:    5,
			ManaCost:    60,
		},
	}

	skill := skills[skillType]
	skill.CurrentCD = 0
	return &skill
}

func (s *Skill) CanUse(mana int) bool {
	return s.CurrentCD == 0 && mana >= s.ManaCost
}

func (s *Skill) Use(mana int) error {
	if !s.CanUse(mana) {
		return fmt.Errorf("技能无法使用")
	}

	s.CurrentCD = s.Cooldown
	return nil
}

func (s *Skill) ReduceCooldown() {
	if s.CurrentCD > 0 {
		s.CurrentCD--
	}
}

type SkillResult struct {
	Success      bool       `json:"success"`
	Message      string     `json:"message"`
	Effect       string     `json:"effect"`
	TargetPos    []Position `json:"target_pos,omitempty"`
	VisualEffect string     `json:"visual_effect"`
}

func ExecuteSkill(skillType SkillType, game *Game, targetPositions []Position) *SkillResult {
	switch skillType {
	case SkillRoar:
		return executeRoar(game, targetPositions)
	case SkillSitDown:
		return executeSitDown(game)
	case SkillReversal:
		return executeReversal(game)
	default:
		return &SkillResult{
			Success: false,
			Message: "未知技能类型",
		}
	}
}

func executeRoar(game *Game, targetPositions []Position) *SkillResult {
	if len(targetPositions) != 1 {
		return &SkillResult{
			Success: false,
			Message: "需要选择一个目标棋子",
		}
	}

	targetPos := targetPositions[0]
	targetColor := game.Players[1-game.CurrentPlayer].Color

	if game.Board.GetPiece(targetPos.X, targetPos.Y) != targetColor {
		return &SkillResult{
			Success: false,
			Message: "目标位置没有对方棋子",
		}
	}

	if err := game.Board.RemovePiece(targetPos.X, targetPos.Y); err != nil {
		return &SkillResult{
			Success: false,
			Message: "移除棋子失败: " + err.Error(),
		}
	}

	return &SkillResult{
		Success:      true,
		Message:      "就这？",
		Effect:       "remove_piece",
		TargetPos:    []Position{targetPos},
		VisualEffect: "screen_shake",
	}
}

func executeSitDown(game *Game) *SkillResult {
	game.SkippedPlayer = game.Players[1-game.CurrentPlayer].Color

	return &SkillResult{
		Success:      true,
		Message:      "淡定，先让我坐下...",
		Effect:       "skip_turn",
		VisualEffect: "chair_sit",
	}
}

func executeReversal(game *Game) *SkillResult {
	blackPieces := game.Board.GetRandomPieces(Black, 1)
	whitePieces := game.Board.GetRandomPieces(White, 1)

	if len(blackPieces) == 0 || len(whitePieces) == 0 {
		return &SkillResult{
			Success: false,
			Message: "棋盘上黑白棋子不足，无法发动反转",
		}
	}

	blackPos := blackPieces[0]
	whitePos := whitePieces[0]

	if err := game.Board.SwapPieces(blackPos, whitePos); err != nil {
		return &SkillResult{
			Success: false,
			Message: "交换棋子失败: " + err.Error(),
		}
	}

	return &SkillResult{
		Success:      true,
		Message:      "这就是命！",
		Effect:       "swap_pieces",
		TargetPos:    []Position{blackPos, whitePos},
		VisualEffect: "destiny_flash",
	}
}
