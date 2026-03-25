package game

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

type BattleSimulationReport struct {
	Iterations      int
	Wins            int
	WinRate         float64
	AvgRemainingHP  float64
	AvgBattleTurns  float64
	TimedOutBattles int
}

func SimulateBattle() BattleSimulationReport {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	report := runBattleSimulation(1000, rng)
	fmt.Printf("玩家胜率: %.2f%%\n", report.WinRate*100)
	fmt.Printf("平均剩余血量: %.2f\n", report.AvgRemainingHP)
	fmt.Printf("平均战斗回合数: %.2f\n", report.AvgBattleTurns)
	return report
}

func runBattleSimulation(iterations int, rng *rand.Rand) BattleSimulationReport {
	if iterations <= 0 {
		return BattleSimulationReport{}
	}

	nodes := simulationCombatNodes()
	totalHP := 0
	totalTurns := 0
	wins := 0
	timeouts := 0

	for i := 0; i < iterations; i++ {
		seed := rng.Int63()
		runRNG := rand.New(rand.NewSource(seed))
		node := nodes[runRNG.Intn(len(nodes))]
		won, hp, turns, timedOut := simulateSingleBattle(runRNG, node)
		if won {
			wins++
		}
		if timedOut {
			timeouts++
		}
		totalHP += hp
		totalTurns += turns
	}

	return BattleSimulationReport{
		Iterations:      iterations,
		Wins:            wins,
		WinRate:         float64(wins) / float64(iterations),
		AvgRemainingHP:  float64(totalHP) / float64(iterations),
		AvgBattleTurns:  float64(totalTurns) / float64(iterations),
		TimedOutBattles: timeouts,
	}
}

func simulateSingleBattle(rng *rand.Rand, node *Node) (won bool, remainingHP int, turns int, timedOut bool) {
	game := NewHeadlessGame(rng)
	enemy := buildEnemy(game.rng, node)
	battle := NewBattle(game, enemy)
	defer battle.Close()

	game.player.MP = game.player.EffectiveMaxMP()
	game.player.Load = 0
	game.player.PendingLoad = 0
	battle.state.DrawPile = cloneCards(game.player.Deck)
	shuffleCards(game.rng, battle.state.DrawPile)
	battle.draw(5)
	battle.startTurn()

	const maxTurns = 80
	const maxActions = 400
	actions := 0

	for game.player.HP > 0 && battle.enemyIntegrity() > 0 {
		if battle.turn > maxTurns || actions > maxActions {
			return false, 0, max(1, battle.turn), true
		}

		actions += playRandomTurn(battle)
		if battle.enemyIntegrity() <= 0 || game.player.HP <= 0 {
			break
		}

		battle.enemyPhase()
		if battle.enemyIntegrity() <= 0 || game.player.HP <= 0 {
			break
		}

		battle.startTurn()
	}

	if game.player.HP <= 0 {
		return false, 0, max(1, battle.turn), false
	}
	return true, game.player.HP, max(1, battle.turn), false
}

func playRandomTurn(b *Battle) int {
	if b.turnLocked {
		return 0
	}

	actions := 0
	for {
		playable := randomPlayableIndexes(b)
		if len(playable) == 0 {
			return actions
		}

		shouldEnd := actions > 0 && b.game.rng.Intn(100) < 35
		if shouldEnd {
			return actions
		}

		index := playable[b.game.rng.Intn(len(playable))]
		if err := b.playHand(index); err != nil {
			return actions
		}
		actions++

		if b.enemyIntegrity() <= 0 || b.turnLocked {
			return actions
		}
	}
}

func randomPlayableIndexes(b *Battle) []int {
	indexes := make([]int, 0, len(b.state.Hand))
	for i, card := range b.state.Hand {
		def := allCards[card]
		if def.Unplayable || b.isHandLocked(i) || b.game.player.MP < def.Cost {
			continue
		}
		indexes = append(indexes, i)
	}
	return indexes
}

func simulationCombatNodes() []*Node {
	world := NewWorld(rand.New(rand.NewSource(1)))
	indexes := world.SortedIndexes()
	nodes := make([]*Node, 0, len(indexes))
	for _, index := range indexes {
		node := world.NodeAt(index)
		if node == nil {
			continue
		}
		if node.Kind != NodeCombat && node.Kind != NodeBoss {
			continue
		}
		nodes = append(nodes, &Node{
			Index: node.Index,
			Name:  node.Name,
			Kind:  node.Kind,
		})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Index < nodes[j].Index })
	return nodes
}
