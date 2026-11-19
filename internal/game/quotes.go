package game

import (
	"math/rand"
	"time"
)

// QuoteType 语录类型
type QuoteType int

const (
	QuoteNormal QuoteType = iota // 普通落子
	QuoteBlock                   // 围堵成功
	QuoteLongThink               // 长考
	QuoteSkill                   // 使用技能
	QuoteWin                     // 获胜
	QuoteLose                    // 失败
	QuoteDanger                  // 危险局面
)

// Quote 语录结构
type Quote struct {
	Text   string     `json:"text"`
	Type   QuoteType  `json:"type"`
	Author string     `json:"author"`
}

// QuoteEngine 语录引擎
type QuoteEngine struct {
	quotes map[QuoteType][]string
	rand   *rand.Rand
}

// NewQuoteEngine 创建语录引擎
func NewQuoteEngine() *QuoteEngine {
	qe := &QuoteEngine{
		quotes: make(map[QuoteType][]string),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	
	// 初始化语录库
	qe.initQuotes()
	return qe
}

// initQuotes 初始化语录库
func (qe *QuoteEngine) initQuotes() {
	// 普通落子语录
	qe.quotes[QuoteNormal] = []string{
		"嘿，这棋下的，很有画面感啊！",
		"逻辑！注意你的逻辑！",
		"你这步棋，很有想法啊！",
		"嗯，这步棋有意思。",
		"让我想想...这步棋不简单。",
		"你这人，有点东西啊！",
		"这棋下的，我有点看不懂了。",
		"有意思，有意思！",
		"你这是在干嘛？我有点懵。",
		"这步棋...有点东西！",
	}
	
	// 围堵成功语录
	qe.quotes[QuoteBlock] = []string{
		"你这是把我往死里整啊，兄弟。",
		"你这人，怎么这么坏呢！",
		"我太难了！真的太难了！",
		"你这是要我的命啊！",
		"完了完了，这下麻烦了。",
		"你这步棋，太狠了！",
		"我受不了了！真的受不了了！",
		"你这是要赶尽杀绝啊！",
		"我的心好痛啊！",
		"这还怎么玩？这还怎么玩？",
	}
	
	// 长考语录
	qe.quotes[QuoteLongThink] = []string{
		"逻辑！注意你的逻辑！",
		"想什么呢？想那么久？",
		"你这人，做事太墨迹了！",
		"快点快点，等不及了！",
		"想好了没有？我等着呢！",
		"你这思考速度，有点慢啊！",
		"是不是想不出来？要不要我帮你？",
		"时间就是生命，你懂不懂？",
		"再想下去，天都黑了！",
		"你这思考姿势，很有水平啊！",
	}
	
	// 使用技能语录
	qe.quotes[QuoteSkill] = []string{
		"厉害厉害！这技能用得漂亮！",
		"我靠！还有这种操作？",
		"这技能...有点东西啊！",
		"完蛋了，这下要出事！",
		"你这是要搞事情啊！",
		"这技能用得，我有点懵。",
		"厉害！这波操作我给满分！",
		"我服了！我真的服了！",
		"你这人，太会玩了！",
		"这技能...绝了！",
	}
	
	// 获胜语录
	qe.quotes[QuoteWin] = []string{
		"赢了！我赢了！哈哈哈！",
		"不好意思，让你失望了！",
		"这就对了！这就对了！",
		"我太厉害了！我太厉害了！",
		"承让承让！承让！",
		"这就结束了？太快了吧！",
		"不好意思，技不如人啊！",
		"这就叫实力！懂不懂？",
		"赢了！今晚加鸡腿！",
		"完美！这局完美！",
	}
	
	// 失败语录
	qe.quotes[QuoteLose] = []string{
		"我输了？我不服！我不服！",
		"完了完了，这下丢人了。",
		"你等着，我下次一定赢你！",
		"不算不算，这局不算！",
		"我大意了！没有闪！",
		"你这是侥幸！绝对是侥幸！",
		"再来一局！我一定能赢！",
		"我太难了！真的太难了！",
		"不玩了不玩了！不玩了！",
		"你等着，我练练再来！",
	}
	
	// 危险局面语录
	qe.quotes[QuoteDanger] = []string{
		"坏了坏了，这下危险了！",
		"完了完了，要输了要输了！",
		"你这是要绝杀啊！",
		"我有点慌了！真的有点慌！",
		"这局面...有点棘手啊！",
		"完了，这下要出事！",
		"你这是要我的命啊！",
		"我受不了了！压力好大！",
		"这还怎么下？这还怎么下？",
		"救命啊！谁来救救我！",
	}
}

// GetRandomQuote 获取随机语录
func (qe *QuoteEngine) GetRandomQuote(quoteType QuoteType) *Quote {
	quotes, exists := qe.quotes[quoteType]
	if !exists || len(quotes) == 0 {
		return &Quote{
			Text:   "嘿嘿，有意思！",
			Type:   QuoteNormal,
			Author: "系统",
		}
	}
	
	text := quotes[qe.rand.Intn(len(quotes))]
	
	return &Quote{
		Text:   text,
		Type:   quoteType,
		Author: "系统",
	}
}

// GetContextualQuote 根据游戏上下文获取语录
func (qe *QuoteEngine) GetContextualQuote(game *Game, action string, playerID string) *Quote {
	player := game.GetPlayerByID(playerID)
	if player == nil {
		return qe.GetRandomQuote(QuoteNormal)
	}
	
	switch action {
	case "move":
		// 检查是否是围堵
		if qe.isBlockingMove(game, player) {
			return qe.GetRandomQuote(QuoteBlock)
		}
		return qe.GetRandomQuote(QuoteNormal)
		
	case "skill":
		return qe.GetRandomQuote(QuoteSkill)
		
	case "win":
		return qe.GetRandomQuote(QuoteWin)
		
	case "lose":
		return qe.GetRandomQuote(QuoteLose)
		
	default:
		return qe.GetRandomQuote(QuoteNormal)
	}
}

// isBlockingMove 检查是否是围堵操作
func (qe *QuoteEngine) isBlockingMove(game *Game, player *Player) bool {
	// 简单的围堵判断：检查最后一步棋是否阻止了对方形成连线
	if game.LastMove.X == -1 || game.LastMove.Y == -1 {
		return false
	}
	
	x, y := game.LastMove.X, game.LastMove.Y
	opponentColor := White
	if player.Color == White {
		opponentColor = Black
	}
	
	// 临时移除这颗棋子，检查对方是否能形成五子
	game.Board.Grid[x][y] = Empty
	
	// 检查周围是否有对方的棋子形成威胁
	directions := [][2]int{
		{0, 1}, {1, 0}, {1, 1}, {1, -1},
	}
	
	for _, dir := range directions {
		count := 0
		for i := -2; i <= 2; i++ {
			if i == 0 {
				continue
			}
			nx, ny := x+dir[0]*i, y+dir[1]*i
			if nx >= 0 && nx < BoardSize && ny >= 0 && ny < BoardSize {
				if game.Board.Grid[nx][ny] == opponentColor {
					count++
				}
			}
		}
		
		// 如果周围有3个或以上的对方棋子，可能是围堵
		if count >= 3 {
			game.Board.Grid[x][y] = player.Color
			return true
		}
	}
	
	// 恢复棋子
	game.Board.Grid[x][y] = player.Color
	return false
}

// GetLongThinkQuote 获取长考语录
func (qe *QuoteEngine) GetLongThinkQuote() *Quote {
	return qe.GetRandomQuote(QuoteLongThink)
}

// GetDangerQuote 获取危险语录
func (qe *QuoteEngine) GetDangerQuote() *Quote {
	return qe.GetRandomQuote(QuoteDanger)
}
