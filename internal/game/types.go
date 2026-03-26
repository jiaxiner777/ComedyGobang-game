package game

import (
	"bufio"
	"io"
	"math/rand"
	"strings"
)

type CardID string

const (
	CardReverse     CardID = "reverse_list.py"
	CardBubble      CardID = "bubble_sort.sh"
	CardBinary      CardID = "binary_search.go"
	CardQuick       CardID = "quick_sort.rs"
	CardBacktrack   CardID = "backtrack.ts"
	CardHeap        CardID = "heap_alloc.cpp"
	CardNullPointer CardID = "null_pointer.exe"
	CardMerge       CardID = "merge_patch.diff"
	CardRadix       CardID = "radix_sort.java"
	CardXor         CardID = "xor_mask.asm"
	CardMemoize     CardID = "memoize.lua"
	CardChecksum    CardID = "checksum.py"
	CardPivot       CardID = "pivot_swap.rb"
	CardDefrag      CardID = "defrag_gc.cs"
	CardGreedy      CardID = "greedy_path.kt"
	CardMemoryLeak  CardID = "memory_leak.bin"
	CardOverflow    CardID = "overflow_error.log"
	CardNoop        CardID = "noop.log"
)

type NodeKind string

const (
	NodeStart    NodeKind = "start"
	NodeCombat   NodeKind = "combat"
	NodeRest     NodeKind = "rest"
	NodeTreasure NodeKind = "treasure"
	NodeBoss     NodeKind = "boss"
)

type Game struct {
	rng           *rand.Rand
	reader        *bufio.Reader
	out           io.Writer
	player        *Player
	world         *World
	activePool    []CardID
	activePoolSet map[CardID]struct{}
	won           bool
	quit          bool
	headless      bool
}

type Player struct {
	HP              int
	MaxHP           int
	MP              int
	MaxMP           int
	Load            int
	PendingLoad     int
	PurgeCredits    int
	Deck            []CardID
	Artifacts       []Artifact
	Lore            []string
	LoreEntries     []LoreEntry
	PointerCharges  int
	StartMPBonus    int
	DrawBonus       int
	GCHeal          int
	StackTrigger    int
	StackEchoDamage int
	UpgradedCards   map[CardID]bool
}

type Artifact struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type World struct {
	Nodes    map[int]*Node
	Current  int
	Final    int
	Protocol string
}

type Node struct {
	Index       int
	Name        string
	Kind        NodeKind
	Cleared     bool
	Balance     int
	Description string
}

type Enemy struct {
	Name               string
	Bits               []int
	Boss               bool
	Elite              bool
	Tier               int
	Reward             string
	Lore               string
	LockedSlots        int
	SortPunishAttack   int
	UnsortedAttackGain int
	InjectionCount     int
	InjectionOnly      bool
	SimpleAttackOnly   bool
	ObfuscatePlayer    bool
	DoSLockHand        bool
	IgnoreLoadPenalty  bool
	DifficultyScore    int
}

type CombatState struct {
	Hand      []CardID
	HandLocks []bool
	DrawPile  []CardID
	Discard   []CardID
}

type BattleSnapshot struct {
	Turn           int
	PlayerHP       int
	PlayerMP       int
	PlayerLoad     int
	PendingLoad    int
	PointerCharges int
	TurnLocked     bool
	EnemyBits      []int
	Hand           []CardID
	HandLocks      []bool
	DrawPile       []CardID
	Discard        []CardID
}

type Battle struct {
	game       *Game
	enemy      *Enemy
	state      CombatState
	turn       int
	snapshots  []BattleSnapshot
	stack      []CardID
	lastCard   CardID
	queue      chan BattleCommand
	logs       []string
	turnLocked bool
}

type BattleCommand struct {
	Card   CardID
	Replay bool
	Reply  chan CommandResult
}

type CommandResult struct {
	Logs      []string
	Animation [][]int
	Err       error
}

type CardDef struct {
	ID          CardID
	Name        string
	Cost        int
	Complexity  string
	Text        string
	Starter     bool
	Reusable    bool
	Unplayable  bool
	Source      string
	Tags        []string
	Excludes    []CardID
	MutexGroups []string
	Enabled     bool
	Play        func(*Battle, bool) CommandResult
}

func cloneInts(src []int) []int {
	dst := make([]int, len(src))
	copy(dst, src)
	return dst
}

func cloneCards(src []CardID) []CardID {
	dst := make([]CardID, len(src))
	copy(dst, src)
	return dst
}

func NewGame(rng *rand.Rand, in io.Reader, out io.Writer) *Game {
	mustConfigureCardLibrary()
	game := &Game{
		rng:    rng,
		reader: bufio.NewReader(in),
		out:    out,
		world:  NewWorld(rng),
	}
	game.activePool = buildActivePool(rng)
	game.activePoolSet = make(map[CardID]struct{}, len(game.activePool))
	for _, card := range game.activePool {
		game.activePoolSet[card] = struct{}{}
	}
	game.player = NewPlayer(game.buildStarterDeck())
	return game
}

func NewHeadlessGame(rng *rand.Rand) *Game {
	game := NewGame(rng, strings.NewReader(""), io.Discard)
	game.headless = true
	return game
}

func NewPlayer(startDeck []CardID) *Player {
	if len(startDeck) == 0 {
		startDeck = defaultStarterDeck()
	}
	return &Player{
		HP:              36,
		MaxHP:           36,
		MaxMP:           6,
		MP:              6,
		Deck:            cloneCards(startDeck),
		Load:            0,
		PendingLoad:     0,
		PurgeCredits:    0,
		PointerCharges:  0,
		DrawBonus:       0,
		GCHeal:          4,
		StackTrigger:    4,
		StackEchoDamage: 2,
		UpgradedCards:   map[CardID]bool{},
	}
}

func (p *Player) EffectiveMaxMP() int {
	return p.MaxMP + p.StartMPBonus
}

func (p *Player) IsUpgraded(card CardID) bool {
	if p == nil || len(p.UpgradedCards) == 0 {
		return false
	}
	return p.UpgradedCards[card]
}

func (p *Player) CardCost(card CardID) int {
	def, ok := allCards[card]
	if !ok {
		return 0
	}
	cost := def.Cost
	if p.IsUpgraded(card) && cost > 0 {
		cost--
	}
	return cost
}

func (p *Player) CardName(card CardID) string {
	def, ok := allCards[card]
	if !ok {
		return string(card)
	}
	if p.IsUpgraded(card) {
		return def.Name + " +"
	}
	return def.Name
}

func (p *Player) CardText(card CardID) string {
	def, ok := allCards[card]
	if !ok {
		return ""
	}
	if p.IsUpgraded(card) {
		return def.Text + " ?????????? -1?"
	}
	return def.Text
}

func (p *Player) UpgradeCard(card CardID) bool {
	if p == nil {
		return false
	}
	if p.UpgradedCards == nil {
		p.UpgradedCards = map[CardID]bool{}
	}
	if p.UpgradedCards[card] {
		return false
	}
	p.UpgradedCards[card] = true
	return true
}

func (p *Player) AddArtifact(artifact Artifact) {
	p.Artifacts = append(p.Artifacts, artifact)
}

func (p *Player) AddLore(fragment string) {
	if fragment == "" {
		return
	}
	p.Lore = append(p.Lore, fragment)
}

func (p *Player) AddLoreEntry(entry LoreEntry) {
	if entry.ID == "" && entry.Text == "" {
		return
	}
	for _, existing := range p.LoreEntries {
		if entry.ID != "" && existing.ID == entry.ID {
			return
		}
	}
	p.LoreEntries = append(p.LoreEntries, entry)
	p.Lore = append(p.Lore, formatLoreEntry(entry))
}

func (p *Player) AddLoreEntryByID(id string) {
	if entry, ok := storyEntry(id); ok {
		p.AddLoreEntry(entry)
	}
}
