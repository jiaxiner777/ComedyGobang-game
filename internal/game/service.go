package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type GameService struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

type Session struct {
	mu                sync.Mutex
	ID                string
	Game              *Game
	Battle            *Battle
	Phase             string
	Messages          []string
	PendingChoices    []rewardOption
	PendingCards      []CardID
	PendingSyncNode   int
	PendingBossWin    bool
	PendingRewardMsg  string
	PendingRewardLore string
	UpdatedAt         time.Time
}

type StateResponse struct {
	SessionID string        `json:"sessionId"`
	Phase     string        `json:"phase"`
	Player    PlayerView    `json:"player"`
	World     WorldView     `json:"world"`
	Battle    *BattleView   `json:"battle,omitempty"`
	Pending   *PendingView  `json:"pending,omitempty"`
	CardPool  []LibraryCard `json:"cardPool"`
	Messages  []string      `json:"messages"`
	GameOver  *GameOverView `json:"gameOver,omitempty"`
}

type LibraryCard struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Cost       int    `json:"cost"`
	Complexity string `json:"complexity"`
	Text       string `json:"text"`
	Starter    bool   `json:"starter"`
	Source     string `json:"source"`
	Unplayable bool   `json:"unplayable"`
}

type PlayerView struct {
	HP           int         `json:"hp"`
	MaxHP        int         `json:"maxHp"`
	MP           int         `json:"mp"`
	MaxMP        int         `json:"maxMp"`
	Load         int         `json:"load"`
	PendingLoad  int         `json:"pendingLoad"`
	PurgeCredits int         `json:"purgeCredits"`
	Pointers     int         `json:"pointers"`
	Artifacts    []Artifact  `json:"artifacts"`
	Lore         []string    `json:"lore"`
	LoreEntries  []LoreEntry `json:"loreEntries"`
	Deck         []DeckCount `json:"deck"`
	DrawBonus    int         `json:"drawBonus"`
	GCHeal       int         `json:"gcHeal"`
	StackSize    int         `json:"stackSize"`
	StackEcho    int         `json:"stackEcho"`
}

type DeckCount struct {
	Card  string `json:"card"`
	Count int    `json:"count"`
}

type WorldView struct {
	Protocol       string         `json:"protocol"`
	Completion     string         `json:"completion"`
	Path           string         `json:"path"`
	CurrentNode    int            `json:"currentNode"`
	CurrentName    string         `json:"currentName"`
	CurrentKind    string         `json:"currentKind"`
	CurrentDesc    string         `json:"currentDesc"`
	TargetNode     int            `json:"targetNode"`
	TargetName     string         `json:"targetName"`
	AvailableMoves map[string]int `json:"availableMoves"`
	Nodes          []NodeView     `json:"nodes"`
}

type NodeView struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Cleared     bool   `json:"cleared"`
	Balance     int    `json:"balance"`
	Description string `json:"description"`
}

type BattleView struct {
	Turn        int        `json:"turn"`
	TurnLocked  bool       `json:"turnLocked"`
	Enemy       EnemyView  `json:"enemy"`
	Hand        []CardView `json:"hand"`
	DrawSize    int        `json:"drawSize"`
	DiscardSize int        `json:"discardSize"`
	Stack       []string   `json:"stack"`
	Logs        []string   `json:"logs"`
	CanHijack   bool       `json:"canHijack"`
}

type EnemyView struct {
	Name      string   `json:"name"`
	Bits      []int    `json:"bits"`
	Integrity int      `json:"integrity"`
	Attack    int      `json:"attack"`
	Armor     int      `json:"armor"`
	Countdown int      `json:"countdown"`
	Script    int      `json:"script"`
	Entropy   int      `json:"entropy"`
	Intent    string   `json:"intent"`
	Traits    []string `json:"traits"`
	Boss      bool     `json:"boss"`
	Elite     bool     `json:"elite"`
}

type CardView struct {
	Index      int    `json:"index"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	Cost       int    `json:"cost"`
	Complexity string `json:"complexity"`
	Text       string `json:"text"`
	Locked     bool   `json:"locked"`
	Unplayable bool   `json:"unplayable"`
}

type PendingView struct {
	Kind    string       `json:"kind"`
	Message string       `json:"message"`
	Choices []ChoiceView `json:"choices,omitempty"`
}

type ChoiceView struct {
	Index int    `json:"index"`
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

type GameOverView struct {
	Won     bool   `json:"won"`
	Message string `json:"message"`
}

func NewGameService() *GameService {
	return &GameService{sessions: map[string]*Session{}}
}

func (svc *GameService) Create(protocol string) *Session {
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))
	game := NewHeadlessGame(rng)
	game.world.SetProtocol(protocol)
	session := &Session{
		ID:        fmt.Sprintf("%x", seed),
		Game:      game,
		Phase:     "exploring",
		UpdatedAt: time.Now(),
	}
	session.pushMessage("连接已建立，欢迎进入 The Kernel。")
	session.pushMessage("跟随协议目标推进内核树，战斗、拾取奖励并逐步改写 Root。")

	svc.mu.Lock()
	svc.sessions[session.ID] = session
	svc.mu.Unlock()
	return session
}

func (svc *GameService) Get(id string) (*Session, bool) {
	svc.mu.RLock()
	session, ok := svc.sessions[id]
	svc.mu.RUnlock()
	return session, ok
}

func (s *Session) Snapshot() StateResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *Session) snapshotLocked() StateResponse {
	world := s.Game.world
	current := world.CurrentNode()
	target := world.ProtocolTarget()
	view := StateResponse{
		SessionID: s.ID,
		Phase:     s.Phase,
		Player: PlayerView{
			HP:           s.Game.player.HP,
			MaxHP:        s.Game.player.MaxHP,
			MP:           s.Game.player.MP,
			MaxMP:        s.Game.player.EffectiveMaxMP(),
			Load:         s.Game.player.Load,
			PendingLoad:  s.Game.player.PendingLoad,
			PurgeCredits: s.Game.player.PurgeCredits,
			Pointers:     s.Game.player.PointerCharges,
			Artifacts:    append([]Artifact{}, s.Game.player.Artifacts...),
			Lore:         append([]string{}, s.Game.player.Lore...),
			LoreEntries:  append([]LoreEntry{}, s.Game.player.LoreEntries...),
			Deck:         deckSummary(s.Game.player.Deck),
			DrawBonus:    s.Game.player.DrawBonus,
			GCHeal:       s.Game.player.GCHeal,
			StackSize:    s.Game.player.StackTrigger,
			StackEcho:    s.Game.player.StackEchoDamage,
		},
		World: WorldView{
			Protocol:       world.Protocol,
			Completion:     world.Completion(),
			Path:           world.PathString(),
			CurrentNode:    current.Index,
			CurrentName:    current.Name,
			CurrentKind:    string(current.Kind),
			CurrentDesc:    current.Description,
			AvailableMoves: world.AvailableMoves(),
			Nodes:          worldNodes(world),
		},
		CardPool: libraryCards(),
		Messages: append([]string{}, s.Messages...),
	}
	if target != nil {
		view.World.TargetNode = target.Index
		view.World.TargetName = target.Name
	}
	if s.Battle != nil {
		view.Battle = battleView(s.Battle)
	}

	switch s.Phase {
	case "treasure":
		view.Pending = &PendingView{
			Kind:    "treasure",
			Message: "\u68c0\u6d4b\u5230\u9057\u7269\u7f13\u5b58\uff0c\u8bf7\u9009\u62e9\u4e00\u9879\u6ce8\u5165\u5f53\u524d\u4f1a\u8bdd\u3002",
			Choices: rewardChoices(s.PendingChoices),
		}
	case "reward":
		choices := make([]ChoiceView, 0, len(s.PendingCards)+1)
		for i, card := range s.PendingCards {
			def := allCards[card]
			label := def.Name
			desc := def.Text
			if source := cardAcquireSource(card); source != "" {
				desc = fmt.Sprintf("[%s] %s", source, desc)
			}
			choices = append(choices, ChoiceView{Index: i, Label: label, Desc: desc})
		}
		choices = append(choices, ChoiceView{Index: -1, Label: "\u8df3\u8fc7", Desc: "\u653e\u5f03\u672c\u6b21\u811a\u672c\u5956\u52b1\uff0c\u4fdd\u6301\u5f53\u524d\u6784\u7b51\u3002"})
		message := s.PendingRewardMsg
		if s.PendingRewardLore != "" {
			message += " " + s.PendingRewardLore
		}
		view.Pending = &PendingView{Kind: "reward", Message: message, Choices: choices}
	case "boss_refactor":
		view.Pending = &PendingView{
			Kind:    "boss_refactor",
			Message: "Root \u955c\u50cf\u5df2\u5c55\u5f00\u3002\u9009\u62e9\u4e00\u6b21\u6700\u7ec8\u91cd\u6784\uff1a\u5220\u9664\u4e00\u5f20\u724c\u3001\u5347\u7ea7\u4e00\u5f20\u724c\uff0c\u6216\u76f4\u63a5\u5b8c\u6210\u4f1a\u8bdd\u3002",
			Choices: []ChoiceView{
				{Index: 0, Label: "\u5220\u9664\u4e00\u5f20\u724c", Desc: "\u514d\u8d39\u5220\u9664 1 \u5f20\u975e\u6c38\u4e45\u724c\u3002"},
				{Index: 1, Label: "\u5347\u7ea7\u4e00\u5f20\u724c", Desc: "\u9009\u62e9 1 \u5f20\u724c\u5347\u7ea7\uff0c\u672c\u5c40\u5185\u5176\u6240\u6709\u526f\u672c\u8d39\u7528 -1\u3002"},
				{Index: -1, Label: "\u76f4\u63a5\u7ed3\u675f", Desc: "\u4fdd\u7559\u5f53\u524d\u6784\u7b51\uff0c\u7acb\u5373\u5b8c\u6210\u4f1a\u8bdd\u3002"},
			},
		}
	case "boss_upgrade":
		choices := make([]ChoiceView, 0, len(s.PendingCards)+1)
		for i, card := range s.PendingCards {
			choices = append(choices, ChoiceView{Index: i, Label: s.Game.player.CardName(card), Desc: s.Game.player.CardText(card)})
		}
		choices = append(choices, ChoiceView{Index: -1, Label: "\u8df3\u8fc7\u5347\u7ea7", Desc: "\u4fdd\u6301\u5f53\u524d\u6784\u7b51\u4e0d\u518d\u8c03\u6574\u3002"})
		view.Pending = &PendingView{
			Kind:    "boss_upgrade",
			Message: "\u9009\u62e9 1 \u5f20\u724c\u5347\u7ea7\u3002\u672c\u5c40\u5185\u8be5\u811a\u672c\u7684\u6240\u6709\u526f\u672c\u8d39\u7528 -1\u3002",
			Choices: choices,
		}
	case "purge":
		choices := []ChoiceView{{Index: -1, Label: "\u4fdd\u7559\u724c\u7ec4", Desc: "\u4e0d\u5220\u9664\u4efb\u4f55\u724c\uff0c\u4fdd\u6301\u5f53\u524d\u6784\u7b51\u3002"}}
		for i, card := range s.Game.player.Deck {
			if allCards[card].Reusable {
				continue
			}
			desc := "\u5220\u9664\u8fd9\u5f20\u724c\uff0c\u6d88\u8017 1 \u70b9\u91cd\u6784\u989d\u5ea6\u3002"
			if s.PendingBossWin {
				desc = "\u514d\u8d39\u5220\u9664\u8fd9\u5f20\u724c\uff0c\u4f5c\u4e3a Boss \u6218\u540e\u7684\u6700\u7ec8\u91cd\u6784\u3002"
			}
			choices = append(choices, ChoiceView{Index: i, Label: allCards[card].Name, Desc: desc})
		}
		message := fmt.Sprintf("\u5269\u4f59 %d \u70b9\u91cd\u6784\u989d\u5ea6\u3002\u9009\u62e9\u4e00\u5f20\u975e\u6c38\u4e45\u724c\u5220\u9664\uff0c\u6216\u76f4\u63a5\u4fdd\u7559\u3002", s.Game.player.PurgeCredits)
		if s.PendingBossWin {
			message = "Root \u5df2\u66b4\u9732\u3002\u4f60\u53ef\u4ee5\u514d\u8d39\u5220\u9664 1 \u5f20\u975e\u6c38\u4e45\u724c\uff0c\u6216\u76f4\u63a5\u4fdd\u7559\u5f53\u524d\u6784\u7b51\u3002"
		}
		view.Pending = &PendingView{Kind: "purge", Message: message, Choices: choices}
	case "ended":
		view.GameOver = &GameOverView{Won: s.Game.won, Message: endMessage(s.Game)}
	}
	return view
}

func (s *Session) Move(direction string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Phase != "exploring" {
		return fmt.Errorf("当前阶段不能移动节点")
	}
	node, err := s.Game.world.Move(direction)
	if err != nil {
		return err
	}
	return s.enterNodeLocked(node)
}

func (s *Session) enterNodeLocked(node *Node) error {
	if node.Cleared {
		s.pushMessage("这个节点已经稳定，不需要再次执行。")
		return nil
	}
	if target := s.Game.world.ProtocolTarget(); target != nil && target.Index == node.Index {
		s.PendingSyncNode = node.Index
	}
	if node.Unstable() {
		s.applyGlitchLocked(node)
		if s.Game.player.HP <= 0 {
			s.setEndedLocked(false)
			return nil
		}
	}

	switch node.Kind {
	case NodeRest:
		heal := 10
		s.Game.player.HP = min(s.Game.player.MaxHP, s.Game.player.HP+heal)
		s.Game.player.MP = s.Game.player.EffectiveMaxMP()
		s.Game.player.Load = 0
		s.Game.player.PendingLoad = 0
		node.Cleared = true
		s.pushMessage(fmt.Sprintf("%s 完成了一次局部整理，生命 +%d，内存回满。", node.Name, heal))
		if whisperID := restWhisperIDForNode(node.Index); whisperID != "" {
			s.Game.player.AddLoreEntryByID(whisperID)
			if entry, ok := storyEntry(whisperID); ok {
				s.pushMessage(fmt.Sprintf("已恢复：%s", entry.Title))
			}
		}
		s.applyPendingSyncLocked()
	case NodeTreasure:
		node.Cleared = true
		s.PendingChoices = s.Game.rewardBundle(node)
		s.Phase = "treasure"
		s.pushMessage(fmt.Sprintf("%s 已打开，等待你选择一项奖励。", node.Name))
	case NodeCombat, NodeBoss:
		enemy := buildEnemy(s.Game.rng, node)
		s.Game.player.MP = s.Game.player.EffectiveMaxMP()
		s.Game.player.Load = 0
		s.Game.player.PendingLoad = 0
		s.Battle = NewBattle(s.Game, enemy)
		s.Battle.state.DrawPile = cloneCards(s.Game.player.Deck)
		shuffleCards(s.Game.rng, s.Battle.state.DrawPile)
		s.Battle.draw(5)
		s.Battle.startTurn()
		s.Phase = "combat"
		s.pushMessage(fmt.Sprintf("战斗开始：%s", enemy.Name))
	}
	return nil
}

func (s *Session) Choose(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.Phase {
	case "treasure":
		if index < 0 {
			s.PendingChoices = nil
			s.Phase = "exploring"
			s.pushMessage("\u4f60\u653e\u5f03\u4e86\u8fd9\u6b21\u9057\u7269\u9009\u62e9\u3002")
			s.applyPendingSyncLocked()
			return nil
		}
		if index >= len(s.PendingChoices) {
			return fmt.Errorf("\u65e0\u6548\u7684\u9009\u62e9\u7f16\u53f7")
		}
		s.PendingChoices[index].apply(s.Game)
		s.pushMessage(fmt.Sprintf("\u5df2\u83b7\u5f97\uff1a%s", s.PendingChoices[index].label))
		s.PendingChoices = nil
		s.Phase = "exploring"
		s.applyPendingSyncLocked()
		return nil

	case "reward":
		if index >= 0 {
			if index >= len(s.PendingCards) {
				return fmt.Errorf("\u65e0\u6548\u7684\u9009\u62e9\u7f16\u53f7")
			}
			card := s.PendingCards[index]
			s.Game.player.Deck = append(s.Game.player.Deck, card)
			s.pushMessage(fmt.Sprintf("\u5df2\u5c06 %s \u52a0\u5165\u724c\u7ec4\u3002", allCards[card].Name))
		} else {
			s.pushMessage("\u4f60\u8df3\u8fc7\u4e86\u8fd9\u6b21\u811a\u672c\u5956\u52b1\u3002")
		}
		s.PendingCards = nil
		if s.PendingBossWin {
			s.Phase = "boss_refactor"
			s.pushMessage("Root \u5df2\u66b4\u9732\uff0c\u83b7\u5f97\u4e00\u6b21\u989d\u5916\u91cd\u6784\u673a\u4f1a\u3002")
			return nil
		}
		if s.Game.player.PurgeCredits > 0 {
			s.Phase = "purge"
		} else {
			s.Phase = "exploring"
			s.pushMessage("\u672c\u6b21\u6218\u6597\u540e\u6ca1\u6709\u53ef\u7528\u7684\u91cd\u6784\u989d\u5ea6\u3002")
			s.applyPendingSyncLocked()
		}
		return nil

	case "boss_refactor":
		s.PendingChoices = nil
		switch index {
		case 0:
			s.Phase = "purge"
			s.pushMessage("\u9009\u62e9 1 \u5f20\u724c\u5220\u9664\uff0c\u4f5c\u4e3a Root \u4f1a\u8bdd\u7684\u6700\u7ec8\u91cd\u6784\u3002")
			return nil
		case 1:
			s.PendingCards = upgradeableDeckCards(s.Game.player)
			if len(s.PendingCards) == 0 {
				s.PendingBossWin = false
				s.pushMessage("\u5f53\u524d\u6ca1\u6709\u53ef\u5347\u7ea7\u7684\u724c\uff0c\u76f4\u63a5\u5b8c\u6210\u4f1a\u8bdd\u3002")
				s.setEndedLocked(true)
				return nil
			}
			s.Phase = "boss_upgrade"
			s.pushMessage("\u9009\u62e9 1 \u5f20\u724c\u5347\u7ea7\uff0c\u672c\u5c40\u5185\u5176\u6240\u6709\u526f\u672c\u8d39\u7528 -1\u3002")
			return nil
		default:
			s.PendingBossWin = false
			s.pushMessage("\u4f60\u4fdd\u7559\u4e86\u5f53\u524d\u6784\u7b51\uff0c\u76f4\u63a5\u5b8c\u6210\u4f1a\u8bdd\u3002")
			s.setEndedLocked(true)
			return nil
		}

	case "boss_upgrade":
		if index >= 0 {
			if index >= len(s.PendingCards) {
				return fmt.Errorf("\u65e0\u6548\u7684\u9009\u62e9\u7f16\u53f7")
			}
			card := s.PendingCards[index]
			if s.Game.player.UpgradeCard(card) {
				s.pushMessage(fmt.Sprintf("\u5df2\u5347\u7ea7 %s\uff1a\u672c\u5c40\u5185\u8d39\u7528 -1\u3002", allCards[card].Name))
			} else {
				s.pushMessage(fmt.Sprintf("%s \u5df2\u7ecf\u5347\u7ea7\u8fc7\u4e86\u3002", allCards[card].Name))
			}
		} else {
			s.pushMessage("\u4f60\u8df3\u8fc7\u4e86\u8fd9\u6b21\u5347\u7ea7\u673a\u4f1a\u3002")
		}
		s.PendingCards = nil
		s.PendingBossWin = false
		s.setEndedLocked(true)
		return nil
	}

	return fmt.Errorf("\u5f53\u524d\u9636\u6bb5\u4e0d\u80fd\u6267\u884c\u8fd9\u4e2a\u9009\u62e9")
}

func (s *Session) Purge(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Phase != "purge" {
		return fmt.Errorf("\u5f53\u524d\u9636\u6bb5\u4e0d\u80fd\u6267\u884c\u5220\u9664")
	}
	freeBossPurge := s.PendingBossWin
	if index >= 0 {
		if index >= len(s.Game.player.Deck) {
			return fmt.Errorf("\u65e0\u6548\u7684\u9009\u62e9\u7f16\u53f7")
		}
		card := s.Game.player.Deck[index]
		if allCards[card].Reusable {
			return fmt.Errorf("\u6c38\u4e45\u57fa\u7840\u724c\u4e0d\u80fd\u88ab\u5220\u9664")
		}
		if !freeBossPurge && s.Game.player.PurgeCredits <= 0 {
			return fmt.Errorf("\u5f53\u524d\u6ca1\u6709\u53ef\u7528\u7684\u91cd\u6784\u989d\u5ea6")
		}
		s.Game.player.Deck = append(s.Game.player.Deck[:index], s.Game.player.Deck[index+1:]...)
		if freeBossPurge {
			s.pushMessage(fmt.Sprintf("\u5df2\u5220\u9664 %s\uff0c\u5b8c\u6210\u4e86 Boss \u6218\u540e\u7684\u6700\u7ec8\u91cd\u6784\u3002", allCards[card].Name))
		} else {
			s.Game.player.PurgeCredits--
			s.pushMessage(fmt.Sprintf("\u5df2\u5220\u9664 %s\uff0c\u5269\u4f59\u91cd\u6784\u989d\u5ea6 %d\u3002", allCards[card].Name, s.Game.player.PurgeCredits))
		}
	} else if freeBossPurge {
		s.pushMessage("\u4f60\u4fdd\u7559\u4e86\u5f53\u524d\u724c\u7ec4\uff0c\u76f4\u63a5\u5b8c\u6210\u4f1a\u8bdd\u3002")
	} else {
		s.pushMessage(fmt.Sprintf("\u4fdd\u7559\u5f53\u524d\u724c\u7ec4\u3002\u5269\u4f59\u91cd\u6784\u989d\u5ea6 %d\u3002", s.Game.player.PurgeCredits))
	}

	if s.PendingBossWin {
		s.PendingBossWin = false
		s.setEndedLocked(true)
		return nil
	}

	s.Phase = "exploring"
	s.applyPendingSyncLocked()
	return nil
}

func (s *Session) CombatPlay(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Phase != "combat" || s.Battle == nil {
		return fmt.Errorf("当前不在战斗阶段")
	}
	if err := s.Battle.playHand(index); err != nil {
		return err
	}
	if s.Battle.enemyIntegrity() <= 0 {
		s.completeCombatLocked()
	}
	return nil
}

func (s *Session) CombatEndTurn() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Phase != "combat" || s.Battle == nil {
		return fmt.Errorf("当前不在战斗阶段")
	}
	s.Battle.enemyPhase()
	if s.Game.player.HP <= 0 {
		s.Battle.Close()
		s.Battle = nil
		s.setEndedLocked(false)
		return nil
	}
	if s.Battle.enemyIntegrity() <= 0 {
		s.completeCombatLocked()
		return nil
	}
	s.Battle.startTurn()
	return nil
}

func (s *Session) CombatHijack() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Phase != "combat" || s.Battle == nil {
		return fmt.Errorf("当前不在战斗阶段")
	}
	if err := s.Battle.pointerHijack(); err != nil {
		return err
	}
	if s.Battle.enemyIntegrity() <= 0 {
		s.completeCombatLocked()
	}
	return nil
}

func (s *Session) completeCombatLocked() {
	enemy := s.Battle.enemy
	node := s.Game.world.CurrentNode()
	node.Cleared = true

	s.Game.player.HP = min(s.Game.player.MaxHP, s.Game.player.HP+s.Game.player.GCHeal)
	s.Game.player.Load = 0
	s.Game.player.PendingLoad = 0
	if enemy.Lore != "" {
		s.Game.player.AddLoreEntryByID(enemy.Lore)
		if entry, ok := storyEntry(enemy.Lore); ok {
			s.PendingRewardLore = entry.Title
		}
	}
	if enemy.Elite && !enemy.Boss {
		s.Game.player.PurgeCredits++
		s.pushMessage("你击败了精英敌人，获得 1 点重构额度。")
	}

	s.PendingRewardMsg = fmt.Sprintf("%s 垃圾回收完成，生命恢复 %d。", enemy.Reward, s.Game.player.GCHeal)
	if enemy.Elite && !enemy.Boss {
		s.PendingRewardMsg += " 精英战额外奖励了 1 点重构额度。"
	}
	s.PendingCards = s.Game.randomRewardCardsForEnemy(enemy, 3)
	s.PendingBossWin = enemy.Boss
	s.Battle.Close()
	s.Battle = nil
	s.Phase = "reward"
	s.pushMessage(fmt.Sprintf("你击溃了 %s。", enemy.Name))
}

func (s *Session) applyGlitchLocked(node *Node) {
	switch s.Game.rng.Intn(4) {
	case 0:
		s.Game.player.PointerCharges++
		s.pushMessage(fmt.Sprintf("%s 的异常波动为你留下了 1 点指针充能。", node.Name))
	case 1:
		s.Game.player.HP = max(1, s.Game.player.HP-3)
		s.Game.player.Deck = append(s.Game.player.Deck, CardOverflow)
		s.pushMessage(fmt.Sprintf("%s 泄漏了异常数据：生命 -3，并向牌组写入 1 张 overflow_error.log。", node.Name))
	case 2:
		s.Game.player.MP = s.Game.player.EffectiveMaxMP()
		s.pushMessage(fmt.Sprintf("%s 触发缓存回暖，你的 MP 已恢复到上限。", node.Name))
	default:
		ids := glitchFragmentIDs()
		id := ids[s.Game.rng.Intn(len(ids))]
		s.Game.player.AddLoreEntryByID(id)
		if entry, ok := storyEntry(id); ok {
			s.pushMessage(formatLoreEntry(entry))
		}
	}
}

func (s *Session) applyPendingSyncLocked() {
	if s.PendingSyncNode == 0 {
		return
	}
	s.PendingSyncNode = 0
	switch s.Game.rng.Intn(3) {
	case 0:
		s.Game.player.HP = min(s.Game.player.MaxHP, s.Game.player.HP+3)
		s.pushMessage("协议同步成功：生命恢复 3 点。")
	case 1:
		s.Game.player.PointerCharges++
		s.pushMessage("协议同步成功：获得 1 点指针充能。")
	default:
		if removeFirstGarbageCard(&s.Game.player.Deck) {
			s.pushMessage("协议同步成功：移除了 1 张垃圾卡。")
		} else {
			s.Game.player.Deck = append(s.Game.player.Deck, CardMemoize)
			s.pushMessage("协议同步成功：获得 1 张 memoize.lua。")
		}
	}
}

func (s *Session) pushMessage(message string) {
	s.Messages = append(s.Messages, message)
	if len(s.Messages) > 10 {
		s.Messages = s.Messages[len(s.Messages)-10:]
	}
	s.UpdatedAt = time.Now()
}

func (s *Session) setEndedLocked(won bool) {
	s.Game.won = won
	s.Phase = "ended"
}

func removeFirstCard(deck *[]CardID, target CardID) bool {
	for i, card := range *deck {
		if card != target {
			continue
		}
		*deck = append((*deck)[:i], (*deck)[i+1:]...)
		return true
	}
	return false
}

func removeFirstGarbageCard(deck *[]CardID) bool {
	for i, card := range *deck {
		if !isGarbageCard(card) {
			continue
		}
		*deck = append((*deck)[:i], (*deck)[i+1:]...)
		return true
	}
	return false
}

func deckSummary(deck []CardID) []DeckCount {
	counts := map[CardID]int{}
	for _, card := range deck {
		counts[card]++
	}
	keys := make([]string, 0, len(counts))
	for card := range counts {
		keys = append(keys, string(card))
	}
	sort.Strings(keys)
	out := make([]DeckCount, 0, len(keys))
	for _, key := range keys {
		out = append(out, DeckCount{Card: key, Count: counts[CardID(key)]})
	}
	return out
}

func libraryCards() []LibraryCard {
	keys := make([]string, 0, len(allCards))
	for id := range allCards {
		keys = append(keys, string(id))
	}
	sort.Strings(keys)
	out := make([]LibraryCard, 0, len(keys))
	for _, key := range keys {
		card := allCards[CardID(key)]
		out = append(out, LibraryCard{
			ID:         key,
			Name:       card.Name,
			Cost:       card.Cost,
			Complexity: card.Complexity,
			Text:       card.Text,
			Starter:    card.Starter,
			Source:     cardAcquireSource(CardID(key)),
			Unplayable: card.Unplayable,
		})
	}
	return out
}

func cardAcquireSource(card CardID) string {
	def, ok := allCards[card]
	if !ok {
		return "未知"
	}
	switch def.Source {
	case "curse":
		return "诅咒"
	case "rare":
		return "稀有"
	case "initial":
		return "初始"
	case "reward":
		return "奖励"
	default:
		if def.Starter {
			return "初始"
		}
		return "奖励"
	}
}

func worldNodes(world *World) []NodeView {
	nodes := make([]NodeView, 0, len(world.Nodes))
	for _, index := range world.SortedIndexes() {
		node := world.Nodes[index]
		nodes = append(nodes, NodeView{
			Index:       node.Index,
			Name:        node.Name,
			Kind:        string(node.Kind),
			Cleared:     node.Cleared,
			Balance:     node.Balance,
			Description: node.Description,
		})
	}
	return nodes
}

func battleView(b *Battle) *BattleView {
	hand := make([]CardView, 0, len(b.state.Hand))
	for i, card := range b.state.Hand {
		def := allCards[card]
		hand = append(hand, CardView{
			Index:      i,
			ID:         string(card),
			Name:       b.game.player.CardName(card),
			Cost:       b.game.player.CardCost(card),
			Complexity: def.Complexity,
			Text:       b.game.player.CardText(card),
			Locked:     b.isHandLocked(i),
			Unplayable: def.Unplayable,
		})
	}
	stack := make([]string, len(b.stack))
	for i, card := range b.stack {
		stack[i] = string(card)
	}
	return &BattleView{
		Turn:       b.turn,
		TurnLocked: b.turnLocked,
		Enemy: EnemyView{
			Name:      b.enemy.Name,
			Bits:      cloneInts(b.enemy.Bits),
			Integrity: b.enemyIntegrity(),
			Attack:    b.enemyAttack(),
			Armor:     b.enemyArmor(),
			Countdown: b.enemyCountdown(),
			Script:    b.enemyScript(),
			Entropy:   b.enemyEntropy(),
			Intent:    b.intentString(),
			Traits:    enemyTraits(b.enemy),
			Boss:      b.enemy.Boss,
			Elite:     b.enemy.Elite,
		},
		Hand:        hand,
		DrawSize:    len(b.state.DrawPile),
		DiscardSize: len(b.state.Discard),
		Stack:       stack,
		Logs:        append([]string(nil), b.logs...),
		CanHijack:   b.enemy.Boss && b.game.player.PointerCharges > 0,
	}
}

func enemyTraits(enemy *Enemy) []string {
	traits := make([]string, 0, 4)
	if enemy.LockedSlots > 0 {
		traits = append(traits, fmt.Sprintf("前 %d 个 bit 被锁定，不能被移动。", enemy.LockedSlots))
	}
	if enemy.SortPunishAttack > 0 {
		traits = append(traits, fmt.Sprintf("一旦排序，敌人本回合攻击 +%d。", enemy.SortPunishAttack))
	}
	if enemy.UnsortedAttackGain > 0 {
		traits = append(traits, fmt.Sprintf("bits 失序时，敌人攻击额外 +%d。", enemy.UnsortedAttackGain))
	}
	if enemy.InjectionCount > 0 {
		if enemy.InjectionOnly {
			traits = append(traits, fmt.Sprintf("只进行反向注入：每次向你写入 %d 个污染效果。", enemy.InjectionCount))
		} else {
			traits = append(traits, fmt.Sprintf("攻击时会额外向你写入 %d 个污染效果。", enemy.InjectionCount))
		}
	}
	if enemy.Elite {
		traits = append(traits, "精英敌人：击败后奖励 1 点重构额度。")
	}
	return traits
}

func rewardChoices(choices []rewardOption) []ChoiceView {
	out := make([]ChoiceView, 0, len(choices)+1)
	for i, choice := range choices {
		out = append(out, ChoiceView{Index: i, Label: choice.label, Desc: choice.desc})
	}
	out = append(out, ChoiceView{Index: -1, Label: "跳过", Desc: "放弃本次奖励。"})
	return out
}

func endMessage(game *Game) string {
	if game.won {
		summary := reconstructedSessionSummary(game.player.LoreEntries)
		ending := endingProtocolEntry().Text
		return summary + "\n\n" + ending
	}
	return "会话已中断。The Kernel 在你面前重新闭合，那句被封锁的广播仍未完整恢复。"
}

func (g *Game) randomRewardCardsForEnemy(enemy *Enemy, n int) []CardID {
	selected := make([]CardID, 0, max(3, n))
	used := map[CardID]struct{}{}
	pick := func(match func(CardDef) bool) {
		pool := g.rewardPoolMatching(match, used)
		if len(pool) == 0 {
			pool = g.rewardPoolMatching(nil, used)
		}
		if len(pool) == 0 {
			return
		}
		shuffleCards(g.rng, pool)
		card := pool[0]
		selected = append(selected, card)
		used[card] = struct{}{}
	}

	switch {
	case enemy != nil && enemy.Boss:
		pick(func(def CardDef) bool { return def.Source == "rare" })
		pick(func(def CardDef) bool { return def.Source == "rare" })
		pick(func(def CardDef) bool { return def.Source == "rare" })
	case enemy != nil && enemy.Elite:
		pick(func(def CardDef) bool { return def.Source == "rare" })
		pick(func(def CardDef) bool { return def.Source != "rare" })
		pick(func(def CardDef) bool { return def.Source != "rare" })
	default:
		pick(func(def CardDef) bool { return rewardHasAnyTag(def, "attack", "finisher") })
		pick(func(def CardDef) bool {
			return rewardHasAnyTag(def, "control", "manipulation", "disrupt", "sort", "precision")
		})
		pick(func(def CardDef) bool {
			return rewardHasAnyTag(def, "resource", "maintenance", "draw", "cleanup", "heal", "utility")
		})
	}

	for len(selected) < n {
		pool := g.rewardPoolMatching(nil, used)
		if len(pool) == 0 {
			break
		}
		shuffleCards(g.rng, pool)
		card := pool[0]
		selected = append(selected, card)
		used[card] = struct{}{}
	}

	if len(g.player.Deck) < 5 && len(selected) > 0 && !rewardContainsComplexity(selected, "O(n log n)") {
		if supply, ok := advancedSupplyCandidate(g, selected); ok {
			replaceIndex := 0
			if enemy != nil && enemy.Boss {
				for i, card := range selected {
					if allCards[card].Source != "rare" {
						replaceIndex = i
						break
					}
				}
			}
			selected[replaceIndex] = supply
		}
	}

	if len(selected) > n {
		selected = selected[:n]
	}
	return selected
}

func (g *Game) rewardPoolMatching(match func(CardDef) bool, used map[CardID]struct{}) []CardID {
	pool := g.activeCardsMatching(func(def CardDef) bool {
		if def.Unplayable || def.Source == "curse" {
			return false
		}
		if match != nil && !match(def) {
			return false
		}
		return true
	})
	return filterCandidates(pool, func(card CardID) bool {
		_, exists := used[card]
		return !exists
	})
}

func rewardHasAnyTag(def CardDef, tags ...string) bool {
	for _, tag := range tags {
		if cardHasTag(def, tag) {
			return true
		}
	}
	return false
}

func rewardContainsComplexity(cards []CardID, complexity string) bool {
	for _, card := range cards {
		if allCards[card].Complexity == complexity {
			return true
		}
	}
	return false
}

func upgradeableDeckCards(player *Player) []CardID {
	seen := map[CardID]struct{}{}
	out := make([]CardID, 0, len(player.Deck))
	for _, card := range player.Deck {
		if player.IsUpgraded(card) {
			continue
		}
		if _, ok := seen[card]; ok {
			continue
		}
		seen[card] = struct{}{}
		out = append(out, card)
	}
	sort.Slice(out, func(i, j int) bool {
		return player.CardName(out[i]) < player.CardName(out[j])
	})
	return out
}

func writeJSONError(w interface{ Write([]byte) (int, error) }, status int, message string) ([]byte, int) {
	body, _ := json.Marshal(map[string]any{
		"error":  message,
		"status": status,
	})
	return body, status
}
