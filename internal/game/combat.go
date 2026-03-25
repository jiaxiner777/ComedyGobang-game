package game

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

func NewBattle(game *Game, enemy *Enemy) *Battle {
	b := &Battle{
		game:  game,
		enemy: enemy,
		queue: make(chan BattleCommand),
		logs: []string{
			fmt.Sprintf("接入战斗目标：%s", enemy.Name),
			fmt.Sprintf("Raw bits detected: %v", enemy.Bits),
		},
	}
	go b.commandProcessor()
	return b
}

func (b *Battle) Close() {
	close(b.queue)
}

func (b *Battle) commandProcessor() {
	for cmd := range b.queue {
		def := allCards[cmd.Card]
		cmd.Reply <- def.Play(b, cmd.Replay)
	}
}

func (g *Game) runCombat(enemy *Enemy) bool {
	battle := NewBattle(g, enemy)
	defer battle.Close()

	g.player.MP = g.player.EffectiveMaxMP()
	g.player.Load = 0
	g.player.PendingLoad = 0
	battle.state.DrawPile = cloneCards(g.player.Deck)
	shuffleCards(g.rng, battle.state.DrawPile)
	battle.draw(5)
	battle.startTurn()

	for g.player.HP > 0 && battle.enemyIntegrity() > 0 {
		battle.render("")
		input := strings.ToLower(g.readLine("play [1-n], end [e], hijack [p], help [h] > "))
		if input == "" {
			continue
		}
		switch input {
		case "e":
			battle.enemyPhase()
			if battle.enemyIntegrity() > 0 && g.player.HP > 0 {
				battle.startTurn()
			}
		case "p":
			if err := battle.pointerHijack(); err != nil {
				battle.pushLog(err.Error())
			}
		case "h":
			battle.renderHelp()
		default:
			index, err := strconv.Atoi(input)
			if err != nil {
				battle.pushLog("请输入有效的手牌编号。")
				continue
			}
			if err := battle.playHand(index - 1); err != nil {
				battle.pushLog(err.Error())
			}
		}
	}

	if g.player.HP <= 0 {
		return false
	}

	signalBell(g.out)
	battle.render(fmt.Sprintf("%s 已被击溃。", enemy.Name))
	g.readLine("press enter > ")
	return true
}

func (b *Battle) startTurn() {
	b.turn++
	b.stack = nil
	b.turnLocked = false

	if b.introProtectionActive() {
		b.game.player.Load = 0
		b.game.player.PendingLoad = 0
		b.game.player.MP = min(b.game.player.EffectiveMaxMP(), b.game.player.MP+4)
		b.draw(2 + b.game.player.DrawBonus)
		b.saveSnapshot()
		b.pushLog(fmt.Sprintf("Turn %d start -> Intro Shield active | MP %d", b.turn, b.game.player.MP))
		return
	}

	b.game.player.Load = max(0, b.game.player.Load-1) + b.game.player.PendingLoad
	b.game.player.PendingLoad = 0

	regen := 4
	drawCount := 2 + b.game.player.DrawBonus
	switch {
	case b.game.player.Load >= 7:
		regen = 1
		drawCount = 0
		b.turnLocked = true
		b.pushLog("System Load 爆表，本回合脚本调度冻结。")
	case b.game.player.Load >= 5:
		regen = 2
		drawCount = max(0, drawCount-2)
		b.pushLog("System Load 过高，本回合 MP 回复和抽牌都被压制。")
	case b.game.player.Load >= 3:
		regen = 3
		drawCount = max(0, drawCount-1)
		b.pushLog("System Load 偏高，本回合抽牌数减少。")
	}

	b.game.player.MP = min(b.game.player.EffectiveMaxMP(), b.game.player.MP+regen)
	b.draw(drawCount)

	leaks := b.countHandCard(CardMemoryLeak)
	if leaks > 0 {
		b.game.player.PendingLoad += leaks
		b.pushLog(fmt.Sprintf("memory_leak.bin 持续占用内存，本回合结束后将追加 %d 点负载。", leaks))
	}

	b.saveSnapshot()
	b.pushLog(fmt.Sprintf("Turn %d start -> MP %d | Load %d (+%d pending)", b.turn, b.game.player.MP, b.game.player.Load, b.game.player.PendingLoad))
}

func (b *Battle) saveSnapshot() {
	b.snapshots = append(b.snapshots, BattleSnapshot{
		Turn:           b.turn,
		PlayerHP:       b.game.player.HP,
		PlayerMP:       b.game.player.MP,
		PlayerLoad:     b.game.player.Load,
		PendingLoad:    b.game.player.PendingLoad,
		PointerCharges: b.game.player.PointerCharges,
		TurnLocked:     b.turnLocked,
		EnemyBits:      cloneInts(b.enemy.Bits),
		Hand:           cloneCards(b.state.Hand),
		HandLocks:      append([]bool(nil), b.state.HandLocks...),
		DrawPile:       cloneCards(b.state.DrawPile),
		Discard:        cloneCards(b.state.Discard),
	})
}

func (b *Battle) restorePreviousSnapshot() bool {
	if len(b.snapshots) < 2 {
		return false
	}
	snap := b.snapshots[len(b.snapshots)-2]
	b.turn = snap.Turn
	b.game.player.HP = snap.PlayerHP
	b.game.player.MP = snap.PlayerMP
	b.game.player.Load = snap.PlayerLoad
	b.game.player.PendingLoad = snap.PendingLoad
	b.game.player.PointerCharges = snap.PointerCharges
	b.turnLocked = snap.TurnLocked
	b.enemy.Bits = cloneInts(snap.EnemyBits)
	b.state.Hand = cloneCards(snap.Hand)
	b.state.HandLocks = append([]bool(nil), snap.HandLocks...)
	b.state.DrawPile = cloneCards(snap.DrawPile)
	b.state.Discard = cloneCards(snap.Discard)
	b.stack = nil
	b.snapshots = b.snapshots[:len(b.snapshots)-1]
	return true
}

func (b *Battle) restoreSystemSnapshot() bool {
	if len(b.snapshots) < 2 {
		return false
	}
	snap := b.snapshots[len(b.snapshots)-2]
	b.turn = snap.Turn
	b.game.player.HP = snap.PlayerHP
	b.game.player.MP = snap.PlayerMP
	b.game.player.Load = snap.PlayerLoad
	b.game.player.PendingLoad = snap.PendingLoad
	b.turnLocked = snap.TurnLocked
	b.enemy.Bits = cloneInts(snap.EnemyBits)
	b.stack = nil
	b.snapshots = b.snapshots[:len(b.snapshots)-1]
	return true
}

func (b *Battle) playHand(index int) error {
	if index < 0 || index >= len(b.state.Hand) {
		return fmt.Errorf("这个手牌位置不存在")
	}
	if b.turnLocked {
		return fmt.Errorf("系统冻结中，本回合只能结束回合")
	}
	if b.isHandLocked(index) {
		return fmt.Errorf("这张手牌已被系统拒绝服务锁定，本回合无法执行")
	}

	card := b.state.Hand[index]
	def := allCards[card]
	cost := b.game.player.CardCost(card)
	if def.Unplayable {
		return fmt.Errorf("%s 是垃圾卡，不能直接执行", def.Name)
	}
	if b.game.player.MP < cost {
		return fmt.Errorf("MP ???%s ?? %d ? MP", b.game.player.CardName(card), cost)
	}

	b.game.player.MP -= cost
	b.removeHandCard(index)
	reply := make(chan CommandResult, 1)
	b.queue <- BattleCommand{Card: card, Reply: reply}
	result := <-reply
	if result.Err != nil {
		b.game.player.MP += cost
		b.state.Hand = append(b.state.Hand, card)
		b.state.HandLocks = append(b.state.HandLocks, false)
		return result.Err
	}

	b.lastCard = card
	b.state.Discard = append(b.state.Discard, card)
	if !b.introProtectionActive() {
		loadGain := complexityLoad(def.Complexity)
		if loadGain > 0 {
			b.game.player.PendingLoad += loadGain
			b.pushLog(fmt.Sprintf("%s 让系统负载 +%d，并在下回合结算。", def.Name, loadGain))
		}
	}
	if len(result.Animation) > 0 {
		b.animate(def.Name, result.Animation)
	}
	for _, line := range result.Logs {
		b.pushLog(line)
	}
	if b.enemyIntegrity() <= 0 {
		return nil
	}

	if card != CardNoop {
		b.stack = append(b.stack, card)
	}
	if len(b.stack) >= b.game.player.StackTrigger {
		b.triggerStackOverflow()
	}
	return nil
}

func (b *Battle) triggerStackOverflow() {
	if len(b.stack) < 2 {
		return
	}
	signalBell(b.game.out)
	top := b.stack[len(b.stack)-1]
	second := b.stack[len(b.stack)-2]
	b.pushLog("Stack Overflow -> 触发 LIFO 连锁，回放最近两张脚本。")
	for _, card := range []CardID{top, second} {
		reply := make(chan CommandResult, 1)
		b.queue <- BattleCommand{Card: card, Replay: true, Reply: reply}
		result := <-reply
		if result.Err != nil {
			continue
		}
		if len(result.Animation) > 0 {
			b.animate(string(card)+" [replay]", result.Animation)
		}
		for _, line := range result.Logs {
			b.pushLog("[replay] " + line)
		}
	}
	if b.game.player.StackEchoDamage > 0 && b.enemyIntegrity() > 0 {
		done := b.damageEnemy(b.game.player.StackEchoDamage, true)
		b.pushLog(fmt.Sprintf("栈回响追加造成 %d 点直伤。", done))
	}
	b.stack = nil
}

func (b *Battle) pointerHijack() error {
	if !b.enemy.Boss {
		return fmt.Errorf("只有 Boss 战才能执行 Pointer Hijack")
	}
	if b.game.player.PointerCharges <= 0 {
		return fmt.Errorf("当前没有可用的 Root 指针充能")
	}

	before := cloneInts(b.enemy.Bits)
	b.game.player.PointerCharges--
	done := b.damageEnemy(7, true)
	if len(b.enemy.Bits) > 2 {
		b.enemy.Bits[2] = max(0, b.enemy.Bits[2]-2)
	}
	b.normalizeEnemy()
	signalBell(b.game.out)
	b.animate("pointer_hijack", [][]int{before, cloneInts(b.enemy.Bits)})
	b.pushLog(fmt.Sprintf("Pointer Hijack -> 直接改写 Boss 底层数值，造成 %d 点伤害。", done))
	return nil
}

func (b *Battle) enemyPhase() {
	if b.enemyIntegrity() <= 0 {
		return
	}
	b.clearHandLocks()
	if b.enemyCountdown() > 0 && len(b.enemy.Bits) > 3 {
		b.enemy.Bits[3]--
	}

	if !b.enemy.SimpleAttackOnly && b.enemyEntropy() > 0 && b.game.rng.Intn(3) == 0 {
		switch b.game.rng.Intn(3) {
		case 0:
			if len(b.enemy.Bits) > 2 {
				b.enemy.Bits[2] = min(16, b.enemy.Bits[2]+1)
			}
			b.pushLog(fmt.Sprintf("%s 临时修补了 1 点护甲。", b.enemy.Name))
		case 1:
			b.enemy.Bits[0] = min(99, b.enemy.Bits[0]+2)
			b.pushLog(fmt.Sprintf("%s 从混沌中恢复了 2 点完整度。", b.enemy.Name))
		default:
			if len(b.enemy.Bits) > 1 {
				b.enemy.Bits[1] = min(24, b.enemy.Bits[1]+1)
			}
			b.pushLog(fmt.Sprintf("%s 编译出了更强的攻击签名。", b.enemy.Name))
		}
	}

	if b.enemyCountdown() <= 0 {
		if b.enemy.InjectionOnly {
			b.pushLog(fmt.Sprintf("%s 放弃直接伤害，转而发动反向注入。", b.enemy.Name))
			for _, line := range b.injectPlayerCorruption(max(1, b.enemy.InjectionCount)) {
				b.pushLog(line)
			}
		} else {
			damage := max(1, b.enemyAttack())
			if !b.enemy.SimpleAttackOnly {
				damage += b.enemyScript() / 4
				if b.enemyEntropy() >= 10 {
					damage += 2
					signalBell(b.game.out)
					b.pushLog("检测到敌方高熵暴走，伤害被进一步放大。")
				}
			}
			b.game.player.HP = max(0, b.game.player.HP-damage)
			b.pushLog(fmt.Sprintf("%s 造成了 %d 点伤害。", b.enemy.Name, damage))
			if b.enemy.InjectionCount > 0 {
				for _, line := range b.injectPlayerCorruption(b.enemy.InjectionCount) {
					b.pushLog(line)
				}
			}
			if b.enemy.ObfuscatePlayer {
				if line := b.flipRandomPlayerBit(); line != "" {
					b.pushLog(line)
				}
			}
		}
		if len(b.enemy.Bits) > 3 {
			b.enemy.Bits[3] = 2 + b.game.rng.Intn(3)
		}
		if b.enemy.Boss && len(b.enemy.Bits) > 2 {
			b.enemy.Bits[2] = min(16, b.enemy.Bits[2]+1)
			b.pushLog("ROOT::Sentinel 在攻击后重新加固了护甲。")
		}
		if b.enemy.DoSLockHand {
			if line := b.lockRandomHandCard(); line != "" {
				b.pushLog(line)
			}
		}
	} else {
		b.pushLog(fmt.Sprintf("%s 正在编译攻击，剩余 %d 个计时。", b.enemy.Name, b.enemyCountdown()))
	}
	b.normalizeEnemy()
}

func (b *Battle) draw(n int) {
	for i := 0; i < n; i++ {
		if len(b.state.Hand) >= 7 {
			return
		}
		if len(b.state.DrawPile) == 0 {
			if len(b.state.Discard) == 0 {
				return
			}
			b.state.DrawPile = append(b.state.DrawPile, b.state.Discard...)
			b.state.Discard = nil
			shuffleCards(b.game.rng, b.state.DrawPile)
		}
		card := b.state.DrawPile[0]
		b.state.DrawPile = b.state.DrawPile[1:]
		b.state.Hand = append(b.state.Hand, card)
		b.state.HandLocks = append(b.state.HandLocks, false)
	}
}

func (b *Battle) countHandCard(card CardID) int {
	count := 0
	for _, candidate := range b.state.Hand {
		if candidate == card {
			count++
		}
	}
	return count
}

func (b *Battle) injectPlayerCorruption(count int) []string {
	if count <= 0 || len(b.enemy.Bits) == 0 {
		return nil
	}
	logs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		value := b.enemy.Bits[b.game.rng.Intn(len(b.enemy.Bits))]
		switch value % 3 {
		case 0:
			b.injectCard(CardMemoryLeak)
			logs = append(logs, fmt.Sprintf("敌人向你的运行时注入了 memory_leak.bin（源 bit=%d）。", value))
		case 1:
			b.injectCard(CardOverflow)
			logs = append(logs, fmt.Sprintf("敌人向你的运行时注入了 overflow_error.log（源 bit=%d）。", value))
		default:
			b.game.player.PendingLoad++
			logs = append(logs, fmt.Sprintf("敌方脏数据让你的待结算负载 +1（源 bit=%d）。", value))
		}
	}
	return logs
}

func (b *Battle) injectCard(card CardID) {
	if len(b.state.Hand) < 7 {
		b.state.Hand = append(b.state.Hand, card)
		b.state.HandLocks = append(b.state.HandLocks, false)
		return
	}
	b.state.Discard = append(b.state.Discard, card)
}

func (b *Battle) purgeOneGarbage() (CardID, bool) {
	for _, zone := range []*[]CardID{&b.state.Hand, &b.state.Discard, &b.state.DrawPile} {
		for i, card := range *zone {
			if !isGarbageCard(card) {
				continue
			}
			*zone = append((*zone)[:i], (*zone)[i+1:]...)
			return card, true
		}
	}
	return "", false
}

func (b *Battle) damageEnemy(amount int, ignoreArmor bool) int {
	if amount <= 0 {
		return 0
	}
	mitigation := 0
	if !ignoreArmor {
		mitigation = b.enemyArmor()
	}
	done := max(1, amount-mitigation)
	b.enemy.Bits[0] = max(0, b.enemy.Bits[0]-done)
	return done
}

func (b *Battle) normalizeEnemy() {
	limits := []int{70, 24, 16, 6, 20, 20}
	for i := range b.enemy.Bits {
		if b.enemy.Bits[i] < 0 {
			b.enemy.Bits[i] = 0
		}
		limit := 99
		if i < len(limits) {
			limit = limits[i]
		}
		if b.enemy.Bits[i] > limit {
			b.enemy.Bits[i] = limit
		}
	}
}

func (b *Battle) enemyIntegrity() int {
	if len(b.enemy.Bits) == 0 {
		return 0
	}
	return max(0, b.enemy.Bits[0])
}

func (b *Battle) enemyAttack() int {
	if len(b.enemy.Bits) < 2 {
		return 0
	}
	attack := max(0, b.enemy.Bits[1])
	if b.enemy.UnsortedAttackGain > 0 && !b.enemyArraySorted() {
		attack += b.enemy.UnsortedAttackGain
	}
	return attack
}

func (b *Battle) enemyArmor() int {
	if len(b.enemy.Bits) < 3 {
		return 0
	}
	return max(0, b.enemy.Bits[2])
}

func (b *Battle) enemyCountdown() int {
	if len(b.enemy.Bits) < 4 {
		return 0
	}
	return max(0, b.enemy.Bits[3])
}

func (b *Battle) enemyScript() int {
	if len(b.enemy.Bits) < 5 {
		return 0
	}
	return max(0, b.enemy.Bits[4])
}

func (b *Battle) enemyEntropy() int {
	if len(b.enemy.Bits) < 6 {
		return 0
	}
	return max(0, b.enemy.Bits[5])
}

func (b *Battle) enemyArraySorted() bool {
	start := b.lockedPrefix()
	if start >= len(b.enemy.Bits) {
		return true
	}
	return sort.IntsAreSorted(b.enemy.Bits[start:])
}

func (b *Battle) lockedPrefix() int {
	return min(b.enemy.LockedSlots, len(b.enemy.Bits))
}

func (b *Battle) mutableBits() []int {
	start := b.lockedPrefix()
	return cloneInts(b.enemy.Bits[start:])
}

func (b *Battle) setMutableBits(bits []int) {
	start := b.lockedPrefix()
	next := append(cloneInts(b.enemy.Bits[:start]), bits...)
	b.enemy.Bits = next
}

func (b *Battle) applySortCountermeasure(cardName string) []string {
	if b.enemy.SortPunishAttack <= 0 {
		return nil
	}
	if len(b.enemy.Bits) > 1 {
		b.enemy.Bits[1] = min(24, b.enemy.Bits[1]+b.enemy.SortPunishAttack)
	}
	return []string{
		fmt.Sprintf("%s 触发了防篡改：排序反制使攻击 +%d。", b.enemy.Name, b.enemy.SortPunishAttack),
	}
}

func complexityLoad(complexity string) int {
	switch complexity {
	case "O(n)", "O(n/2)", "O(n log n)":
		return 1
	case "O(n*k)", "O(n^2)", "O(branch^depth)":
		return 2
	default:
		return 0
	}
}

func buildEnemy(rng *rand.Rand, node *Node) *Enemy {
	enemy := enemyTemplate(node, rng)
	applyDifficultyScaling(rng, node, enemy)
	return enemy
}

func shuffleCards(rng *rand.Rand, cards []CardID) {
	for i := len(cards) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		cards[i], cards[j] = cards[j], cards[i]
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
