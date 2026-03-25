package game

import (
	"math/rand"
	"testing"
)

func TestEarlyEnemyProtectionUsesShortBitsAndNoLoadPenalty(t *testing.T) {
	enemy := buildEnemy(rand.New(rand.NewSource(1)), &Node{Index: 2, Kind: NodeCombat, Name: "Trace"})
	if got := len(enemy.Bits); got != 4 {
		t.Fatalf("expected early protected enemy to have 4 bits, got %d", got)
	}
	if !enemy.IgnoreLoadPenalty {
		t.Fatal("expected early protected enemy to ignore load penalty")
	}
}

func TestLateEnemyScalingUnlocksLongerBitsAndDoS(t *testing.T) {
	enemy := buildEnemy(rand.New(rand.NewSource(2)), &Node{Index: 13, Kind: NodeCombat, Name: "Dead Sector"})
	if got := len(enemy.Bits); got < 8 {
		t.Fatalf("expected late enemy to have expanded bits, got %d", got)
	}
	if !enemy.DoSLockHand {
		t.Fatal("expected node 13 enemy to unlock DoS hand lock")
	}
	if !enemy.ObfuscatePlayer {
		t.Fatal("expected node 13 enemy to keep player-bit obfuscation")
	}
}

func TestLowDeckGuaranteesAdvancedSupplyCard(t *testing.T) {
	game := NewHeadlessGame(rand.New(rand.NewSource(3)))
	game.player.Deck = []CardID{CardReverse, CardHeap, CardNullPointer, CardBinary}
	rewards := game.randomRewardCardsForEnemy(nil, 3)
	if len(rewards) == 0 {
		t.Fatal("expected rewards to be generated")
	}
	found := false
	for _, card := range rewards {
		if allCards[card].Complexity == "O(n log n)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one O(n log n) supply card, got %v", rewards)
	}
}

func TestStarterProtectionCardsAreReusable(t *testing.T) {
	mustConfigureCardLibrary()
	for _, card := range starterProtectionCards() {
		if !allCards[card].Reusable {
			t.Fatalf("expected %s to be marked reusable", card)
		}
	}
}

