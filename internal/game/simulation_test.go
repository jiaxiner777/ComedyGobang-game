package game

import (
	"math/rand"
	"testing"
)

func TestRunBattleSimulation(t *testing.T) {
	report := runBattleSimulation(25, rand.New(rand.NewSource(7)))
	if report.Iterations != 25 {
		t.Fatalf("expected 25 iterations, got %d", report.Iterations)
	}
	if report.Wins < 0 || report.Wins > report.Iterations {
		t.Fatalf("wins out of range: %+v", report)
	}
	if report.WinRate < 0 || report.WinRate > 1 {
		t.Fatalf("invalid win rate: %+v", report)
	}
	if report.AvgRemainingHP < 0 {
		t.Fatalf("average hp should be non-negative: %+v", report)
	}
	if report.AvgBattleTurns <= 0 {
		t.Fatalf("average turns should be positive: %+v", report)
	}
}

func TestSimulationCombatNodesOnlyContainCombat(t *testing.T) {
	nodes := simulationCombatNodes()
	if len(nodes) == 0 {
		t.Fatal("expected combat nodes")
	}
	for _, node := range nodes {
		if node.Kind != NodeCombat && node.Kind != NodeBoss {
			t.Fatalf("unexpected node kind in simulation pool: %+v", node)
		}
	}
}
