package game

import (
	"math/rand"
	"testing"
)

func TestComplexityLoadMatchesRules(t *testing.T) {
	cases := map[string]int{
		"O(1)":            0,
		"O(log n)":        0,
		"O(n)":            1,
		"O(n log n)":      1,
		"O(n*k)":          2,
		"O(n^2)":          2,
		"O(branch^depth)": 2,
	}
	for complexity, want := range cases {
		if got := complexityLoad(complexity); got != want {
			t.Fatalf("complexity %s: want %d, got %d", complexity, want, got)
		}
	}
}

func TestNormalRewardOffersOutputControlAndResource(t *testing.T) {
	game := NewHeadlessGame(rand.New(rand.NewSource(11)))
	rewards := game.randomRewardCardsForEnemy(&Enemy{Name: "normal"}, 3)
	if len(rewards) != 3 {
		t.Fatalf("expected 3 rewards, got %d", len(rewards))
	}
	if !containsRewardRole(rewards, func(def CardDef) bool { return rewardHasAnyTag(def, "attack", "finisher") }) {
		t.Fatalf("expected an output card in %v", rewards)
	}
	if !containsRewardRole(rewards, func(def CardDef) bool {
		return rewardHasAnyTag(def, "control", "manipulation", "disrupt", "sort", "precision")
	}) {
		t.Fatalf("expected a control card in %v", rewards)
	}
	if !containsRewardRole(rewards, func(def CardDef) bool {
		return rewardHasAnyTag(def, "resource", "maintenance", "draw", "cleanup", "heal", "utility")
	}) {
		t.Fatalf("expected a resource card in %v", rewards)
	}
}

func TestEliteRewardOffersOneRareAndTwoCommon(t *testing.T) {
	game := NewHeadlessGame(rand.New(rand.NewSource(17)))
	rewards := game.randomRewardCardsForEnemy(&Enemy{Name: "elite", Elite: true}, 3)
	if len(rewards) != 3 {
		t.Fatalf("expected 3 rewards, got %d", len(rewards))
	}
	rare := 0
	common := 0
	for _, card := range rewards {
		if allCards[card].Source == "rare" {
			rare++
		} else {
			common++
		}
	}
	if rare != 1 || common != 2 {
		t.Fatalf("expected 1 rare and 2 common rewards, got %v", rewards)
	}
}

func TestBossRewardOffersThreeRareCards(t *testing.T) {
	game := NewHeadlessGame(rand.New(rand.NewSource(23)))
	rewards := game.randomRewardCardsForEnemy(&Enemy{Name: "boss", Boss: true, Elite: true}, 3)
	if len(rewards) != 3 {
		t.Fatalf("expected 3 rewards, got %d", len(rewards))
	}
	for _, card := range rewards {
		if allCards[card].Source != "rare" {
			t.Fatalf("expected boss rewards to all be rare, got %v", rewards)
		}
	}
}

func TestNoopDoesNotCountTowardStackOverflow(t *testing.T) {
	game := NewHeadlessGame(rand.New(rand.NewSource(29)))
	game.player.MP = 20
	enemy := &Enemy{Name: "dummy", Bits: []int{40, 3, 1, 2}}
	battle := NewBattle(game, enemy)
	defer battle.Close()
	battle.state.Hand = []CardID{CardReverse, CardHeap, CardBinary, CardNoop}
	battle.state.HandLocks = []bool{false, false, false, false}
	battle.turn = 1
	battle.saveSnapshot()

	for i := 0; i < 4; i++ {
		if err := battle.playHand(0); err != nil {
			t.Fatalf("play %d failed: %v", i, err)
		}
	}
	if got := len(battle.stack); got != 3 {
		t.Fatalf("expected noop.log to be ignored by stack overflow count, stack=%v", battle.stack)
	}
}

func containsRewardRole(cards []CardID, match func(CardDef) bool) bool {
	for _, card := range cards {
		if match(allCards[card]) {
			return true
		}
	}
	return false
}
