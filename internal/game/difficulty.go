package game

import (
	"fmt"
	"math"
	"math/rand"
)

func enemyTemplate(node *Node, rng *rand.Rand) *Enemy {
	switch node.Index {
	case 2:
		return &Enemy{
			Name:               "Trace Walker",
			Bits:               []int{22, 6, 3, 2, 8, 5},
			Tier:               1,
			UnsortedAttackGain: 1,
			Reward:             "你截获了追踪守卫的一段内存片段。",
			Lore:               combatLoreIDForNode(2),
		}
	case 4:
		return &Enemy{
			Name:           "Fork Bomb Warden",
			Bits:           []int{28, 8, 4, 2, 10, 7},
			Tier:           2,
			Elite:          true,
			InjectionCount: 2,
			InjectionOnly:  true,
			Reward:         "递归复制体崩解后，掉出了一段可复用的异常脚本。",
			Lore:           combatLoreIDForNode(4),
		}
	case 6:
		return &Enemy{
			Name:        "AVL Rotator",
			Bits:        []int{31, 9, 5, 3, 11, 9},
			Tier:        2,
			LockedSlots: 2,
			Reward:      "平衡旋转模块掉出了一段稳定器代码。",
			Lore:        combatLoreIDForNode(6),
		}
	case 9:
		return &Enemy{
			Name:               "Diff Hound",
			Bits:               []int{27, 7, 3, 2, 12, 8},
			Tier:               2,
			UnsortedAttackGain: 1,
			Reward:             "补丁猎犬吐出了被标记过的差异块。",
			Lore:               combatLoreIDForNode(9),
		}
	case 11:
		return &Enemy{
			Name:             "Red-Black Priest",
			Bits:             []int{34, 9, 6, 2, 13, 10},
			Tier:             2,
			Elite:            true,
			SortPunishAttack: 3,
			Reward:           "红黑祭司的防篡改脚本已经被你拆解。",
			Lore:             combatLoreIDForNode(11),
		}
	case 13:
		return &Enemy{
			Name:               "Dead Sector",
			Bits:               []int{35, 10, 5, 3, 14, 11},
			Tier:               3,
			Elite:              true,
			InjectionCount:     1,
			UnsortedAttackGain: 2,
			Reward:             "坏扇区深处吐出了一段仍在闪烁的恢复代码。",
			Lore:               combatLoreIDForNode(13),
		}
	case 15:
		return &Enemy{
			Name:             "ROOT::Sentinel",
			Bits:             []int{44, 12, 7, 2, 16, 12},
			Boss:             true,
			Elite:            true,
			Tier:             4,
			LockedSlots:      2,
			SortPunishAttack: 2,
			InjectionCount:   2,
			Reward:           "Root 根目录的写入权限暂时向你敞开。",
			Lore:             combatLoreIDForNode(15),
		}
	default:
		base := 20 + rng.Intn(10)
		return &Enemy{
			Name:   node.Name,
			Bits:   []int{base, 5 + rng.Intn(4), 2 + rng.Intn(3), 2 + rng.Intn(2), 7 + rng.Intn(6), 4 + rng.Intn(6)},
			Tier:   1,
			Reward: "你从这个节点剥离出了一段还能继续执行的脚本。",
			Lore:   "",
		}
	}
}

func applyDifficultyScaling(rng *rand.Rand, node *Node, enemy *Enemy) {
	if enemy == nil || node == nil {
		return
	}

	protection := node.Index <= 6
	baseHP := 18
	if len(enemy.Bits) > 0 && enemy.Bits[0] > 0 {
		baseHP = enemy.Bits[0]
	}
	difficulty := int(math.Round(float64(baseHP) * (1 + float64(node.Index)*0.15)))
	if difficulty < baseHP {
		difficulty = baseHP
	}

	bitsLen := dynamicBitsLength(node.Index)
	if protection {
		bitsLen = 4
	}
	bits := make([]int, bitsLen)
	bits[0] = difficulty
	bits[1] = max(2, scaledEnemyStat(enemy.Bits, 1, node.Index, 0.35))
	bits[2] = max(0, scaledEnemyStat(enemy.Bits, 2, node.Index, 0.22))
	bits[3] = 2 + rng.Intn(2)
	if bitsLen > 4 {
		bits[4] = max(1, scaledEnemyStat(enemy.Bits, 4, node.Index, 0.3))
	}
	if bitsLen > 5 {
		bits[5] = max(0, scaledEnemyStat(enemy.Bits, 5, node.Index, 0.26))
	}
	for i := 6; i < bitsLen; i++ {
		bits[i] = max(0, 2+rng.Intn(5)+node.Index/2)
	}

	if protection {
		enemy.LockedSlots = 0
		enemy.SortPunishAttack = 0
		enemy.UnsortedAttackGain = 0
		enemy.InjectionCount = 0
		enemy.InjectionOnly = false
	}

	switch {
	case node.Index <= 5:
		enemy.SimpleAttackOnly = true
		enemy.ObfuscatePlayer = false
		enemy.DoSLockHand = false
	case node.Index <= 10:
		enemy.SimpleAttackOnly = false
		enemy.ObfuscatePlayer = true
		enemy.DoSLockHand = false
	default:
		enemy.SimpleAttackOnly = false
		enemy.ObfuscatePlayer = true
		enemy.DoSLockHand = true
	}

	enemy.IgnoreLoadPenalty = protection
	enemy.DifficultyScore = difficulty
	enemy.Bits = bits
}

func dynamicBitsLength(nodeIndex int) int {
	if nodeIndex <= 1 {
		return 4
	}
	growth := int(math.Round(float64(nodeIndex-1) * 6.0 / 14.0))
	return min(10, max(4, 4+growth))
}

func scaledEnemyStat(bits []int, index int, nodeIndex int, scale float64) int {
	base := 1 + nodeIndex/2
	if index < len(bits) {
		base = bits[index]
	}
	return int(math.Round(float64(base) * (1 + float64(nodeIndex)*scale/10.0)))
}

func advancedSupplyCandidate(g *Game, exclude []CardID) (CardID, bool) {
	if g == nil {
		return "", false
	}
	pool := g.activeCardsMatching(func(def CardDef) bool {
		return def.Complexity == "O(n log n)" && !def.Unplayable && def.Source != "curse"
	})
	pool = filterCandidates(pool, func(card CardID) bool {
		return !containsCardID(exclude, card)
	})
	if len(pool) == 0 {
		for id, def := range allCards {
			if def.Complexity != "O(n log n)" || def.Unplayable || def.Source == "curse" || !def.Enabled || containsCardID(exclude, id) {
				continue
			}
			pool = append(pool, id)
		}
	}
	if len(pool) == 0 {
		return "", false
	}
	shuffleCards(g.rng, pool)
	return pool[0], true
}

func starterProtectionCards() []CardID {
	return []CardID{CardReverse, CardHeap, CardNullPointer}
}

func (b *Battle) introProtectionActive() bool {
	return b != nil && b.enemy != nil && b.enemy.IgnoreLoadPenalty
}

func (b *Battle) isHandLocked(index int) bool {
	if index < 0 || index >= len(b.state.HandLocks) {
		return false
	}
	return b.state.HandLocks[index]
}

func (b *Battle) clearHandLocks() {
	if len(b.state.HandLocks) == 0 {
		return
	}
	for i := range b.state.HandLocks {
		b.state.HandLocks[i] = false
	}
}

func (b *Battle) removeHandCard(index int) CardID {
	card := b.state.Hand[index]
	b.state.Hand = append(b.state.Hand[:index], b.state.Hand[index+1:]...)
	if index < len(b.state.HandLocks) {
		b.state.HandLocks = append(b.state.HandLocks[:index], b.state.HandLocks[index+1:]...)
	}
	return card
}

func (b *Battle) flipRandomPlayerBit() string {
	if b == nil || b.game == nil || b.game.player == nil {
		return ""
	}
	type playerBit struct {
		label string
		get   func() int
		set   func(int)
		max   func() int
	}
	bits := []playerBit{
		{label: "HP", get: func() int { return b.game.player.HP }, set: func(v int) { b.game.player.HP = max(1, min(b.game.player.MaxHP, v)) }, max: func() int { return b.game.player.MaxHP }},
		{label: "MP", get: func() int { return b.game.player.MP }, set: func(v int) { b.game.player.MP = max(0, min(b.game.player.EffectiveMaxMP(), v)) }, max: func() int { return b.game.player.EffectiveMaxMP() }},
		{label: "Load", get: func() int { return b.game.player.Load }, set: func(v int) { b.game.player.Load = max(0, v) }, max: func() int { return 99 }},
		{label: "Pointer", get: func() int { return b.game.player.PointerCharges }, set: func(v int) { b.game.player.PointerCharges = max(0, v) }, max: func() int { return 9 }},
	}
	pick := bits[b.game.rng.Intn(len(bits))]
	before := pick.get()
	after := before ^ 1
	if after > pick.max() {
		after = before
	}
	pick.set(after)
	return fmt.Sprintf("代码混淆翻转了你的 %s：%d -> %d", pick.label, before, pick.get())
}

func (b *Battle) lockRandomHandCard() string {
	if b == nil || len(b.state.Hand) == 0 {
		return ""
	}
	available := make([]int, 0, len(b.state.Hand))
	for i := range b.state.Hand {
		available = append(available, i)
	}
	if len(available) == 0 {
		return ""
	}
	index := available[b.game.rng.Intn(len(available))]
	if len(b.state.HandLocks) < len(b.state.Hand) {
		missing := len(b.state.Hand) - len(b.state.HandLocks)
		b.state.HandLocks = append(b.state.HandLocks, make([]bool, missing)...)
	}
	b.state.HandLocks[index] = true
	return fmt.Sprintf("绯荤粺鎷掔粷鏈嶅姟閿佸畾浜嗘墜鐗岋細%s", allCards[b.state.Hand[index]].Name)
}




