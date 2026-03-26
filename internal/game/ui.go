package game

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

type rewardOption struct {
	label string
	desc  string
	apply func(*Game)
}

func (g *Game) Run() {
	g.showBoot()
	for !g.quit && g.player.HP > 0 && !g.won {
		g.renderWorld()
		input := strings.ToLower(g.readLine("navigate [l/r/b], deck [d], status [s], help [h], quit [q] > "))
		switch input {
		case "l", "r", "b":
			node, err := g.world.Move(input)
			if err != nil {
				g.pause(err.Error())
				continue
			}
			g.resolveNode(node)
		case "d":
			g.renderDeck()
		case "s":
			g.renderStatus()
		case "h":
			g.renderMetaHelp()
		case "q":
			g.quit = true
		default:
			g.pause("unknown command")
		}
	}
	g.renderEnding()
}

func (g *Game) showBoot() {
	clearScreen(g.out)
	lines := []string{
		"Checking RAM .......... OK",
		"Scanning floppy relics  OK",
		"Searching for Kernel .. FOUND",
		"Attaching rogue terminal",
		"Bypassing sentinel logs",
		"Injecting operator: codebreaker",
	}
	fmt.Fprintln(g.out, "CODE BREAKER :: THE MATRIX ROGUELIKE")
	fmt.Fprintln(g.out)
	for _, line := range lines {
		fmt.Fprintf(g.out, "%s\n", line)
		time.Sleep(140 * time.Millisecond)
	}
	fmt.Fprintln(g.out)
	fmt.Fprintln(g.out, "You are a real-world code infiltrator entering The Kernel.")
	fmt.Fprintln(g.out, "Every enemy is a live memory slice. Every card is an algorithm script.")
	fmt.Fprintln(g.out)
	g.readLine("press enter to attach > ")

	clearScreen(g.out)
	fmt.Fprintln(g.out, "Traversal Protocol")
	fmt.Fprintln(g.out, "1. preorder  -> clear parents before children")
	fmt.Fprintln(g.out, "2. postorder -> clear leaves before parents")
	switch g.readLine("select [1-2] > ") {
	case "2":
		g.world.SetProtocol("postorder")
	default:
		g.world.SetProtocol("preorder")
	}
}

func (g *Game) renderWorld() {
	clearScreen(g.out)
	fmt.Fprintln(g.out, matrixBanner(g.rng.Intn))
	fmt.Fprintln(g.out, "THE KERNEL :: TREE DESCENT")
	fmt.Fprintf(g.out, "HP %d/%d | MP %d/%d | Pointer %d | Completion %s\n",
		g.player.HP, g.player.MaxHP, g.player.MP, g.player.EffectiveMaxMP(), g.player.PointerCharges, g.world.Completion())
	target := g.world.ProtocolTarget()
	targetLabel := "complete"
	if target != nil {
		targetLabel = target.Name
	}
	fmt.Fprintf(g.out, "Protocol %s | Sync target %s\n", strings.ToUpper(g.world.Protocol), targetLabel)
	fmt.Fprintf(g.out, "Path %s\n", g.world.PathString())
	fmt.Fprintln(g.out)
	fmt.Fprintln(g.out, g.treeDiagram())
	node := g.world.CurrentNode()
	fmt.Fprintf(g.out, "Current Node %s [%s]\n", node.Name, strings.ToUpper(string(node.Kind)))
	fmt.Fprintf(g.out, "Balance %d", node.Balance)
	if node.Unstable() {
		fmt.Fprint(g.out, "  <- unstable")
	}
	fmt.Fprintln(g.out)
	fmt.Fprintln(g.out, node.Description)
	fmt.Fprintln(g.out)
	fmt.Fprintf(g.out, "Available moves %s\n", g.availableMoveString())
	fmt.Fprintln(g.out, "Traversal note: follow the protocol target to gain a sync reward.")
}

func (g *Game) treeDiagram() string {
	rows := [][]int{
		{1},
		{2, 3},
		{4, 5, 6, 7},
		{8, 9, 10, 11, 12, 13, 14, 15},
	}
	indents := []string{
		"                                   ",
		"                       ",
		"             ",
		"",
	}
	gaps := []string{
		"",
		"                  ",
		"        ",
		"  ",
	}
	var lines []string
	for rowIndex, row := range rows {
		var parts []string
		for _, index := range row {
			parts = append(parts, g.formatNode(index))
		}
		lines = append(lines, indents[rowIndex]+strings.Join(parts, gaps[rowIndex]))
	}
	return strings.Join(lines, "\n")
}

func (g *Game) formatNode(index int) string {
	node := g.world.NodeAt(index)
	if node == nil {
		return "[         ]"
	}
	state := " "
	if node.Cleared {
		state = "x"
	}
	if index == g.world.Current {
		state = ">"
	}
	name := node.Name
	if len(name) > 7 {
		name = name[:7]
	}
	if node.Unstable() && len(name) > 0 {
		name = "!" + name[:min(6, len(name))]
	}
	return fmt.Sprintf("[%s%-8s]", state, name)
}

func (g *Game) availableMoveString() string {
	moves := g.world.AvailableMoves()
	if len(moves) == 0 {
		return "(none)"
	}
	order := []string{"l", "r", "b"}
	parts := make([]string, 0, len(moves))
	for _, key := range order {
		index, ok := moves[key]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, g.world.Nodes[index].Name))
	}
	return strings.Join(parts, " | ")
}

func (g *Game) resolveNode(node *Node) {
	if node.Cleared {
		g.pause("this node is already stable")
		return
	}
	guided := false
	if target := g.world.ProtocolTarget(); target != nil && target.Index == node.Index {
		guided = true
	}
	if node.Unstable() {
		g.triggerGlitch(node)
		if g.player.HP <= 0 {
			return
		}
	}

	switch node.Kind {
	case NodeRest:
		heal := 10
		g.player.HP = min(g.player.MaxHP, g.player.HP+heal)
		g.player.MP = g.player.EffectiveMaxMP()
		node.Cleared = true
		g.pause(fmt.Sprintf("%s ran a local defrag. HP +%d, MP restored.", node.Name, heal))
	case NodeTreasure:
		node.Cleared = true
		g.resolveTreasure(node)
	case NodeCombat, NodeBoss:
		enemy := buildEnemy(g.rng, node)
		if g.runCombat(enemy) {
			node.Cleared = true
			g.resolveCombatRewards(enemy)
			if node.Kind == NodeBoss {
				g.won = true
			}
		}
	}
	if guided && node.Cleared && !g.won {
		g.protocolSync(node)
	}
}

func (g *Game) protocolSync(node *Node) {
	clearScreen(g.out)
	fmt.Fprintf(g.out, "TRAVERSAL SYNC :: %s matched %s protocol\n\n", node.Name, strings.ToUpper(g.world.Protocol))
	switch g.rng.Intn(3) {
	case 0:
		g.player.HP = min(g.player.MaxHP, g.player.HP+3)
		fmt.Fprintln(g.out, "Kernel alignment restored 3 HP.")
	case 1:
		g.player.PointerCharges++
		fmt.Fprintln(g.out, "A clean pointer path opened. Gain 1 Pointer Charge.")
	default:
		if g.removeCardFromDeck(CardNoop) {
			fmt.Fprintln(g.out, "Protocol cleanup purged one noop.log from the deck.")
		} else {
			g.player.Deck = append(g.player.Deck, CardMemoize)
			fmt.Fprintln(g.out, "Protocol cache gifted a memoize.lua script.")
		}
	}
	g.readLine("press enter > ")
}

func (g *Game) triggerGlitch(node *Node) {
	clearScreen(g.out)
	fmt.Fprintln(g.out, "UNBALANCED NODE DETECTED")
	fmt.Fprintf(g.out, "Balance Factor = %d on %s\n\n", node.Balance, node.Name)
	switch g.rng.Intn(4) {
	case 0:
		g.player.PointerCharges++
		fmt.Fprintln(g.out, "You captured a dangling pointer. Boss fights now have one more hijack charge.")
	case 1:
		g.player.HP = max(1, g.player.HP-3)
		g.player.Deck = append(g.player.Deck, CardNoop)
		fmt.Fprintln(g.out, "A leak polluted the deck. Lose 3 HP and gain one noop.log.")
	case 2:
		g.player.MP = g.player.EffectiveMaxMP()
		fmt.Fprintln(g.out, "A cache surge preheated your memory pool back to full MP.")
	default:
		fragment := "Glitch fragment: 'He said bugs were just messages from the deeper machine.'"
		g.player.AddLore(fragment)
		fmt.Fprintln(g.out, "A ghost comment surfaced from the unstable branch.")
		fmt.Fprintln(g.out, fragment)
	}
	g.readLine("press enter > ")
}

func (g *Game) resolveTreasure(node *Node) {
	clearScreen(g.out)
	fmt.Fprintf(g.out, "%s\n", strings.ToUpper(node.Name))
	fmt.Fprintln(g.out, "Relics and cracked source fragments float in a green afterglow.")
	rewards := g.rewardBundle(node)
	for i, reward := range rewards {
		fmt.Fprintf(g.out, "%d. %s\n", i+1, reward.label)
		fmt.Fprintf(g.out, "   %s\n", reward.desc)
	}
	fmt.Fprintln(g.out, "0. skip")
	input := g.readLine("claim reward > ")
	switch input {
	case "1", "2", "3":
		index := int(input[0] - '1')
		rewards[index].apply(g)
		g.pause("reward injected into runtime")
	default:
		g.pause("you leave the relics where the old code can keep whispering")
	}
}

func (g *Game) resolveCombatRewards(enemy *Enemy) {
	g.player.HP = min(g.player.MaxHP, g.player.HP+g.player.GCHeal)
	if enemy.Lore != "" {
		g.player.AddLore(enemy.Lore)
	}
	clearScreen(g.out)
	fmt.Fprintf(g.out, "%s\n\n", enemy.Reward)
	fmt.Fprintf(g.out, "Garbage Collection complete: HP +%d.\n", g.player.GCHeal)
	if enemy.Lore != "" {
		fmt.Fprintln(g.out, "Recovered lore fragment:")
		fmt.Fprintln(g.out, enemy.Lore)
		fmt.Fprintln(g.out)
	}
	fmt.Fprintln(g.out, "Pick one new script from the dump.")
	options := g.randomRewardCards(3)
	for i, card := range options {
		def := allCards[card]
		fmt.Fprintf(g.out, "%d. %s [%d MP, %s]\n", i+1, def.Name, def.Cost, def.Complexity)
		fmt.Fprintf(g.out, "   %s\n", def.Text)
	}
	fmt.Fprintln(g.out, "0. skip")
	input := g.readLine("select script > ")
	if input == "1" || input == "2" || input == "3" {
		index := int(input[0] - '1')
		g.player.Deck = append(g.player.Deck, options[index])
	}
	g.purgeCardFlow()
}

func (g *Game) purgeCardFlow() {
	clearScreen(g.out)
	fmt.Fprintln(g.out, "GARBAGE COLLECTION")
	fmt.Fprintln(g.out, "Delete one card from the deck, or keep everything.")
	for i, card := range g.player.Deck {
		def := allCards[card]
		fmt.Fprintf(g.out, "%d. %s\n", i+1, def.Name)
	}
	fmt.Fprintln(g.out, "0. keep all")
	input := g.readLine("purge card > ")
	if input == "0" || input == "" {
		return
	}
	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(g.player.Deck) {
		g.pause("invalid purge selection")
		return
	}
	index--
	removed := g.player.Deck[index]
	g.player.Deck = append(g.player.Deck[:index], g.player.Deck[index+1:]...)
	g.pause(fmt.Sprintf("%s removed from deck", allCards[removed].Name))
}

func (g *Game) removeCardFromDeck(card CardID) bool {
	for i, candidate := range g.player.Deck {
		if candidate != card {
			continue
		}
		g.player.Deck = append(g.player.Deck[:i], g.player.Deck[i+1:]...)
		return true
	}
	return false
}

func (g *Game) renderDeck() {
	clearScreen(g.out)
	fmt.Fprintln(g.out, "DECK DUMP")
	counts := map[CardID]int{}
	for _, card := range g.player.Deck {
		counts[card]++
	}
	var ids []string
	for id := range counts {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, raw := range ids {
		def := allCards[CardID(raw)]
		fmt.Fprintf(g.out, "%s x%d [%d MP, %s]\n", def.Name, counts[CardID(raw)], def.Cost, def.Complexity)
		fmt.Fprintf(g.out, "  %s\n", def.Text)
	}
	g.readLine("press enter > ")
}

func (g *Game) renderStatus() {
	clearScreen(g.out)
	fmt.Fprintln(g.out, "OPERATOR STATUS")
	fmt.Fprintf(g.out, "HP %d/%d\n", g.player.HP, g.player.MaxHP)
	fmt.Fprintf(g.out, "MP %d/%d\n", g.player.MP, g.player.EffectiveMaxMP())
	fmt.Fprintf(g.out, "Pointer Charges %d\n", g.player.PointerCharges)
	fmt.Fprintf(g.out, "Draw Bonus %d\n", g.player.DrawBonus)
	fmt.Fprintf(g.out, "Stack Trigger %d\n", g.player.StackTrigger)
	fmt.Fprintf(g.out, "Stack Echo Damage %d\n", g.player.StackEchoDamage)
	fmt.Fprintf(g.out, "GC Heal %d\n", g.player.GCHeal)
	if len(g.player.Artifacts) == 0 {
		fmt.Fprintln(g.out, "Artifacts: none")
	} else {
		fmt.Fprintln(g.out, "Artifacts:")
		for _, artifact := range g.player.Artifacts {
			fmt.Fprintf(g.out, "  - %s: %s\n", artifact.Name, artifact.Desc)
		}
	}
	if len(g.player.Lore) == 0 {
		fmt.Fprintln(g.out, "Lore: none")
	} else {
		fmt.Fprintln(g.out, "Lore fragments:")
		for i, fragment := range g.player.Lore {
			fmt.Fprintf(g.out, "  %d. %s\n", i+1, fragment)
		}
	}
	g.readLine("press enter > ")
}

func (g *Game) renderMetaHelp() {
	clearScreen(g.out)
	fmt.Fprintln(g.out, "HELP")
	fmt.Fprintln(g.out, "1. Enemy bits decode to integrity, attack, armor, countdown, script, and entropy.")
	fmt.Fprintln(g.out, "2. Sorting and reversing change the enemy's live state because stats move with slots.")
	fmt.Fprintln(g.out, "3. Playing three scripts in one turn triggers Stack Overflow and replays the top two.")
	fmt.Fprintln(g.out, "4. Unstable nodes trigger glitch events.")
	fmt.Fprintln(g.out, "5. Boss fights allow `p` when you have a leaked Root pointer.")
	fmt.Fprintln(g.out, "6. Follow the traversal protocol target to gain a sync reward.")
	g.readLine("press enter > ")
}

func (g *Game) renderEnding() {
	clearScreen(g.out)
	switch {
	case g.won:
		fmt.Fprintln(g.out, "ROOT REWRITTEN")
		fmt.Fprintln(g.out)
		if len(g.player.Lore) > 0 {
			fmt.Fprintln(g.out, "Recovered fragments:")
			for _, fragment := range g.player.Lore {
				fmt.Fprintln(g.out, fragment)
			}
			fmt.Fprintln(g.out)
		}
		fmt.Fprintln(g.out, "The green rain slows. Root accepts your write access.")
		fmt.Fprintln(g.out, "\"If you can read this, you already own the machine.\"")
	case g.player.HP <= 0:
		fmt.Fprintln(g.out, "SESSION TERMINATED")
		fmt.Fprintln(g.out)
		fmt.Fprintln(g.out, "The sentinel net severs your shell and The Kernel closes around the wound.")
	default:
		fmt.Fprintln(g.out, "DISCONNECTED")
	}
}

func (b *Battle) render(message string) {
	clearScreen(b.game.out)
	fmt.Fprintln(b.game.out, matrixBanner(b.game.rng.Intn))
	fmt.Fprintf(b.game.out, "COMBAT :: %s\n", b.enemy.Name)
	fmt.Fprintf(b.game.out, "HP %d/%d | MP %d/%d | Pointer %d | Turn %d\n",
		b.game.player.HP, b.game.player.MaxHP, b.game.player.MP, b.game.player.EffectiveMaxMP(), b.game.player.PointerCharges, b.turn)
	fmt.Fprintln(b.game.out)
	fmt.Fprintf(b.game.out, "Enemy bits   %v\n", b.enemy.Bits)
	fmt.Fprintf(b.game.out, "Decoded      integrity=%d attack=%d armor=%d countdown=%d script=%d entropy=%d\n",
		b.enemyIntegrity(), b.enemyAttack(), b.enemyArmor(), b.enemyCountdown(), b.enemyScript(), b.enemyEntropy())
	fmt.Fprintf(b.game.out, "Intent       %s\n", b.intentString())
	fmt.Fprintf(b.game.out, "Turn stack   %s\n", b.stackString())
	fmt.Fprintln(b.game.out)
	fmt.Fprintln(b.game.out, "Hand")
	for i, card := range b.state.Hand {
		def := allCards[card]
		fmt.Fprintf(b.game.out, "%d. %s [%d MP, %s]\n", i+1, def.Name, def.Cost, def.Complexity)
		fmt.Fprintf(b.game.out, "   %s\n", def.Text)
	}
	fmt.Fprintln(b.game.out)
	fmt.Fprintln(b.game.out, "Logs")
	for _, line := range b.logs {
		fmt.Fprintf(b.game.out, "> %s\n", line)
	}
	if message != "" {
		fmt.Fprintf(b.game.out, "\n%s\n", message)
	}
}

func (b *Battle) renderHelp() {
	clearScreen(b.game.out)
	fmt.Fprintln(b.game.out, "BATTLE HELP")
	fmt.Fprintln(b.game.out, "1. Type a hand index to execute a script.")
	fmt.Fprintln(b.game.out, "2. You may chain multiple cards before ending the turn with `e`.")
	fmt.Fprintln(b.game.out, "3. Sort scripts pair well with binary_search.go and null_pointer.exe.")
	fmt.Fprintln(b.game.out, "4. backtrack.ts restores the previous turn snapshot.")
	fmt.Fprintln(b.game.out, "5. memoize.lua copies useful scripts out of discard.")
	fmt.Fprintln(b.game.out, "6. `p` is reserved for boss fights with Pointer Charges.")
	b.game.readLine("press enter > ")
}

func (b *Battle) animate(title string, frames [][]int) {
	if b.game.headless {
		return
	}
	for _, frame := range frames {
		clearScreen(b.game.out)
		fmt.Fprintf(b.game.out, "EXECUTE :: %s\n\n", title)
		fmt.Fprintln(b.game.out, matrixBanner(b.game.rng.Intn))
		fmt.Fprintf(b.game.out, "bits => %v\n", frame)
		fmt.Fprintf(b.game.out, "decoded => integrity=%d attack=%d armor=%d countdown=%d script=%d entropy=%d\n",
			slotValue(frame, 0), slotValue(frame, 1), slotValue(frame, 2), slotValue(frame, 3), slotValue(frame, 4), slotValue(frame, 5))
		time.Sleep(85 * time.Millisecond)
	}
}

func (b *Battle) intentString() string {
	if b.enemy.InjectionOnly && b.enemyCountdown() <= 0 {
		return fmt.Sprintf("即将发动反向注入，写入 %d 个污染 bit。", max(1, b.enemy.InjectionCount))
	}
	if b.enemyCountdown() <= 0 {
		return fmt.Sprintf("敌方即将执行主攻击，预计造成 %d 点伤害。", b.enemyAttack()+b.enemyScript()/4)
	}
	if b.enemy.InjectionCount > 0 {
		return fmt.Sprintf("敌人攻击时会额外注入 %d 个污染 bit。", b.enemy.InjectionCount)
	}
	if b.enemyEntropy() >= 10 {
		return "高熵状态：小心爆发伤害或自我修复。"
	}
	if b.enemyArmor() >= 8 {
		return "敌方护甲很高，优先考虑穿透、削甲或重排脚本位。"
	}
	return fmt.Sprintf("敌方正在蓄力攻击，还需 %d 回合。", b.enemyCountdown())
}

func (b *Battle) stackString() string {
	if len(b.stack) == 0 {
		return "(empty)"
	}
	parts := make([]string, len(b.stack))
	for i, card := range b.stack {
		parts[i] = string(card)
	}
	return strings.Join(parts, " -> ")
}

func (b *Battle) pushLog(line string) {
	b.logs = append(b.logs, line)
	if len(b.logs) > 8 {
		b.logs = b.logs[len(b.logs)-8:]
	}
}

func (g *Game) rewardBundle(node *Node) []rewardOption {
	artifacts := []rewardOption{
		{
			label: "Leaked Root Pointer",
			desc:  "获得 1 次额外的 Boss 指针劫持充能。",
			apply: func(g *Game) {
				g.player.PointerCharges++
				g.player.AddArtifact(Artifact{Name: "Leaked Root Pointer", Desc: "获得 1 次额外的 Boss 指针劫持充能。"})
			},
		},
		{
			label: "Green CRT Filter",
			desc:  "战斗中的 MP 上限 +1。",
			apply: func(g *Game) {
				g.player.StartMPBonus++
				g.player.AddArtifact(Artifact{Name: "Green CRT Filter", Desc: "战斗中的 MP 上限 +1。"})
			},
		},
		{
			label: "Threaded Stack",
			desc:  "栈溢出回放的额外伤害 +2。",
			apply: func(g *Game) {
				g.player.StackEchoDamage += 2
				g.player.AddArtifact(Artifact{Name: "Threaded Stack", Desc: "栈溢出回放的额外伤害 +2。"})
			},
		},
		{
			label: "Defrag Patch",
			desc:  "最大生命值 +6，并立刻恢复 6 点生命。",
			apply: func(g *Game) {
				g.player.MaxHP += 6
				g.player.HP = min(g.player.MaxHP, g.player.HP+6)
				g.player.AddArtifact(Artifact{Name: "Defrag Patch", Desc: "最大生命值 +6。"})
			},
		},
		{
			label: "Warm Cache Latch",
			desc:  "每回合开始时额外抽 1 张牌。",
			apply: func(g *Game) {
				g.player.DrawBonus++
				g.player.AddArtifact(Artifact{Name: "Warm Cache Latch", Desc: "每回合开始时额外抽 1 张牌。"})
			},
		},
		{
			label: "Compact Collector",
			desc:  "战斗结束后的垃圾回收额外恢复 2 点生命。",
			apply: func(g *Game) {
				g.player.GCHeal += 2
				g.player.AddArtifact(Artifact{Name: "Compact Collector", Desc: "战斗结束后的垃圾回收额外恢复 2 点生命。"})
			},
		},
	}

	cardChoices := g.randomRewardCards(2)
	if len(cardChoices) > 1 && g.rng.Intn(100) < 20 {
		cardChoices[1] = CardDefrag
	}

	entry, hasEntry := storyEntry(treasureLoreIDForNode(node.Index))
	lore := rewardOption{
		label: "档案碎片",
		desc:  fmt.Sprintf("恢复 %s 中遗失的一段存档。", node.Name),
		apply: func(g *Game) {
			if hasEntry {
				g.player.AddLoreEntry(entry)
				return
			}
			g.player.AddLore(fmt.Sprintf("档案碎片 %s：这台机器仍记得每一个停留过久的操作员。", node.Name))
		},
	}
	if hasEntry {
		lore.label = entry.Title
		lore.desc = entry.Text
	}

	return []rewardOption{
		artifacts[g.rng.Intn(len(artifacts))],
		{
			label: allCards[cardChoices[0]].Name,
			desc:  allCards[cardChoices[0]].Text,
			apply: func(g *Game) {
				g.player.Deck = append(g.player.Deck, cardChoices[0])
			},
		},
		lore,
	}
}
func (g *Game) randomRewardCards(n int) []CardID {
	return g.drawRewardCardsFromActivePool(n)
}

func (g *Game) readLine(prompt string) string {
	fmt.Fprint(g.out, prompt)
	line, _ := g.reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func (g *Game) pause(message string) {
	clearScreen(g.out)
	fmt.Fprintln(g.out, message)
	g.readLine("press enter > ")
}

func clearScreen(out io.Writer) {
	fmt.Fprint(out, "\033[H\033[2J")
}

func matrixBanner(intn func(int) int) string {
	parts := make([]string, 3)
	for i := range parts {
		var row strings.Builder
		for j := 0; j < 42; j++ {
			switch intn(5) {
			case 0:
				row.WriteByte('1')
			case 1:
				row.WriteByte('0')
			default:
				row.WriteByte('.')
			}
		}
		parts[i] = row.String()
	}
	return strings.Join(parts, "\n")
}

func signalBell(out io.Writer) {
	fmt.Fprint(out, "\a")
}

func slotValue(frame []int, index int) int {
	if index >= len(frame) {
		return 0
	}
	return max(0, frame[index])
}


